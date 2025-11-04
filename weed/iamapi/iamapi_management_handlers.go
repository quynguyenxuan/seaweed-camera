package iamapi

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	seaweedSts "github.com/seaweedfs/seaweedfs/weed/iam/sts"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/iam_pb"
	"github.com/seaweedfs/seaweedfs/weed/s3api/policy_engine"
	"github.com/seaweedfs/seaweedfs/weed/s3api/s3_constants"
	"github.com/seaweedfs/seaweedfs/weed/s3api/s3err"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	jwt "github.com/golang-jwt/jwt/v5"
	cred "github.com/seaweedfs/seaweedfs/weed/credential"
)

const (
	charsetUpper            = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charset                 = charsetUpper + "abcdefghijklmnopqrstuvwxyz/"
	policyDocumentVersion   = "2012-10-17"
	StatementActionAdmin    = "*"
	StatementActionWrite    = "Put*"
	StatementActionWriteAcp = "PutBucketAcl"
	StatementActionRead     = "Get*"
	StatementActionReadAcp  = "GetBucketAcl"
	StatementActionList     = "List*"
	StatementActionTagging  = "Tagging*"
	StatementActionDelete   = "DeleteBucket*"
)

var (
	seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
	policyDocuments = map[string]*policy_engine.PolicyDocument{}
	policyLock      = sync.RWMutex{}
)

func MapToStatementAction(action string) string {
	switch action {
	case StatementActionAdmin:
		return s3_constants.ACTION_ADMIN
	case StatementActionWrite:
		return s3_constants.ACTION_WRITE
	case StatementActionWriteAcp:
		return s3_constants.ACTION_WRITE_ACP
	case StatementActionRead:
		return s3_constants.ACTION_READ
	case StatementActionReadAcp:
		return s3_constants.ACTION_READ_ACP
	case StatementActionList:
		return s3_constants.ACTION_LIST
	case StatementActionTagging:
		return s3_constants.ACTION_TAGGING
	case StatementActionDelete:
		return s3_constants.ACTION_DELETE_BUCKET
	default:
		return ""
	}
}

func MapToIdentitiesAction(action string) string {
	switch action {
	case s3_constants.ACTION_ADMIN:
		return StatementActionAdmin
	case s3_constants.ACTION_WRITE:
		return StatementActionWrite
	case s3_constants.ACTION_WRITE_ACP:
		return StatementActionWriteAcp
	case s3_constants.ACTION_READ:
		return StatementActionRead
	case s3_constants.ACTION_READ_ACP:
		return StatementActionReadAcp
	case s3_constants.ACTION_LIST:
		return StatementActionList
	case s3_constants.ACTION_TAGGING:
		return StatementActionTagging
	case s3_constants.ACTION_DELETE_BUCKET:
		return StatementActionDelete
	default:
		return ""
	}
}

const (
	USER_DOES_NOT_EXIST = "the user with name %s cannot be found."
)

type Policies struct {
	Policies map[string]policy_engine.PolicyDocument `json:"policies"`
}

func Hash(s *string) string {
	h := sha1.New()
	h.Write([]byte(*s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// // QUYNGUYEN ADD
// func (iama *IamApiServer) GetActionsFromPolicy(policyName string) (actions []string, err error) {
// 	policies := Policies{}
// 	policyLock.Lock()
// 	defer policyLock.Unlock()
// 	err = iama.s3ApiConfig.GetPolicies(&policies)
// 	if err != nil {
// 		glog.V(1).Infof("Error getting policies: %v", err)
// 	}
// 	if policyDocument, exists := policies.Policies[policyName]; exists {
// 		actions, err = GetActions(&policyDocument)
// 		if err != nil {
// 			glog.V(1).Infof("Error getting actions: %v", err)
// 		}
// 	}
// 	return actions, err
// }

// func (iama *IamApiServer) GetActionsFromRoleArn(roleArn string) (actions []string, err error) {
// 	if roleArn == "" {
// 		return nil, errors.New("Role ARN is empty")
// 	}
// 	lastIndex := strings.LastIndex(roleArn, "/")
// 	policyName := ""
// 	if lastIndex != -1 {
// 		policyName = roleArn[lastIndex+1:]
// 		fmt.Println("Policy Name:", policyName)
// 	} else {
// 		fmt.Println("Invalid ARN format")
// 	}
// 	if policyName != "" {
// 		actions, err = iama.GetActionsFromPolicy(policyName)
// 	}
// 	return actions, err
// }

func (iama *IamApiServer) ListUsers(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp ListUsersResponse) {
	for _, ident := range s3cfg.Identities {
		resp.ListUsersResult.Users = append(resp.ListUsersResult.Users, &iam.User{UserName: &ident.Name})
	}
	return resp
}

func (iama *IamApiServer) ListAccessKeys(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp ListAccessKeysResponse) {
	status := iam.StatusTypeActive
	userName := values.Get("UserName")
	for _, ident := range s3cfg.Identities {
		if userName != "" && userName != ident.Name {
			continue
		}
		for _, cred := range ident.Credentials {
			//QUYNGUYEN add to avoid STS key
			if cred.Expiration > 0 {
				continue
			}
			//QUYNGUYEN end
			resp.ListAccessKeysResult.AccessKeyMetadata = append(resp.ListAccessKeysResult.AccessKeyMetadata,
				&iam.AccessKeyMetadata{UserName: &ident.Name, AccessKeyId: &cred.AccessKey, Status: &status},
			)
		}
	}
	return resp
}

func (iama *IamApiServer) CreateUser(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp CreateUserResponse) {
	userName := values.Get("UserName")
	//QUYNGUYEN add to fix duplicate user
	for _, ident := range s3cfg.Identities {
		if userName == ident.Name {
			resp.CreateUserResult.User.UserName = &userName
			return resp
		}
	}
	//QUYNGUYEN end
	resp.CreateUserResult.User.UserName = &userName
	s3cfg.Identities = append(s3cfg.Identities, &iam_pb.Identity{Name: userName})
	return resp
}

func (iama *IamApiServer) DeleteUser(s3cfg *iam_pb.S3ApiConfiguration, userName string) (resp DeleteUserResponse, err *IamError) {
	for i, ident := range s3cfg.Identities {
		if userName == ident.Name {
			s3cfg.Identities = append(s3cfg.Identities[:i], s3cfg.Identities[i+1:]...)
			return resp, nil
		}
	}
	return resp, &IamError{Code: iam.ErrCodeNoSuchEntityException, Error: fmt.Errorf(USER_DOES_NOT_EXIST, userName)}
}

func (iama *IamApiServer) GetUser(s3cfg *iam_pb.S3ApiConfiguration, userName string) (resp GetUserResponse, err *IamError) {
	for _, ident := range s3cfg.Identities {
		if userName == ident.Name {
			resp.GetUserResult.User = iam.User{UserName: &ident.Name}
			return resp, nil
		}
	}
	return resp, &IamError{Code: iam.ErrCodeNoSuchEntityException, Error: fmt.Errorf(USER_DOES_NOT_EXIST, userName)}
}

func (iama *IamApiServer) UpdateUser(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp UpdateUserResponse, err *IamError) {
	userName := values.Get("UserName")
	newUserName := values.Get("NewUserName")
	if newUserName != "" {
		for _, ident := range s3cfg.Identities {
			if userName == ident.Name {
				ident.Name = newUserName
				return resp, nil
			}
		}
	} else {
		return resp, nil
	}
	return resp, &IamError{Code: iam.ErrCodeNoSuchEntityException, Error: fmt.Errorf(USER_DOES_NOT_EXIST, userName)}
}

func GetPolicyDocument(policy *string) (policy_engine.PolicyDocument, error) {
	var policyDocument policy_engine.PolicyDocument
	if err := json.Unmarshal([]byte(*policy), &policyDocument); err != nil {
		return policy_engine.PolicyDocument{}, err
	}
	return policyDocument, nil
}

func (iama *IamApiServer) CreatePolicy(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp CreatePolicyResponse, iamError *IamError) {
	policyName := values.Get("PolicyName")
	policyDocumentString := values.Get("PolicyDocument")
	policyDocument, err := GetPolicyDocument(&policyDocumentString)
	if err != nil {
		return CreatePolicyResponse{}, &IamError{Code: iam.ErrCodeMalformedPolicyDocumentException, Error: err}
	}
	policyId := Hash(&policyDocumentString)
	arn := fmt.Sprintf("arn:aws:iam:::policy/%s", policyName)
	resp.CreatePolicyResult.Policy.PolicyName = &policyName
	resp.CreatePolicyResult.Policy.Arn = &arn
	resp.CreatePolicyResult.Policy.PolicyId = &policyId
	policies := Policies{}
	policyLock.Lock()
	defer policyLock.Unlock()
	if err = iama.s3ApiConfig.GetPolicies(&policies); err != nil {
		return resp, &IamError{Code: iam.ErrCodeServiceFailureException, Error: err}
	}
	policies.Policies[policyName] = policyDocument
	if err = iama.s3ApiConfig.PutPolicies(&policies); err != nil {
		return resp, &IamError{Code: iam.ErrCodeServiceFailureException, Error: err}
	}
	return resp, nil
}

type IamError struct {
	Code  string
	Error error
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_PutUserPolicy.html
func (iama *IamApiServer) PutUserPolicy(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp PutUserPolicyResponse, iamError *IamError) {
	userName := values.Get("UserName")
	policyName := values.Get("PolicyName")
	policyDocumentString := values.Get("PolicyDocument")
	policyDocument, err := GetPolicyDocument(&policyDocumentString)
	if err != nil {
		return PutUserPolicyResponse{}, &IamError{Code: iam.ErrCodeMalformedPolicyDocumentException, Error: err}
	}
	policyDocuments[policyName] = &policyDocument
	actions, err := GetActions(&policyDocument)
	if err != nil {
		return PutUserPolicyResponse{}, &IamError{Code: iam.ErrCodeMalformedPolicyDocumentException, Error: err}
	}
	// Log the actions
	glog.V(3).Infof("PutUserPolicy: actions=%v", actions)
	for _, ident := range s3cfg.Identities {
		if userName != ident.Name {
			continue
		}
		ident.Actions = actions
		return resp, nil
	}
	return PutUserPolicyResponse{}, &IamError{Code: iam.ErrCodeNoSuchEntityException, Error: fmt.Errorf("the user with name %s cannot be found", userName)}
}

func (iama *IamApiServer) GetUserPolicy(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp GetUserPolicyResponse, err *IamError) {
	userName := values.Get("UserName")
	policyName := values.Get("PolicyName")
	for _, ident := range s3cfg.Identities {
		if userName != ident.Name {
			continue
		}

		resp.GetUserPolicyResult.UserName = userName
		resp.GetUserPolicyResult.PolicyName = policyName
		if len(ident.Actions) == 0 {
			return resp, &IamError{Code: iam.ErrCodeNoSuchEntityException, Error: errors.New("no actions found")}
		}

		policyDocument := policy_engine.PolicyDocument{Version: policyDocumentVersion}
		statements := make(map[string][]string)
		for _, action := range ident.Actions {
			// parse "Read:EXAMPLE-BUCKET"
			act := strings.Split(action, ":")

			resource := "*"
			if len(act) == 2 {
				resource = fmt.Sprintf("arn:aws:s3:::%s/*", act[1])
			}
			statements[resource] = append(statements[resource],
				fmt.Sprintf("s3:%s", MapToIdentitiesAction(act[0])),
			)
		}
		for resource, actions := range statements {
			isEqAction := false
			for i, statement := range policyDocument.Statement {
				if reflect.DeepEqual(statement.Action.Strings(), actions) {
					policyDocument.Statement[i].Resource = policy_engine.NewStringOrStringSlice(append(
						policyDocument.Statement[i].Resource.Strings(), resource)...)
					isEqAction = true
					break
				}
			}
			if isEqAction {
				continue
			}
			policyDocumentStatement := policy_engine.PolicyStatement{
				Effect:   policy_engine.PolicyEffectAllow,
				Action:   policy_engine.NewStringOrStringSlice(actions...),
				Resource: policy_engine.NewStringOrStringSlice(resource),
			}
			policyDocument.Statement = append(policyDocument.Statement, policyDocumentStatement)
		}
		policyDocumentJSON, err := json.Marshal(policyDocument)
		if err != nil {
			return resp, &IamError{Code: iam.ErrCodeServiceFailureException, Error: err}
		}
		resp.GetUserPolicyResult.PolicyDocument = string(policyDocumentJSON)
		return resp, nil
	}
	return resp, &IamError{Code: iam.ErrCodeNoSuchEntityException, Error: fmt.Errorf(USER_DOES_NOT_EXIST, userName)}
}

func (iama *IamApiServer) DeleteUserPolicy(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp PutUserPolicyResponse, err *IamError) {
	userName := values.Get("UserName")
	for i, ident := range s3cfg.Identities {
		if ident.Name == userName {
			s3cfg.Identities = append(s3cfg.Identities[:i], s3cfg.Identities[i+1:]...)
			return resp, nil
		}
	}
	return resp, &IamError{Code: iam.ErrCodeNoSuchEntityException, Error: fmt.Errorf(USER_DOES_NOT_EXIST, userName)}
}

func GetActions(policy *policy_engine.PolicyDocument) ([]string, error) {
	var actions []string

	for _, statement := range policy.Statement {
		if statement.Effect != policy_engine.PolicyEffectAllow {
			return nil, fmt.Errorf("not a valid effect: '%s'. Only 'Allow' is possible", statement.Effect)
		}
		for _, resource := range statement.Resource.Strings() {
			// Parse "arn:aws:s3:::my-bucket/shared/*"
			res := strings.Split(resource, ":")
			if len(res) != 6 || res[0] != "arn" || res[1] != "aws" || res[2] != "s3" {
				continue
			}
			for _, action := range statement.Action.Strings() {
				// Parse "s3:Get*"
				act := strings.Split(action, ":")
				if len(act) != 2 || act[0] != "s3" {
					continue
				}
				statementAction := MapToStatementAction(act[1])

				if statementAction == "" {
					return nil, fmt.Errorf("not a valid action: '%s'", act[1])
				}

				path := res[5]
				if path == "*" {
					actions = append(actions, statementAction)
					continue
				}
				actions = append(actions, fmt.Sprintf("%s:%s", statementAction, path))
			}
		}
	}
	return actions, nil
}

func (iama *IamApiServer) CreateAccessKey(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp CreateAccessKeyResponse) {
	userName := values.Get("UserName")
	status := iam.StatusTypeActive
	accessKeyId := StringWithCharset(21, charsetUpper)
	secretAccessKey := StringWithCharset(42, charset)
	resp.CreateAccessKeyResult.AccessKey.AccessKeyId = &accessKeyId
	resp.CreateAccessKeyResult.AccessKey.SecretAccessKey = &secretAccessKey
	resp.CreateAccessKeyResult.AccessKey.UserName = &userName
	resp.CreateAccessKeyResult.AccessKey.Status = &status
	changed := false
	for _, ident := range s3cfg.Identities {
		if userName == ident.Name {
			ident.Credentials = append(ident.Credentials,
				&iam_pb.Credential{AccessKey: accessKeyId, SecretKey: secretAccessKey})
			changed = true
			break
		}
	}
	if !changed {
		s3cfg.Identities = append(s3cfg.Identities,
			&iam_pb.Identity{
				Name: userName,
				Credentials: []*iam_pb.Credential{
					{
						AccessKey: accessKeyId,
						SecretKey: secretAccessKey,
					},
				},
			},
		)
	}
	return resp
}

// QUYNGUYEN add
func (iama *IamApiServer) AssumeRoleWithWebIdentity(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp AssumeRoleWithWebIdentityResponse, err *IamError) {
	// Parse parameters từ request
	roleArn := values.Get("RoleArn")
	roleSessionName := values.Get("RoleSessionName")
	durationSecondsStr := values.Get("DurationSeconds")
	policy := values.Get("Policy")
	webIdentityToken := values.Get("WebIdentityToken")

	// Validate required parameters
	if roleArn == "" || roleSessionName == "" {
		err = &IamError{
			Code:  iam.ErrCodeInvalidInputException,
			Error: errors.New("RoleArn and RoleSessionName are required"),
		}
		return
	}

	// Tạo AssumeRoleRequest
	request := &seaweedSts.AssumeRoleWithWebIdentityRequest{
		RoleArn:          roleArn,
		RoleSessionName:  roleSessionName,
		WebIdentityToken: webIdentityToken,
	}

	// Parse DurationSeconds nếu có
	if durationSecondsStr != "" {
		if durationSeconds, err := strconv.ParseInt(durationSecondsStr, 10, 64); err == nil {
			request.DurationSeconds = &durationSeconds
		}
	}

	// Set Policy nếu có
	if policy != "" {
		request.Policy = &policy
	}

	// Gọi iamManager.AssumeRole
	ctx := context.Background()
	iamManager := iama.iam.GetIAMManager()
	if iamManager == nil {
		err = &IamError{
			Code:  iam.ErrCodeInvalidInputException,
			Error: errors.New("IAM manager not initialized"),
		}
		return
	}
	assumeRoleResp, assumeRoleErr := iamManager.AssumeRoleWithWebIdentity(ctx, request)
	if assumeRoleErr != nil {
		glog.V(0).Infof("Error assuming role: %v", assumeRoleErr)
		err = &IamError{
			Code:  iam.ErrCodeInvalidInputException,
			Error: errors.New("RoleArn and RoleSessionName are required"),
		}

		return
	}

	// Convert response
	resp.AssumeRoleWithWebIdentityResult.Credentials = sts.Credentials{
		AccessKeyId:     &assumeRoleResp.Credentials.AccessKeyId,
		SecretAccessKey: &assumeRoleResp.Credentials.SecretAccessKey,
		SessionToken:    &assumeRoleResp.Credentials.SessionToken,
		Expiration:      &assumeRoleResp.Credentials.Expiration,
	}
	resp.AssumeRoleWithWebIdentityResult.AssumedRoleUser = sts.AssumedRoleUser{
		AssumedRoleId: &assumeRoleResp.AssumedRoleUser.AssumedRoleId,
		Arn:           &assumeRoleResp.AssumedRoleUser.Arn,
	}
	return
}
func (iama *IamApiServer) AssumeRoleWithCredentials(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp AssumeRoleResponse, err *IamError) {
	// Parse parameters từ request
	roleArn := values.Get("RoleArn")
	roleSessionName := values.Get("RoleSessionName")
	durationSecondsStr := values.Get("DurationSeconds")
	policy := values.Get("Policy")
	accessKeyId := values.Get("AccessKeyId")
	secretAccessKey := values.Get("SecretAccessKey")
	providerName := values.Get("ProviderName")

	// Validate required parameters
	if roleArn == "" || roleSessionName == "" {
		err = &IamError{
			Code:  iam.ErrCodeInvalidInputException,
			Error: errors.New("RoleArn and RoleSessionName are required"),
		}
		return
	}

	// Tạo AssumeRoleRequest
	request := &seaweedSts.AssumeRoleWithCredentialsRequest{
		RoleArn:         roleArn,
		RoleSessionName: roleSessionName,
		Username:        accessKeyId,
		Password:        secretAccessKey,
	}

	// Parse DurationSeconds nếu có
	if durationSecondsStr != "" {
		if durationSeconds, err := strconv.ParseInt(durationSecondsStr, 10, 64); err == nil {
			request.DurationSeconds = &durationSeconds
		}
	}

	// Set Policy nếu có
	if policy != "" {
		request.Policy = &policy
	}

	if providerName != "" {
		request.ProviderName = providerName
	} else {
		request.ProviderName = "keycloak"
	}

	// Gọi iamManager.AssumeRole
	ctx := context.Background()
	iamManager := iama.iam.GetIAMManager()
	if iamManager == nil {
		err = &IamError{
			Code:  iam.ErrCodeInvalidInputException,
			Error: errors.New("IAM manager not initialized"),
		}
		return
	}
	assumeRoleResp, assumeRoleErr := iamManager.AssumeRoleWithCredentials(ctx, request)
	if assumeRoleErr != nil {
		glog.V(0).Infof("Error assuming role: %v", assumeRoleErr)
		err = &IamError{
			Code:  iam.ErrCodeInvalidInputException,
			Error: errors.New("RoleArn and RoleSessionName are required"),
		}

		return
	}

	// Convert response
	resp.AssumeRoleResult.Credentials = sts.Credentials{
		AccessKeyId:     &assumeRoleResp.Credentials.AccessKeyId,
		SecretAccessKey: &assumeRoleResp.Credentials.SecretAccessKey,
		SessionToken:    &assumeRoleResp.Credentials.SessionToken,
		Expiration:      &assumeRoleResp.Credentials.Expiration,
	}
	resp.AssumeRoleResult.AssumedRoleUser = sts.AssumedRoleUser{
		AssumedRoleId: &assumeRoleResp.AssumedRoleUser.AssumedRoleId,
		Arn:           &assumeRoleResp.AssumedRoleUser.Arn,
	}
	return
}
func (iama *IamApiServer) GetSessionToken(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp GetSessionTokenResponse, err *IamError) {
	durationSecondsStr := values.Get("DurationSeconds")
	serialNumber := values.Get("SerialNumber")
	tokenCode := values.Get("TokenCode")

	// Tạo GetSessionTokenRequest
	request := &seaweedSts.GetSessionTokenRequest{}

	// Parse DurationSeconds nếu có
	if durationSecondsStr != "" {
		if durationSeconds, err := strconv.ParseInt(durationSecondsStr, 10, 64); err == nil {
			request.DurationSeconds = &durationSeconds
		}
	}

	// Set SerialNumber và TokenCode nếu có
	if serialNumber != "" {
		request.SerialNumber = &serialNumber
	}
	if tokenCode != "" {
		request.TokenCode = &tokenCode
	}

	ctx := context.Background()
	iamManager := iama.iam.GetIAMManager()
	if iamManager == nil {
		err = &IamError{
			Code:  iam.ErrCodeInvalidInputException,
			Error: errors.New("IAM manager not initialized"),
		}
		return
	}

	// Gọi stsService.GetSessionToken
	getSessionTokenResp, sessionTokenErr := iamManager.GetSessionToken(ctx, request)
	if err != nil {
		glog.V(0).Infof("Error assuming role: %v", sessionTokenErr)
		err = &IamError{
			Code:  iam.ErrCodeInvalidInputException,
			Error: errors.New("RoleArn and RoleSessionName are required"),
		}
		return
	}

	// Convert response
	response := GetSessionTokenResponse{}
	if getSessionTokenResp.Credentials != nil {
		response.GetSessionTokenResult.Credentials = sts.Credentials{
			AccessKeyId:     &getSessionTokenResp.Credentials.AccessKeyId,
			SecretAccessKey: &getSessionTokenResp.Credentials.SecretAccessKey,
			SessionToken:    &getSessionTokenResp.Credentials.SessionToken,
			Expiration:      &getSessionTokenResp.Credentials.Expiration,
		}
	}
	return
}

//QUYNGUYEN end

func CreateSessionToken(payload cred.SessionTokenPayload, secret string) (string, error) {
	// Create a new JWT with HS512 algorithm
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"accessKey": payload.AccessKey,
		"exp":       payload.Exp,
		"iat":       payload.Iat,
	})

	// Set the header explicitly
	token.Header["alg"] = "HS512"
	token.Header["typ"] = "JWT"

	// Sign the token with the secret
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		glog.Infof("Error creating custom SessionToken: %v", err)
		return "", err
	}

	return tokenString, nil
}

//QUYNGUYEN end

func (iama *IamApiServer) DeleteAccessKey(s3cfg *iam_pb.S3ApiConfiguration, values url.Values) (resp DeleteAccessKeyResponse) {
	userName := values.Get("UserName")
	accessKeyId := values.Get("AccessKeyId")
	for _, ident := range s3cfg.Identities {
		if userName == ident.Name {
			for i, cred := range ident.Credentials {
				if cred.AccessKey == accessKeyId {
					ident.Credentials = append(ident.Credentials[:i], ident.Credentials[i+1:]...)
					break
				}
			}
			break
		}
	}
	return resp
}

// handleImplicitUsername adds username who signs the request to values if 'username' is not specified
// According to https://awscli.amazonaws.com/v2/documentation/api/latest/reference/iam/create-access-key.html/
// "If you do not specify a user name, IAM determines the user name implicitly based on the Amazon Web
// Services access key ID signing the request."
func handleImplicitUsername(r *http.Request, values url.Values) {
	if len(r.Header["Authorization"]) == 0 || values.Get("UserName") != "" {
		return
	}
	// get username who signs the request. For a typical Authorization:
	// "AWS4-HMAC-SHA256 Credential=197FSAQ7HHTA48X64O3A/20220420/test1/iam/aws4_request, SignedHeaders=content-type;
	// host;x-amz-date, Signature=6757dc6b3d7534d67e17842760310e99ee695408497f6edc4fdb84770c252dc8",
	// the "test1" will be extracted as the username
	glog.V(4).Infof("Authorization field: %v", r.Header["Authorization"][0])
	s := strings.Split(r.Header["Authorization"][0], "Credential=")
	if len(s) < 2 {
		return
	}
	s = strings.Split(s[1], ",")
	if len(s) < 2 {
		return
	}
	s = strings.Split(s[0], "/")
	if len(s) < 5 {
		return
	}
	userName := s[2]
	values.Set("UserName", userName)
}

func (iama *IamApiServer) DoActions(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s3err.WriteErrorResponse(w, r, s3err.ErrInvalidRequest)
		return
	}
	values := r.PostForm
	s3cfg := &iam_pb.S3ApiConfiguration{}
	if err := iama.s3ApiConfig.GetS3ApiConfiguration(s3cfg); err != nil && !errors.Is(err, filer_pb.ErrNotFound) {
		s3err.WriteErrorResponse(w, r, s3err.ErrInternalError)
		return
	}

	glog.V(4).Infof("DoActions: %+v", values)
	var response interface{}
	var iamError *IamError
	changed := true
	switch r.Form.Get("Action") {
	case "ListUsers":
		response = iama.ListUsers(s3cfg, values)
		changed = false
	case "ListAccessKeys":
		handleImplicitUsername(r, values)
		response = iama.ListAccessKeys(s3cfg, values)
		changed = false
	case "CreateUser":
		response = iama.CreateUser(s3cfg, values)
	case "GetUser":
		userName := values.Get("UserName")
		response, iamError = iama.GetUser(s3cfg, userName)
		if iamError != nil {
			writeIamErrorResponse(w, r, iamError)
			return
		}
		changed = false
	case "UpdateUser":
		response, iamError = iama.UpdateUser(s3cfg, values)
		if iamError != nil {
			glog.Errorf("UpdateUser: %+v", iamError.Error)
			s3err.WriteErrorResponse(w, r, s3err.ErrInvalidRequest)
			return
		}
	case "DeleteUser":
		userName := values.Get("UserName")
		response, iamError = iama.DeleteUser(s3cfg, userName)
		if iamError != nil {
			writeIamErrorResponse(w, r, iamError)
			return
		}
	case "CreateAccessKey":
		handleImplicitUsername(r, values)
		response = iama.CreateAccessKey(s3cfg, values)
	case "DeleteAccessKey":
		handleImplicitUsername(r, values)
		response = iama.DeleteAccessKey(s3cfg, values)
	case "CreatePolicy":
		response, iamError = iama.CreatePolicy(s3cfg, values)
		if iamError != nil {
			glog.Errorf("CreatePolicy:  %+v", iamError.Error)
			s3err.WriteErrorResponse(w, r, s3err.ErrInvalidRequest)
			return
		}
	case "PutUserPolicy":
		var iamError *IamError
		response, iamError = iama.PutUserPolicy(s3cfg, values)
		if iamError != nil {
			glog.Errorf("PutUserPolicy:  %+v", iamError.Error)

			writeIamErrorResponse(w, r, iamError)
			return
		}
	case "GetUserPolicy":
		response, iamError = iama.GetUserPolicy(s3cfg, values)
		if iamError != nil {
			writeIamErrorResponse(w, r, iamError)
			return
		}
		changed = false
	case "DeleteUserPolicy":
		if response, iamError = iama.DeleteUserPolicy(s3cfg, values); iamError != nil {
			writeIamErrorResponse(w, r, iamError)
			return
		}
	//QUYNGUYEN add
	case "GetSessionToken":
		if response, iamError = iama.GetSessionToken(s3cfg, values); iamError != nil {
			writeIamErrorResponse(w, r, iamError)
			return
		}
	case "AssumeRole":
		if response, iamError = iama.AssumeRoleWithCredentials(s3cfg, values); iamError != nil {
			writeIamErrorResponse(w, r, iamError)
			return
		}
	case "AssumeRoleWithWebIdentity":
		if response, iamError = iama.AssumeRoleWithWebIdentity(s3cfg, values); iamError != nil {
			writeIamErrorResponse(w, r, iamError)
			return
		}
	//QUYNGUYEN end
	default:
		errNotImplemented := s3err.GetAPIError(s3err.ErrNotImplemented)
		errorResponse := ErrorResponse{}
		errorResponse.Error.Code = &errNotImplemented.Code
		errorResponse.Error.Message = &errNotImplemented.Description
		s3err.WriteXMLResponse(w, r, errNotImplemented.HTTPStatusCode, errorResponse)
		return
	}
	if changed {
		err := iama.s3ApiConfig.PutS3ApiConfiguration(s3cfg)
		if err != nil {
			var iamError = IamError{Code: iam.ErrCodeServiceFailureException, Error: err}
			writeIamErrorResponse(w, r, &iamError)
			return
		}
	}
	s3err.WriteXMLResponse(w, r, http.StatusOK, response)
}

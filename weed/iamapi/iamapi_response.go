package iamapi

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
)

type CommonResponse struct {
	ResponseMetadata struct {
		RequestId string `xml:"RequestId"`
	} `xml:"ResponseMetadata"`
}

type ListUsersResponse struct {
	CommonResponse
	XMLName         xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ ListUsersResponse"`
	ListUsersResult struct {
		Users       []*iam.User `xml:"Users>member"`
		IsTruncated bool        `xml:"IsTruncated"`
	} `xml:"ListUsersResult"`
}

type ListAccessKeysResponse struct {
	CommonResponse
	XMLName              xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ ListAccessKeysResponse"`
	ListAccessKeysResult struct {
		AccessKeyMetadata []*iam.AccessKeyMetadata `xml:"AccessKeyMetadata>member"`
		IsTruncated       bool                     `xml:"IsTruncated"`
	} `xml:"ListAccessKeysResult"`
}

type DeleteAccessKeyResponse struct {
	CommonResponse
	XMLName xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ DeleteAccessKeyResponse"`
}

type CreatePolicyResponse struct {
	CommonResponse
	XMLName            xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ CreatePolicyResponse"`
	CreatePolicyResult struct {
		Policy iam.Policy `xml:"Policy"`
	} `xml:"CreatePolicyResult"`
}

type CreateUserResponse struct {
	CommonResponse
	XMLName          xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ CreateUserResponse"`
	CreateUserResult struct {
		User iam.User `xml:"User"`
	} `xml:"CreateUserResult"`
}

type DeleteUserResponse struct {
	CommonResponse
	XMLName xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ DeleteUserResponse"`
}

type GetUserResponse struct {
	CommonResponse
	XMLName       xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ GetUserResponse"`
	GetUserResult struct {
		User iam.User `xml:"User"`
	} `xml:"GetUserResult"`
}

type UpdateUserResponse struct {
	CommonResponse
	XMLName xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ UpdateUserResponse"`
}

type CreateAccessKeyResponse struct {
	CommonResponse
	XMLName               xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ CreateAccessKeyResponse"`
	CreateAccessKeyResult struct {
		AccessKey iam.AccessKey `xml:"AccessKey"`
	} `xml:"CreateAccessKeyResult"`
}

type AssumeRoleResponse struct {
	CommonResponse
	XMLName          xml.Name `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleResponse"`
	AssumeRoleResult struct {
		Credentials     sts.Credentials     `xml:"Credentials"`
		AssumedRoleUser sts.AssumedRoleUser `xml:"AssumedRoleUser"`
	} `xml:"AssumeRoleResult"`
}

type GetSessionTokenResponse struct {
	CommonResponse
	XMLName          xml.Name `xml:"https://sts.amazonaws.com/doc/2011-06-15/ GetSessionTokenResponse"`
	AssumeRoleResult struct {
		Credentials sts.Credentials `xml:"Credentials"`
	} `xml:"GetSessionTokenResult"`
}
type GetFederationTokenResponse struct {
	CommonResponse
	XMLName          xml.Name `xml:"https://sts.amazonaws.com/doc/2011-06-15/ GetFederationTokenResponse"`
	AssumeRoleResult struct {
		Credentials   sts.Credentials   `xml:"Credentials"`
		FederatedUser sts.FederatedUser `xml:"FederatedUser"`
	} `xml:"GetFederationTokenResult"`
}

type PutUserPolicyResponse struct {
	CommonResponse
	XMLName xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ PutUserPolicyResponse"`
}

type GetUserPolicyResponse struct {
	CommonResponse
	XMLName             xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ GetUserPolicyResponse"`
	GetUserPolicyResult struct {
		UserName       string `xml:"UserName"`
		PolicyName     string `xml:"PolicyName"`
		PolicyDocument string `xml:"PolicyDocument"`
	} `xml:"GetUserPolicyResult"`
}

type ErrorResponse struct {
	CommonResponse
	XMLName xml.Name `xml:"https://iam.amazonaws.com/doc/2010-05-08/ ErrorResponse"`
	Error   struct {
		iam.ErrorDetails
		Type string `xml:"Type"`
	} `xml:"Error"`
}

func (r *CommonResponse) SetRequestId() {
	r.ResponseMetadata.RequestId = fmt.Sprintf("%d", time.Now().UnixNano())
}

package dash

import (
	"context"
	"fmt"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/credential"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/iam/integration"
	"github.com/seaweedfs/seaweedfs/weed/iam/policy"
	"github.com/seaweedfs/seaweedfs/weed/s3api/policy_engine"
)

type IAMPolicy struct {
	Name         string                       `json:"name"`
	Document     policy_engine.PolicyDocument `json:"document"`
	DocumentJSON string                       `json:"document_json"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
}

type PoliciesCollection struct {
	Policies map[string]policy_engine.PolicyDocument `json:"policies"`
}

type PoliciesData struct {
	Username      string      `json:"username"`
	Policies      []IAMPolicy `json:"policies"`
	TotalPolicies int         `json:"total_policies"`
	LastUpdated   time.Time   `json:"last_updated"`
}

// Policy management request structures
type CreatePolicyRequest struct {
	Name         string                       `json:"name" binding:"required"`
	Document     policy_engine.PolicyDocument `json:"document" binding:"required"`
	DocumentJSON string                       `json:"document_json"`
}

type UpdatePolicyRequest struct {
	Document     policy_engine.PolicyDocument `json:"document" binding:"required"`
	DocumentJSON string                       `json:"document_json"`
}

// Role management request structures
type IAMRole struct {
	Name             string                `json:"name"`
	RoleArn          string                `json:"role_arn"`
	TrustPolicy      policy.PolicyDocument `json:"trust_policy"`
	TrustPolicyJSON  string                `json:"trust_policy_json"`
	AttachedPolicies []string              `json:"attached_policies"`
	Description      string                `json:"description,omitempty"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

type RolesCollection struct {
	Roles map[string]integration.RoleDefinition `json:"roles"`
}

type RolesData struct {
	Username    string    `json:"username"`
	Roles       []IAMRole `json:"roles"`
	TotalRoles  int       `json:"total_roles"`
	LastUpdated time.Time `json:"last_updated"`
}

type IAMPoliciesData struct {
	Username      string                                  `json:"username"`
	Policies      map[string]policy_engine.PolicyDocument `json:"policies"`
	TotalPolicies int                                     `json:"total_policies"`
	LastUpdated   time.Time                               `json:"last_updated"`
}

type CreateRoleRequest struct {
	Name             string                `json:"name" binding:"required"`
	RoleArn          string                `json:"role_arn"`
	TrustPolicy      policy.PolicyDocument `json:"trust_policy" binding:"required"`
	TrustPolicyJSON  string                `json:"trust_policy_json"`
	AttachedPolicies []string              `json:"attached_policies"`
	Description      string                `json:"description,omitempty"`
}

type UpdateRoleRequest struct {
	TrustPolicy      policy.PolicyDocument `json:"trust_policy" binding:"required"`
	TrustPolicyJSON  string                `json:"trust_policy_json"`
	AttachedPolicies []string              `json:"attached_policies"`
	Description      string                `json:"description,omitempty"`
}

// PolicyManager interface is now in the credential package

// CredentialStorePolicyManager implements credential.PolicyManager by delegating to the credential store
type CredentialStorePolicyManager struct {
	credentialManager *credential.CredentialManager
}

// NewCredentialStorePolicyManager creates a new CredentialStorePolicyManager
func NewCredentialStorePolicyManager(credentialManager *credential.CredentialManager) *CredentialStorePolicyManager {
	return &CredentialStorePolicyManager{
		credentialManager: credentialManager,
	}
}

// GetPolicies retrieves all IAM policies via credential store
func (cspm *CredentialStorePolicyManager) GetPolicies(ctx context.Context) (map[string]policy_engine.PolicyDocument, error) {
	// Get policies from credential store
	// We'll use the credential store to access the filer indirectly
	// Since policies are stored separately, we need to access the underlying store
	store := cspm.credentialManager.GetStore()
	glog.V(1).Infof("Getting policies from credential store: %T", store)

	// Check if the store supports policy management
	if policyStore, ok := store.(credential.PolicyManager); ok {
		glog.V(1).Infof("Store supports policy management, calling GetPolicies")
		policies, err := policyStore.GetPolicies(ctx)
		if err != nil {
			glog.Errorf("Error getting policies from store: %v", err)
			return nil, err
		}
		glog.V(1).Infof("Got %d policies from store", len(policies))
		return policies, nil
	} else {
		// Fallback: use empty policies for stores that don't support policies
		glog.V(1).Infof("Credential store doesn't support policy management, returning empty policies")
		return make(map[string]policy_engine.PolicyDocument), nil
	}
}

// CreatePolicy creates a new IAM policy via credential store
func (cspm *CredentialStorePolicyManager) CreatePolicy(ctx context.Context, name string, document policy_engine.PolicyDocument) error {
	store := cspm.credentialManager.GetStore()

	if policyStore, ok := store.(credential.PolicyManager); ok {
		return policyStore.CreatePolicy(ctx, name, document)
	}

	return fmt.Errorf("credential store doesn't support policy creation")
}

// UpdatePolicy updates an existing IAM policy via credential store
func (cspm *CredentialStorePolicyManager) UpdatePolicy(ctx context.Context, name string, document policy_engine.PolicyDocument) error {
	store := cspm.credentialManager.GetStore()

	if policyStore, ok := store.(credential.PolicyManager); ok {
		return policyStore.UpdatePolicy(ctx, name, document)
	}

	return fmt.Errorf("credential store doesn't support policy updates")
}

// DeletePolicy deletes an IAM policy via credential store
func (cspm *CredentialStorePolicyManager) DeletePolicy(ctx context.Context, name string) error {
	store := cspm.credentialManager.GetStore()

	if policyStore, ok := store.(credential.PolicyManager); ok {
		return policyStore.DeletePolicy(ctx, name)
	}

	return fmt.Errorf("credential store doesn't support policy deletion")
}

// GetPolicy retrieves a specific IAM policy via credential store
func (cspm *CredentialStorePolicyManager) GetPolicy(ctx context.Context, name string) (*policy_engine.PolicyDocument, error) {
	store := cspm.credentialManager.GetStore()

	if policyStore, ok := store.(credential.PolicyManager); ok {
		return policyStore.GetPolicy(ctx, name)
	}

	return nil, fmt.Errorf("credential store doesn't support policy retrieval")
}

// AdminServer policy management methods using credential.PolicyManager
func (s *AdminServer) GetPolicyManager() credential.PolicyManager {
	if s.credentialManager == nil {
		glog.V(1).Infof("Credential manager is nil, policy management not available")
		return nil
	}
	glog.V(1).Infof("Credential manager available, creating CredentialStorePolicyManager")
	return NewCredentialStorePolicyManager(s.credentialManager)
}

// GetPolicies retrieves all IAM policies
func (s *AdminServer) GetPolicies() ([]IAMPolicy, error) {
	policyManager := s.GetPolicyManager()
	if policyManager == nil {
		return nil, fmt.Errorf("policy manager not available")
	}

	ctx := context.Background()
	policyMap, err := policyManager.GetPolicies(ctx)
	if err != nil {
		return nil, err
	}

	// Convert map[string]PolicyDocument to []IAMPolicy
	var policies []IAMPolicy
	for name, doc := range policyMap {
		policy := IAMPolicy{
			Name:         name,
			Document:     doc,
			DocumentJSON: "", // Will be populated if needed
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

// CreatePolicy creates a new IAM policy
func (s *AdminServer) CreatePolicy(name string, document policy_engine.PolicyDocument) error {
	policyManager := s.GetPolicyManager()
	if policyManager == nil {
		return fmt.Errorf("policy manager not available")
	}

	ctx := context.Background()
	return policyManager.CreatePolicy(ctx, name, document)
}

// UpdatePolicy updates an existing IAM policy
func (s *AdminServer) UpdatePolicy(name string, document policy_engine.PolicyDocument) error {
	policyManager := s.GetPolicyManager()
	if policyManager == nil {
		return fmt.Errorf("policy manager not available")
	}

	ctx := context.Background()
	return policyManager.UpdatePolicy(ctx, name, document)
}

// DeletePolicy deletes an IAM policy
func (s *AdminServer) DeletePolicy(name string) error {
	policyManager := s.GetPolicyManager()
	if policyManager == nil {
		return fmt.Errorf("policy manager not available")
	}

	ctx := context.Background()
	return policyManager.DeletePolicy(ctx, name)
}

// GetPolicy retrieves a specific IAM policy
func (s *AdminServer) GetPolicy(name string) (*IAMPolicy, error) {
	policyManager := s.GetPolicyManager()
	if policyManager == nil {
		return nil, fmt.Errorf("policy manager not available")
	}

	ctx := context.Background()
	policyDoc, err := policyManager.GetPolicy(ctx, name)
	if err != nil {
		return nil, err
	}

	if policyDoc == nil {
		return nil, nil
	}

	// Convert PolicyDocument to IAMPolicy
	policy := &IAMPolicy{
		Name:         name,
		Document:     *policyDoc,
		DocumentJSON: "", // Will be populated if needed
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return policy, nil
}

// Role management methods using IAM Manager

// GetIAMManager retrieves or creates an IAM manager instance
func (s *AdminServer) GetIAMManager() *integration.IAMManager {

	glog.V(1).Infof("GetIAMManager called, credentialManager is nil: %v", s.credentialManager == nil)
	if s.credentialManager == nil {
		glog.V(0).Infof("Credential manager is nil, IAM manager not available")
		return nil
	}

	return s.iamManager
}

// GetRoles retrieves all IAM roles
func (s *AdminServer) GetRoles() ([]IAMRole, error) {
	iamManager := s.GetIAMManager()
	if iamManager == nil {
		return nil, fmt.Errorf("IAM manager not available")
	}

	ctx := context.Background()
	roleNames, err := iamManager.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	// Convert role names to []IAMRole
	var roles []IAMRole
	for _, roleName := range roleNames {
		// Get role from role store directly since IAM manager doesn't expose GetRole
		roleStore := iamManager.GetRoleStore()
		if roleStore == nil {
			glog.Errorf("Role store is not available")
			continue
		}

		roleDef, err := roleStore.GetRole(ctx, "", roleName)
		if err != nil {
			glog.Errorf("Failed to get role %s: %v", roleName, err)
			continue // Skip roles that can't be retrieved
		}

		role := IAMRole{
			Name:             roleName,
			RoleArn:          roleDef.RoleArn,
			TrustPolicy:      *roleDef.TrustPolicy,
			TrustPolicyJSON:  "", // Will be populated if needed
			AttachedPolicies: roleDef.AttachedPolicies,
			Description:      roleDef.Description,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// CreateRole creates a new IAM role
func (s *AdminServer) CreateRole(name string, roleDef *integration.RoleDefinition) error {
	iamManager := s.GetIAMManager()
	if iamManager == nil {
		return fmt.Errorf("IAM manager not available")
	}

	ctx := context.Background()
	return iamManager.CreateRole(ctx, "", name, roleDef)
}

// UpdateRole updates an existing IAM role
func (s *AdminServer) UpdateRole(name string, roleDef *integration.RoleDefinition) error {
	iamManager := s.GetIAMManager()
	if iamManager == nil {
		return fmt.Errorf("IAM manager not available")
	}

	ctx := context.Background()
	return iamManager.CreateRole(ctx, "", name, roleDef) // CreateRole also updates existing roles
}

// DeleteRole deletes an IAM role
func (s *AdminServer) DeleteRole(name string) error {
	iamManager := s.GetIAMManager()
	if iamManager == nil {
		return fmt.Errorf("IAM manager not available")
	}

	ctx := context.Background()
	return iamManager.DeleteRole(ctx, name)
}

// GetRole retrieves a specific IAM role
func (s *AdminServer) GetRole(name string) (*IAMRole, error) {
	iamManager := s.GetIAMManager()
	if iamManager == nil {
		return nil, fmt.Errorf("IAM manager not available")
	}

	ctx := context.Background()

	// Get role from role store directly since IAM manager doesn't expose GetRole
	roleStore := iamManager.GetRoleStore()
	if roleStore == nil {
		return nil, fmt.Errorf("Role store is not available")
	}

	roleDef, err := roleStore.GetRole(ctx, "", name)
	if err != nil {
		return nil, err
	}

	if roleDef == nil {
		return nil, nil
	}

	// Convert RoleDefinition to IAMRole
	role := &IAMRole{
		Name:             name,
		RoleArn:          roleDef.RoleArn,
		TrustPolicy:      *roleDef.TrustPolicy,
		TrustPolicyJSON:  "", // Will be populated if needed
		AttachedPolicies: roleDef.AttachedPolicies,
		Description:      roleDef.Description,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	return role, nil
}

package iamapi

// This file re-exports IAM response types from the shared weed/iam package
// for backwards compatibility with existing code.

import (
	iamlib "github.com/seaweedfs/seaweedfs/weed/iam"
)

// Type aliases for IAM response types from shared package
type (
	CommonResponse                    = iamlib.CommonResponse
	ListUsersResponse                 = iamlib.ListUsersResponse
	ListAccessKeysResponse            = iamlib.ListAccessKeysResponse
	DeleteAccessKeyResponse           = iamlib.DeleteAccessKeyResponse
	CreatePolicyResponse              = iamlib.CreatePolicyResponse
	DeletePolicyResponse              = iamlib.DeletePolicyResponse
	ListPoliciesResponse              = iamlib.ListPoliciesResponse
	CreateUserResponse                = iamlib.CreateUserResponse
	DeleteUserResponse                = iamlib.DeleteUserResponse
	GetUserResponse                   = iamlib.GetUserResponse
	UpdateUserResponse                = iamlib.UpdateUserResponse
	CreateAccessKeyResponse           = iamlib.CreateAccessKeyResponse
	PutUserPolicyResponse             = iamlib.PutUserPolicyResponse
	DeleteUserPolicyResponse          = iamlib.DeleteUserPolicyResponse
	GetUserPolicyResponse             = iamlib.GetUserPolicyResponse
	ErrorResponse                     = iamlib.ErrorResponse
	AssumeRoleResponse                = iamlib.AssumeRoleResponse
	AssumeRoleWithWebIdentityResponse = iamlib.AssumeRoleWithWebIdentityResponse
	GetSessionTokenResponse           = iamlib.GetSessionTokenResponse
	CreateRoleResponse                = iamlib.CreateRoleResponse
	DeleteRoleResponse                = iamlib.DeleteRoleResponse
	ListRolesResponse                 = iamlib.ListRolesResponse
)

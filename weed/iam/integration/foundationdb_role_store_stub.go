//go:build !foundationdb
// +build !foundationdb

package integration

import (
	"fmt"
)

// NewFoundationDBPolicyStore creates a new FoundationDB-backed policy store
// This function is a stub that will be replaced by the actual implementation
// when the foundationdb build tag is used
// NewFoundationDBRoleStore creates a FoundationDB role store (stub for when foundationdb build tag is not used)
func NewFoundationDBRoleStore(config *RoleStoreConfig) (RoleStore, error) {
	return nil, fmt.Errorf("FoundationDB support not compiled. Please rebuild with -tags foundationdb")
}

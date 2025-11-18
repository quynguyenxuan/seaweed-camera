//go:build !foundationdb
// +build !foundationdb

package policy

import (
	"fmt"
)

// NewFoundationDBPolicyStore creates a new FoundationDB-backed policy store
// This function is a stub that will be replaced by the actual implementation
// when the foundationdb build tag is used
func NewFoundationDBPolicyStore(config map[string]interface{}) (PolicyStore, error) {
	return nil, fmt.Errorf("FoundationDB policy store not available - please build with foundationdb tag")
}

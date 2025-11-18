//go:build foundationdb
// +build foundationdb

package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/seaweedfs/seaweedfs/weed/filer/foundationdb"
	"github.com/seaweedfs/seaweedfs/weed/glog"
)

// FoundationDBPolicyStore implements PolicyStore interface using FoundationDB
type FoundationDBPolicyStore struct {
	kvDir         directory.DirectorySubspace
	timeout       time.Duration
	maxRetryDelay time.Duration
	store         *foundationdb.FoundationDBStore
}

// NewFoundationDBPolicyStore creates a new FoundationDB-backed policy store
func NewFoundationDBPolicyStore(config map[string]interface{}) (PolicyStore, error) {
	if config == nil {
		return nil, fmt.Errorf("policy store config cannot be nil")
	}

	filerStore := foundationdb.GetInstance()
	database := filerStore.GetDatabase()

	// Set timeout
	timeout := 5 * time.Second
	if timeoutStr, ok := config["timeout"].(string); ok && timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	// Set max retry delay
	maxRetryDelay := 2 * time.Second
	if retryDelayStr, ok := config["max_retry_delay"].(string); ok && retryDelayStr != "" {
		if parsedDelay, err := time.ParseDuration(retryDelayStr); err == nil {
			maxRetryDelay = parsedDelay
		}
	}

	// Create directory for policy storage
	kvDir, err := directory.Create(database, []string{"sunfs", "policies", "kv"}, nil)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create FoundationDB directory: %v", err)
	}

	store := &FoundationDBPolicyStore{
		kvDir:         kvDir,
		timeout:       timeout,
		maxRetryDelay: maxRetryDelay,
		store:         filerStore,
	}

	glog.V(0).Infof("FoundationDB policy store initialized")
	return store, nil
}

// StorePolicy stores a policy document in FoundationDB
func (f *FoundationDBPolicyStore) StorePolicy(ctx context.Context, filerAddress string, policyName string, policy *PolicyDocument) error {
	if policyName == "" {
		return fmt.Errorf("policy name cannot be empty")
	}
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	// Serialize policy to JSON
	policyData, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	// Create role key
	policyKey := f.kvDir.Pack(tuple.Tuple{string(policyName)})

	// Store in FoundationDB
	err = f.store.KeyPut(ctx, policyKey, policyData)

	if err != nil {
		return fmt.Errorf("failed to store policy: %w", err)
	}

	glog.V(2).Infof("Stored policy %s in FoundationDB", policyName)
	return nil
}

// GetPolicy retrieves a policy document from FoundationDB
func (f *FoundationDBPolicyStore) GetPolicy(ctx context.Context, filerAddress string, policyName string) (*PolicyDocument, error) {
	if policyName == "" {
		return nil, fmt.Errorf("policy name cannot be empty")
	}
	roleKey := f.kvDir.Pack(tuple.Tuple{string(policyName)})

	policyData, err := f.store.KeyGet(ctx, roleKey)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve policy: %w", err)
	}

	if len(policyData) == 0 {
		return nil, fmt.Errorf("policy not found: %s", policyName)
	}

	// Deserialize policy from JSON
	var policy PolicyDocument
	if err := json.Unmarshal(policyData, &policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	glog.V(2).Infof("Retrieved policy %s from FoundationDB", policyName)
	return &policy, nil
}

// DeletePolicy deletes a policy document from FoundationDB
func (f *FoundationDBPolicyStore) DeletePolicy(ctx context.Context, filerAddress string, policyName string) error {
	if policyName == "" {
		return fmt.Errorf("policy name cannot be empty")
	}
	roleKey := f.kvDir.Pack(tuple.Tuple{string(policyName)})
	err := f.store.KeyDelete(ctx, roleKey)
	

	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	glog.V(2).Infof("Deleted policy %s from FoundationDB", policyName)
	return nil
}

// ListPolicies lists all policy names in FoundationDB
func (f *FoundationDBPolicyStore) ListPolicies(ctx context.Context, filerAddress string) ([]string, error) {
	var policyNames []string
	kvSlice, err := f.store.KeyList(ctx, fdb.RangeOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %v", err)
	}

	// Extract policy names from keys
	for _, kv := range kvSlice {
		t, err := f.kvDir.Unpack(kv.Key)
		if err != nil {
			log.Printf("failed to unpack policy key %v: %v", kv.Key, err)
			continue
		}
		if len(t) >= 1 {
			if policyName, ok := t[0].(string); ok && policyName != "" {
				policyNames = append(policyNames, policyName)
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	glog.V(2).Infof("Listed %d policies from FoundationDB", len(policyNames))
	return policyNames, nil
}

// Close closes the FoundationDB connection
func (f *FoundationDBPolicyStore) Close() error {
	database := f.store.GetDatabase() 
	if database != nil {
		database.Close()
	}
	return nil
}

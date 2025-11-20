//go:build foundationdb
// +build foundationdb

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/seaweedfs/seaweedfs/weed/filer/foundationdb"
	"github.com/seaweedfs/seaweedfs/weed/glog"
)

// FoundationDBRoleStore implements RoleStore interface using FoundationDB
type FoundationDBRoleStore struct {
	kvDir         directory.DirectorySubspace
	timeout       time.Duration
	maxRetryDelay time.Duration
	store         *foundationdb.FoundationDBStore
}

// NewFoundationDBRoleStore creates a new FoundationDB-based role store
func NewFoundationDBRoleStore(config map[string]interface{}) (*FoundationDBRoleStore, error) {
	if config == nil {
		return nil, fmt.Errorf("role store config cannot be nil")
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

	// Create directory for role storage
	kvDir, err := directory.Create(database, []string{"sunfs", "roles", "kv"}, nil)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create FoundationDB directory: %v", err)
	}

	store := &FoundationDBRoleStore{
		kvDir:         kvDir,
		timeout:       timeout,
		maxRetryDelay: maxRetryDelay,
		store:         filerStore,
	}

	glog.V(0).Infof("FoundationDB role store initialized with cluster file:")
	return store, nil
}

// StoreRole stores a role definition in FoundationDB
func (f *FoundationDBRoleStore) StoreRole(ctx context.Context, filerAddress string, roleName string, role *RoleDefinition) error {
	if roleName == "" {
		return fmt.Errorf("role name cannot be empty")
	}
	if role == nil {
		return fmt.Errorf("role cannot be nil")
	}

	// Serialize role to JSON
	roleData, err := json.Marshal(role)
	if err != nil {
		return fmt.Errorf("failed to serialize role: %v", err)
	}

	// Create role key
	roleKey := f.kvDir.Pack(tuple.Tuple{string(roleName)})

	// Store in FoundationDB
	err = f.store.KeyPut(ctx, roleKey, roleData)

	if err != nil {
		return fmt.Errorf("failed to store role: %v", err)
	}

	return nil
}

// GetRole retrieves a role definition from FoundationDB
func (f *FoundationDBRoleStore) GetRole(ctx context.Context, filerAddress string, roleName string) (*RoleDefinition, error) {
	if roleName == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	// Create role key
	roleKey := f.kvDir.Pack(tuple.Tuple{string(roleName)})

	roleData, err := f.store.KeyGet(ctx, roleKey)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve role: %v", err)
	}

	if len(roleData) == 0 {
		return nil, fmt.Errorf("role not found: %s", roleName)
	}

	var role RoleDefinition
	if err := json.Unmarshal(roleData, &role); err != nil {
		return nil, fmt.Errorf("failed to deserialize role: %v", err)
	}

	return &role, nil
}

// ListRoles lists all role names from FoundationDB
func (f *FoundationDBRoleStore) ListRoles(ctx context.Context, filerAddress string) ([]string, error) {
	var roleNames []string

	// Get all role keys
	kvSlice, err := f.store.KeyList(ctx, fdb.RangeOptions{})

	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %v", err)
	}

	// Extract role names from keys
	for _, kv := range kvSlice {
		t, err := f.kvDir.Unpack(kv.Key)
		if err != nil {
			glog.Warningf("failed to unpack role key %v: %v", kv.Key, err)
			continue
		}
		if len(t) >= 1 {
			if roleName, ok := t[0].(string); ok && roleName != "" {
				roleNames = append(roleNames, roleName)
			}
		}
	}

	return roleNames, nil
}

// DeleteRole deletes a role definition from FoundationDB
func (f *FoundationDBRoleStore) DeleteRole(ctx context.Context, filerAddress string, roleName string) error {
	if roleName == "" {
		return fmt.Errorf("role name cannot be empty")
	}
	// Create role key
	roleKey := f.kvDir.Pack(tuple.Tuple{string(roleName)})

	// Delete from FoundationDB
	err := f.store.KeyDelete(ctx, roleKey)

	if err != nil {
		return fmt.Errorf("failed to delete role: %v", err)
	}

	return nil
}

// Shutdown closes the FoundationDB connection
func (f *FoundationDBRoleStore) Shutdown() {
	database := f.store.GetDatabase()
	if database != nil {
		database.Close()
	}
}

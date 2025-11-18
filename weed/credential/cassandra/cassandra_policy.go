package cassandra

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/s3api/policy_engine"
)

// GetPolicies retrieves all IAM policies from Cassandra
func (store *CassandraStore) GetPolicies(ctx context.Context) (map[string]policy_engine.PolicyDocument, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	query := fmt.Sprintf(`
		SELECT policy_name, policy_data FROM %s.%s_policies`, store.keyspace, store.tableName)

	iter := store.session.Query(query).
		WithContext(ctx).
		Iter()

	policies := make(map[string]policy_engine.PolicyDocument)
	var policyName, policyData string

	for iter.Scan(&policyName, &policyData) {
		var policy policy_engine.PolicyDocument
		err := json.Unmarshal([]byte(policyData), &policy)
		if err != nil {
			glog.V(1).Infof("Warning: failed to unmarshal policy %s: %v", policyName, err)
			continue
		}
		policies[policyName] = policy
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to get policies: %v", err)
	}

	return policies, nil
}

// CreatePolicy creates a new IAM policy in Cassandra
func (store *CassandraStore) CreatePolicy(ctx context.Context, name string, document policy_engine.PolicyDocument) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Check if policy already exists
	_, err := store.GetPolicy(ctx, name)
	if err == nil {
		return fmt.Errorf("policy %s already exists", name)
	}

	// Serialize policy to JSON
	policyData, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %v", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.%s_policies (policy_name, policy_data, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, store.keyspace, store.tableName)

	err = store.session.Query(query, name, string(policyData), time.Now(), time.Now()).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to create policy: %v", err)
	}

	glog.V(2).Infof("Created policy %s in Cassandra", name)
	return nil
}

// UpdatePolicy updates an existing IAM policy in Cassandra
func (store *CassandraStore) UpdatePolicy(ctx context.Context, name string, document policy_engine.PolicyDocument) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Check if policy exists
	_, err := store.GetPolicy(ctx, name)
	if err != nil {
		return fmt.Errorf("policy %s not found", name)
	}

	// Serialize updated policy
	policyData, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %v", err)
	}

	query := fmt.Sprintf(`
		UPDATE %s.%s_policies
		SET policy_data = ?, updated_at = ?
		WHERE policy_name = ?`, store.keyspace, store.tableName)

	err = store.session.Query(query, string(policyData), time.Now(), name).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to update policy: %v", err)
	}

	glog.V(2).Infof("Updated policy %s in Cassandra", name)
	return nil
}

// DeletePolicy removes an IAM policy from Cassandra
func (store *CassandraStore) DeletePolicy(ctx context.Context, name string) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Check if policy exists
	_, err := store.GetPolicy(ctx, name)
	if err != nil {
		return fmt.Errorf("policy %s not found", name)
	}

	query := fmt.Sprintf(`
		DELETE FROM %s.%s_policies
		WHERE policy_name = ?`, store.keyspace, store.tableName)

	err = store.session.Query(query, name).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to delete policy: %v", err)
	}

	glog.V(2).Infof("Deleted policy %s from Cassandra", name)
	return nil
}

// GetPolicy retrieves a specific IAM policy from Cassandra
func (store *CassandraStore) GetPolicy(ctx context.Context, name string) (*policy_engine.PolicyDocument, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	query := fmt.Sprintf(`
		SELECT policy_data FROM %s.%s_policies
		WHERE policy_name = ? LIMIT 1`, store.keyspace, store.tableName)

	var policyData string
	err := store.session.Query(query, name).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&policyData)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, fmt.Errorf("policy %s not found", name)
		}
		return nil, fmt.Errorf("failed to get policy: %v", err)
	}

	// Deserialize policy
	var policy policy_engine.PolicyDocument
	err = json.Unmarshal([]byte(policyData), &policy)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %v", err)
	}

	return &policy, nil
}

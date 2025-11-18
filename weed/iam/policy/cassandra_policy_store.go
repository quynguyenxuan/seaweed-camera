package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/seaweedfs/seaweedfs/weed/filer/cassandra2"
	"github.com/seaweedfs/seaweedfs/weed/glog"
)

// CassandraPolicyStore implements PolicyStore interface using Cassandra
type CassandraPolicyStore struct {
	session   *gocql.Session
	keyspace  string
	tableName string
}

// NewCassandraPolicyStore creates a new Cassandra-backed policy store
func NewCassandraPolicyStore(config map[string]interface{}) (PolicyStore, error) {
	if config == nil {
		return nil, fmt.Errorf("policy store config cannot be nil")
	}

	// Get keyspace
	keyspace := "sunfs"
	if ks, ok := config["keyspace"].(string); ok && ks != "" {
		keyspace = ks
	}

	// Get table name
	tableName := "policies"
	if tn, ok := config["table_name"].(string); ok && tn != "" {
		tableName = tn
	}

	// Set timeout
	// timeout := 5 * time.Second
	// if timeoutStr, ok := config["timeout"].(string); ok && timeoutStr != "" {
	// 	if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
	// 		timeout = parsedTimeout
	// 	}
	// }

	// Create cluster configuration
	filerStore := cassandra2.GetInstance()
	// cluster := filerStore.GetCluster()
	// cluster.Keyspace = keyspace
	// cluster.Timeout = timeout
	// cluster.Consistency = gocql.LocalQuorum

	// Connect to Cassandra
	// session, err := cluster.CreateSession()
	session := filerStore.GetSession()
	if session == nil {
		return nil, fmt.Errorf("failed to connect to Cassandra: %v")
	}

	// Create keyspace and table if they don't exist
	if err := createKeyspaceAndTable(session, keyspace, tableName); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create keyspace and table: %v", err)
	}

	store := &CassandraPolicyStore{
		session:   session,
		keyspace:  keyspace,
		tableName: tableName,
	}

	glog.V(0).Infof("Cassandra policy store initialized with keyspace: %s, table: %s", keyspace, tableName)
	return store, nil
}

// createKeyspaceAndTable creates the keyspace and table if they don't exist
func createKeyspaceAndTable(session *gocql.Session, keyspace, tableName string) error {
	// Create keyspace if it doesn't exist
	err := session.Query(fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s 
		WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }
	`, keyspace)).Exec()
	if err != nil {
		return fmt.Errorf("failed to create keyspace: %v", err)
	}

	// Create table if it doesn't exist
	err = session.Query(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			policy_name text PRIMARY KEY,
			policy_data text,
			created_at timestamp,
			updated_at timestamp
		)`, keyspace, tableName)).Exec()
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	return nil
}

// StorePolicy stores a policy document in Cassandra
func (c *CassandraPolicyStore) StorePolicy(ctx context.Context, filerAddress string, policyName string, policy *PolicyDocument) error {
	if policyName == "" {
		return fmt.Errorf("policy name cannot be empty")
	}
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	// Serialize policy to JSON
	policyData, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to serialize policy: %v", err)
	}

	// Store in Cassandra
	query := fmt.Sprintf(`
		INSERT INTO %s.%s (policy_name, policy_data, created_at, updated_at) 
		VALUES (?, ?, ?, ?)`, c.keyspace, c.tableName)

	err = c.session.Query(query, policyName, string(policyData), time.Now(), time.Now()).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to store policy: %v", err)
	}

	glog.V(2).Infof("Stored policy %s in Cassandra", policyName)
	return nil
}

// GetPolicy retrieves a policy document from Cassandra
func (c *CassandraPolicyStore) GetPolicy(ctx context.Context, filerAddress string, policyName string) (*PolicyDocument, error) {
	if policyName == "" {
		return nil, fmt.Errorf("policy name cannot be empty")
	}

	var policyData string
	var createdAt, updatedAt time.Time

	query := fmt.Sprintf(`
		SELECT policy_data, created_at, updated_at 
		FROM %s.%s 
		WHERE policy_name = ?`, c.keyspace, c.tableName)

	err := c.session.Query(query, policyName).
		WithContext(ctx).
		Scan(&policyData, &createdAt, &updatedAt)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, fmt.Errorf("policy '%s' not found", policyName)
		}
		return nil, fmt.Errorf("failed to retrieve policy: %v", err)
	}

	// Deserialize policy from JSON
	var policy PolicyDocument
	if err := json.Unmarshal([]byte(policyData), &policy); err != nil {
		return nil, fmt.Errorf("failed to deserialize policy: %v", err)
	}

	glog.V(2).Infof("Retrieved policy %s from Cassandra", policyName)
	return &policy, nil
}

// DeletePolicy deletes a policy document from Cassandra
func (c *CassandraPolicyStore) DeletePolicy(ctx context.Context, filerAddress string, policyName string) error {
	if policyName == "" {
		return fmt.Errorf("policy name cannot be empty")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s.%s 
		WHERE policy_name = ?`, c.keyspace, c.tableName)

	err := c.session.Query(query, policyName).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to delete policy: %v", err)
	}

	glog.V(2).Infof("Deleted policy %s from Cassandra", policyName)
	return nil
}

// ListPolicies lists all policy names from Cassandra
func (c *CassandraPolicyStore) ListPolicies(ctx context.Context, filerAddress string) ([]string, error) {
	var policyNames []string

	query := fmt.Sprintf(`
		SELECT policy_name 
		FROM %s.%s`, c.keyspace, c.tableName)

	iter := c.session.Query(query).
		WithContext(ctx).
		Iter()

	var policyName string
	for iter.Scan(&policyName) {
		policyNames = append(policyNames, policyName)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to list policies: %v", err)
	}

	glog.V(2).Infof("Listed %d policies from Cassandra", len(policyNames))
	return policyNames, nil
}

// Close closes the Cassandra connection
func (c *CassandraPolicyStore) Close() error {
	if c.session != nil {
		c.session.Close()
		glog.V(0).Infof("Cassandra policy store closed")
	}
	return nil
}

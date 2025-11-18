package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/seaweedfs/seaweedfs/weed/filer/cassandra2"
)

// CassandraRoleStore implements RoleStore interface using Apache Cassandra
type CassandraRoleStore struct {
	session   *gocql.Session
	keyspace  string
	tableName string
	timeout   time.Duration
	// consistency gocql.Consistency
	// filerStore    *cassandra2.Cassandra2Store
}

// NewCassandraRoleStore creates a new Cassandra-based role store
func NewCassandraRoleStore(config *RoleStoreConfig) (*CassandraRoleStore, error) {
	if config == nil {
		return nil, fmt.Errorf("role store config cannot be nil")
	}

	// Get keyspace
	keyspace := "sunfs"
	if ks, ok := config.StoreConfig["keyspace"].(string); ok && ks != "" {
		keyspace = ks
	}

	// Get table name
	tableName := "iam_roles"
	if tn, ok := config.StoreConfig["table_name"].(string); ok && tn != "" {
		tableName = tn
	}

	// Set timeout
	timeout := 5 * time.Second
	if timeoutStr, ok := config.StoreConfig["timeout"].(string); ok && timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	filerStore := cassandra2.GetInstance()
	cluster := filerStore.GetCluster()
	// consistency := cluster.Consistency

	// Create cluster configuration
	cluster.Keyspace = keyspace
	cluster.Timeout = timeout
	// cluster.Consistency = consistency

	// Create session
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create Cassandra session: %v", err)
	}

	// Create keyspace and table if they don't exist
	if err := createKeyspaceAndTable(session, keyspace, tableName); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create keyspace and table: %v", err)
	}

	return &CassandraRoleStore{
		session:   session,
		keyspace:  keyspace,
		tableName: tableName,
		timeout:   timeout,
	}, nil
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
			role_name text PRIMARY KEY,
			role_data text,
			created_at timestamp,
			updated_at timestamp
		)`, keyspace, tableName)).Exec()
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	return nil
}

// StoreRole stores a role definition in Cassandra
func (c *CassandraRoleStore) StoreRole(ctx context.Context, filerAddress string, roleName string, role *RoleDefinition) error {
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

	// Store in Cassandra
	query := fmt.Sprintf(`
		INSERT INTO %s.%s (role_name, role_data, created_at, updated_at) 
		VALUES (?, ?, ?, ?)`, c.keyspace, c.tableName)

	err = c.session.Query(query, roleName, string(roleData), time.Now(), time.Now()).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to store role: %v", err)
	}

	return nil
}

// GetRole retrieves a role definition from Cassandra
func (c *CassandraRoleStore) GetRole(ctx context.Context, filerAddress string, roleName string) (*RoleDefinition, error) {
	if roleName == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	var roleData string
	var createdAt, updatedAt time.Time

	query := fmt.Sprintf(`
		SELECT role_data, created_at, updated_at 
		FROM %s.%s 
		WHERE role_name = ?`, c.keyspace, c.tableName)

	err := c.session.Query(query, roleName).
		WithContext(ctx).
		Scan(&roleData, &createdAt, &updatedAt)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, fmt.Errorf("role '%s' not found", roleName)
		}
		return nil, fmt.Errorf("failed to retrieve role: %v", err)
	}

	// Deserialize role from JSON
	var role RoleDefinition
	if err := json.Unmarshal([]byte(roleData), &role); err != nil {
		return nil, fmt.Errorf("failed to deserialize role: %v", err)
	}

	return &role, nil
}

// ListRoles lists all role names for a given filer address from Cassandra
func (c *CassandraRoleStore) ListRoles(ctx context.Context, filerAddress string) ([]string, error) {
	var roleNames []string

	query := fmt.Sprintf(`
		SELECT role_name 
		FROM %s.%s`, c.keyspace, c.tableName)

	iter := c.session.Query(query).
		WithContext(ctx).
		Iter()

	var roleName string
	for iter.Scan(&roleName) {
		roleNames = append(roleNames, roleName)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to list roles: %v", err)
	}

	return roleNames, nil
}

// DeleteRole deletes a role definition from Cassandra
func (c *CassandraRoleStore) DeleteRole(ctx context.Context, filerAddress string, roleName string) error {
	if roleName == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s.%s 
		WHERE role_name = ?`, c.keyspace, c.tableName)

	err := c.session.Query(query, roleName).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to delete role: %v", err)
	}

	return nil
}

// Shutdown closes the Cassandra session
func (c *CassandraRoleStore) Shutdown() {
	if c.session != nil {
		c.session.Close()
	}
}

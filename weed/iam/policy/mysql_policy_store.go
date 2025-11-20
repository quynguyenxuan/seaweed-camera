package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/seaweedfs/seaweedfs/weed/glog"
)

// MysqlPolicyStore implements PolicyStore interface using MySQL
type MysqlPolicyStore struct {
	mu        sync.RWMutex
	db        *sql.DB
	database  string
	tableName string
}

// NewMysqlPolicyStore creates a new MySQL-backed policy store
func NewMysqlPolicyStore(config map[string]interface{}) (PolicyStore, error) {
	if config == nil {
		return nil, fmt.Errorf("policy store config cannot be nil")
	}

	// Get database connection details
	hostname := "localhost"
	if h, ok := config["hostname"].(string); ok && h != "" {
		hostname = h
	}

	port := 3306
	if p, ok := config["port"].(int); ok && p > 0 {
		port = p
	}

	username := "root"
	if u, ok := config["username"].(string); ok && u != "" {
		username = u
	}

	password := ""
	if p, ok := config["password"].(string); ok {
		password = p
	}

	database := "sunfs"
	if db, ok := config["database"].(string); ok && db != "" {
		database = db
	}

	// Get table name
	tableName := "policies"
	if tn, ok := config["table_name"].(string); ok && tn != "" {
		tableName = tn
	}

	// Set timeout
	timeout := 5 * time.Second
	if timeoutStr, ok := config["timeout"].(string); ok && timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	// Build connection string
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%s&parseTime=true",
		username, password, hostname, port, database, timeout.String())

	// Connect to MySQL
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL at %s:%d: %v", hostname, port, err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Create table if it doesn't exist
	if err := createTable(db, tableName); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table '%s': %v", tableName, err)
	}

	store := &MysqlPolicyStore{
		db:        db,
		database:  database,
		tableName: tableName,
	}

	glog.V(0).Infof("MySQL policy store initialized with database: %s, table: %s", database, tableName)
	return store, nil
}

// createTable creates the policies table if it doesn't exist
func createTable(db *sql.DB, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			policy_name VARCHAR(255) PRIMARY KEY,
			policy_data JSON NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, tableName)

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	return nil
}

// StorePolicy stores a policy document in MySQL
func (m *MysqlPolicyStore) StorePolicy(ctx context.Context, filerAddress string, policyName string, policy *PolicyDocument) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

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

	// Store in MySQL using INSERT ... ON DUPLICATE KEY UPDATE
	query := fmt.Sprintf(`
		INSERT INTO %s (policy_name, policy_data, created_at, updated_at) 
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
		policy_data = VALUES(policy_data),
		updated_at = VALUES(updated_at)
	`, m.tableName)

	_, err = m.db.ExecContext(ctx, query, policyName, string(policyData), time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to store policy: %v", err)
	}

	glog.V(2).Infof("Stored policy %s in MySQL", policyName)
	return nil
}

// GetPolicy retrieves a policy document from MySQL
func (m *MysqlPolicyStore) GetPolicy(ctx context.Context, filerAddress string, policyName string) (*PolicyDocument, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if policyName == "" {
		return nil, fmt.Errorf("policy name cannot be empty")
	}

	var policyData string
	var createdAt, updatedAt time.Time

	query := fmt.Sprintf(`
		SELECT policy_data, created_at, updated_at 
		FROM %s 
		WHERE policy_name = ?
	`, m.tableName)

	err := m.db.QueryRowContext(ctx, query, policyName).Scan(&policyData, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("policy '%s' not found", policyName)
		}
		return nil, fmt.Errorf("failed to retrieve policy: %v", err)
	}

	// Deserialize policy from JSON
	var policy PolicyDocument
	if err := json.Unmarshal([]byte(policyData), &policy); err != nil {
		return nil, fmt.Errorf("failed to deserialize policy: %v", err)
	}

	glog.V(2).Infof("Retrieved policy %s from MySQL", policyName)
	return &policy, nil
}

// DeletePolicy deletes a policy document from MySQL
func (m *MysqlPolicyStore) DeletePolicy(ctx context.Context, filerAddress string, policyName string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if policyName == "" {
		return fmt.Errorf("policy name cannot be empty")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE policy_name = ?
	`, m.tableName)

	result, err := m.db.ExecContext(ctx, query, policyName)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		glog.V(2).Infof("Policy %s not found for deletion in MySQL", policyName)
	} else {
		glog.V(2).Infof("Deleted policy %s from MySQL", policyName)
	}

	return nil
}

// ListPolicies lists all policy names from MySQL
func (m *MysqlPolicyStore) ListPolicies(ctx context.Context, filerAddress string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var policyNames []string

	query := fmt.Sprintf(`
		SELECT policy_name 
		FROM %s 
		ORDER BY policy_name
	`, m.tableName)

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var policyName string
		if err := rows.Scan(&policyName); err != nil {
			return nil, fmt.Errorf("failed to scan policy name: %v", err)
		}
		policyNames = append(policyNames, policyName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policies: %v", err)
	}

	glog.V(2).Infof("Listed %d policies from MySQL", len(policyNames))
	return policyNames, nil
}

// Close closes the MySQL connection
func (m *MysqlPolicyStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		err := m.db.Close()
		m.db = nil
		if err != nil {
			glog.V(0).Infof("Error closing MySQL policy store: %v", err)
			return err
		}
		glog.V(0).Infof("MySQL policy store closed")
	}
	return nil
}

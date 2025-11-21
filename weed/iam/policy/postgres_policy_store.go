package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/seaweedfs/seaweedfs/weed/glog"
)

// PostgresPolicyStore implements PolicyStore interface using PostgreSQL
type PostgresPolicyStore struct {
	mu        sync.RWMutex
	db        *sql.DB
	database  string
	schema    string
	tableName string
}

// NewPostgresPolicyStore creates a new PostgreSQL-backed policy store
func NewPostgresPolicyStore(config map[string]interface{}) (PolicyStore, error) {
	if config == nil {
		return nil, fmt.Errorf("policy store config cannot be nil")
	}

	// Get database connection details
	hostname := "localhost"
	if h, ok := config["hostname"].(string); ok && h != "" {
		hostname = h
	}

	port := 5432
	if p, ok := config["port"].(int); ok && p > 0 {
		port = p
	} else if p, ok := config["port"].(float64); ok && p > 0 {
		port = int(p)
	}

	username := "postgres"
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

	schema := "public"
	if s, ok := config["schema"].(string); ok && s != "" {
		schema = s
	}

	sslmode := "disable"
	if s, ok := config["sslmode"].(string); ok && s != "" {
		sslmode = s
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
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s connect_timeout=%d",
		hostname, port, username, password, database, sslmode, schema, int(timeout.Seconds()))

	// Connect to PostgreSQL
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL at %s:%d: %v", hostname, port, err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Create table if it doesn't exist
	if err := createPolicyTable(db, tableName); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table '%s': %v", tableName, err)
	}

	store := &PostgresPolicyStore{
		db:        db,
		database:  database,
		schema:    schema,
		tableName: tableName,
	}

	glog.V(0).Infof("PostgreSQL policy store initialized with database: %s, schema: %s, table: %s", database, schema, tableName)
	return store, nil
}

// createPolicyTable creates the policies table if it doesn't exist
func createPolicyTable(db *sql.DB, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			policy_name VARCHAR(255) PRIMARY KEY,
			policy_data JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, tableName)

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	// Create GIN index on policy_data for better query performance
	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_policy_data ON %s USING GIN (policy_data)
	`, tableName, tableName)

	if _, err := db.Exec(indexQuery); err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	return nil
}

// StorePolicy stores a policy document in PostgreSQL
func (p *PostgresPolicyStore) StorePolicy(ctx context.Context, filerAddress string, policyName string, policy *PolicyDocument) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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

	// Store in PostgreSQL using INSERT ... ON CONFLICT DO UPDATE
	query := fmt.Sprintf(`
		INSERT INTO %s (policy_name, policy_data, created_at, updated_at) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (policy_name) DO UPDATE 
		SET policy_data = EXCLUDED.policy_data,
		    updated_at = EXCLUDED.updated_at
	`, p.tableName)

	_, err = p.db.ExecContext(ctx, query, policyName, string(policyData), time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to store policy: %v", err)
	}

	glog.V(2).Infof("Stored policy %s in PostgreSQL", policyName)
	return nil
}

// GetPolicy retrieves a policy document from PostgreSQL
func (p *PostgresPolicyStore) GetPolicy(ctx context.Context, filerAddress string, policyName string) (*PolicyDocument, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if policyName == "" {
		return nil, fmt.Errorf("policy name cannot be empty")
	}

	var policyData string
	var createdAt, updatedAt time.Time

	query := fmt.Sprintf(`
		SELECT policy_data, created_at, updated_at 
		FROM %s 
		WHERE policy_name = $1
	`, p.tableName)

	err := p.db.QueryRowContext(ctx, query, policyName).Scan(&policyData, &createdAt, &updatedAt)
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

	glog.V(2).Infof("Retrieved policy %s from PostgreSQL", policyName)
	return &policy, nil
}

// DeletePolicy deletes a policy document from PostgreSQL
func (p *PostgresPolicyStore) DeletePolicy(ctx context.Context, filerAddress string, policyName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if policyName == "" {
		return fmt.Errorf("policy name cannot be empty")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE policy_name = $1
	`, p.tableName)

	result, err := p.db.ExecContext(ctx, query, policyName)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		glog.V(2).Infof("Policy %s not found for deletion in PostgreSQL", policyName)
	} else {
		glog.V(2).Infof("Deleted policy %s from PostgreSQL", policyName)
	}

	return nil
}

// ListPolicies lists all policy names from PostgreSQL
func (p *PostgresPolicyStore) ListPolicies(ctx context.Context, filerAddress string) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var policyNames []string

	query := fmt.Sprintf(`
		SELECT policy_name 
		FROM %s 
		ORDER BY policy_name
	`, p.tableName)

	rows, err := p.db.QueryContext(ctx, query)
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

	glog.V(2).Infof("Listed %d policies from PostgreSQL", len(policyNames))
	return policyNames, nil
}

// Close closes the PostgreSQL connection
func (p *PostgresPolicyStore) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db != nil {
		err := p.db.Close()
		p.db = nil
		if err != nil {
			glog.V(0).Infof("Error closing PostgreSQL policy store: %v", err)
			return err
		}
		glog.V(0).Infof("PostgreSQL policy store closed")
	}
	return nil
}

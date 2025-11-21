package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresRoleStore implements RoleStore interface using PostgreSQL
type PostgresRoleStore struct {
	mu        sync.RWMutex
	db        *sql.DB
	database  string
	schema    string
	tableName string
	timeout   time.Duration
}

// NewPostgresRoleStore creates a new PostgreSQL-based role store
func NewPostgresRoleStore(config map[string]interface{}) (*PostgresRoleStore, error) {
	if config == nil {
		return nil, fmt.Errorf("role store config cannot be nil")
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

	database := "sunfs"
	if d, ok := config["database"].(string); ok && d != "" {
		database = d
	}

	username := "postgres"
	if u, ok := config["username"].(string); ok && u != "" {
		username = u
	}

	password := ""
	if p, ok := config["password"].(string); ok && p != "" {
		password = p
	}

	schema := "public"
	if s, ok := config["schema"].(string); ok && s != "" {
		schema = s
	}

	sslmode := "disable"
	if s, ok := config["sslmode"].(string); ok && s != "" {
		sslmode = s
	}

	tableName := "iam_roles"
	if t, ok := config["table_name"].(string); ok && t != "" {
		tableName = t
	}

	timeout := 30 * time.Second
	if t, ok := config["timeout"].(time.Duration); ok && t > 0 {
		timeout = t
	} else if t, ok := config["timeout"].(string); ok && t != "" {
		if parsed, err := time.ParseDuration(t); err == nil {
			timeout = parsed
		}
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s",
		hostname, port, username, password, database, sslmode, schema)

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
	if err := createPostgresRoleTable(db, tableName); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	return &PostgresRoleStore{
		db:        db,
		database:  database,
		schema:    schema,
		tableName: tableName,
		timeout:   timeout,
	}, nil
}

// createPostgresRoleTable creates the roles table if it doesn't exist
func createPostgresRoleTable(db *sql.DB, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			role_name VARCHAR(255) PRIMARY KEY,
			role_data JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, tableName)

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	// Create index on role_data for better query performance
	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_role_data ON %s USING GIN (role_data)
	`, tableName, tableName)

	if _, err := db.Exec(indexQuery); err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	return nil
}

// StoreRole stores a role definition in PostgreSQL
func (p *PostgresRoleStore) StoreRole(ctx context.Context, filerAddress string, roleName string, role *RoleDefinition) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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

	// Store in PostgreSQL using INSERT ... ON CONFLICT DO UPDATE
	query := fmt.Sprintf(`
		INSERT INTO %s (role_name, role_data, created_at, updated_at) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (role_name) DO UPDATE 
		SET role_data = EXCLUDED.role_data,
		    updated_at = EXCLUDED.updated_at
	`, p.tableName)

	now := time.Now()
	_, err = p.db.ExecContext(ctx, query, roleName, string(roleData), now, now)
	if err != nil {
		return fmt.Errorf("failed to store role: %v", err)
	}

	return nil
}

// GetRole retrieves a role definition from PostgreSQL
func (p *PostgresRoleStore) GetRole(ctx context.Context, filerAddress string, roleName string) (*RoleDefinition, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if roleName == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	var roleData string
	var createdAt, updatedAt time.Time

	query := fmt.Sprintf(`
		SELECT role_data, created_at, updated_at 
		FROM %s 
		WHERE role_name = $1
	`, p.tableName)

	err := p.db.QueryRowContext(ctx, query, roleName).Scan(&roleData, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
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

// ListRoles lists all role names for a given filer address from PostgreSQL
func (p *PostgresRoleStore) ListRoles(ctx context.Context, filerAddress string) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var roleNames []string

	query := fmt.Sprintf(`
		SELECT role_name 
		FROM %s 
		ORDER BY role_name
	`, p.tableName)

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, fmt.Errorf("failed to scan role name: %v", err)
		}
		roleNames = append(roleNames, roleName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %v", err)
	}

	return roleNames, nil
}

// DeleteRole deletes a role definition from PostgreSQL
func (p *PostgresRoleStore) DeleteRole(ctx context.Context, filerAddress string, roleName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if roleName == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE role_name = $1
	`, p.tableName)

	result, err := p.db.ExecContext(ctx, query, roleName)
	if err != nil {
		return fmt.Errorf("failed to delete role: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		// Role not found - not an error for idempotent deletion
	} else {
		// Role was deleted
	}

	return nil
}

// Shutdown closes the PostgreSQL connection
func (p *PostgresRoleStore) Shutdown() {
	if p.db != nil {
		p.db.Close()
		p.db = nil
	}
}

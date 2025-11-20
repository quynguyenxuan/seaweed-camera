package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MysqlRoleStore implements RoleStore interface using MySQL
type MysqlRoleStore struct {
	mu        sync.RWMutex
	db        *sql.DB
	database  string
	tableName string
	timeout   time.Duration
}

// NewMysqlRoleStore creates a new MySQL-based role store
func NewMysqlRoleStore(config map[string]interface{}) (*MysqlRoleStore, error) {
	if config == nil {
		return nil, fmt.Errorf("role store config cannot be nil")
	}

	// Get database connection details
	hostname := "localhost"
	if h, ok := config["hostname"].(string); ok && h != "" {
		hostname = h
	}

	port := 3306
	if p, ok := config["port"].(int); ok && p > 0 {
		port = p
	} else if p, ok := config["port"].(float64); ok && p > 0 {
		port = int(p)
	}

	database := "sunfs"
	if d, ok := config["database"].(string); ok && d != "" {
		database = d
	}

	username := "root"
	if u, ok := config["username"].(string); ok && u != "" {
		username = u
	}

	password := ""
	if p, ok := config["password"].(string); ok && p != "" {
		password = p
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
	if err := createRoleTable(db, tableName); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	return &MysqlRoleStore{
		db:        db,
		database:  database,
		tableName: tableName,
		timeout:   timeout,
	}, nil
}

// createRoleTable creates the roles table if it doesn't exist
func createRoleTable(db *sql.DB, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			role_name VARCHAR(255) PRIMARY KEY,
			role_data JSON NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, tableName)

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	return nil
}

// StoreRole stores a role definition in MySQL
func (m *MysqlRoleStore) StoreRole(ctx context.Context, filerAddress string, roleName string, role *RoleDefinition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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

	// Store in MySQL using INSERT ... ON DUPLICATE KEY UPDATE
	query := fmt.Sprintf(`
		INSERT INTO %s (role_name,role_data,created_at,updated_at) 
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
		role_data = ?,
		updated_at = ?
	`, m.tableName)

	now := time.Now()
	_, err = m.db.ExecContext(ctx, query, roleName, string(roleData), now, now, string(roleData), now)
	if err != nil {
		return fmt.Errorf("failed to store role: %v", err)
	}

	return nil
}

// GetRole retrieves a role definition from MySQL
func (m *MysqlRoleStore) GetRole(ctx context.Context, filerAddress string, roleName string) (*RoleDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if roleName == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	var roleData string
	var createdAt, updatedAt time.Time

	query := fmt.Sprintf(`
		SELECT role_data,created_at,updated_at 
		FROM %s 
		WHERE role_name = ?
	`, m.tableName)

	err := m.db.QueryRowContext(ctx, query, roleName).Scan(&roleData, &createdAt, &updatedAt)
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

// ListRoles lists all role names for a given filer address from MySQL
func (m *MysqlRoleStore) ListRoles(ctx context.Context, filerAddress string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var roleNames []string

	query := fmt.Sprintf(`
		SELECT role_name 
		FROM %s 
		ORDER BY role_name
	`, m.tableName)

	rows, err := m.db.QueryContext(ctx, query)
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

// DeleteRole deletes a role definition from MySQL
func (m *MysqlRoleStore) DeleteRole(ctx context.Context, filerAddress string, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if roleName == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE role_name = ?
	`, m.tableName)

	result, err := m.db.ExecContext(ctx, query, roleName)
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

// Shutdown closes the MySQL connection
func (m *MysqlRoleStore) Shutdown() {
	if m.db != nil {
		m.db.Close()
		m.db = nil
	}
}

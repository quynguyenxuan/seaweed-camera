package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/credential/bunsql"
	"github.com/uptrace/bun"
)

// BunSqlRoleStore implements RoleStore interface using Bun
type BunSqlRoleStore struct {
	mu        sync.RWMutex
	db        *bun.DB
	database  string
	driver    string
	tableName string
	timeout   time.Duration
}

type RoleModel struct {
	bun.BaseModel `bun:"table:iam_roles,alias:r"`

	RoleName  string    `bun:"role_name,pk"`
	RoleData  string    `bun:"role_data,type:jsonb"` // JSONB content as string
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

// NewBunSqlRoleStore creates a new Bun-based role store
func NewBunSqlRoleStore(config map[string]interface{}) (*BunSqlRoleStore, error) {
	if config == nil {
		return nil, fmt.Errorf("role store config cannot be nil")
	}

	//QUYNGUYEN Use shared helper function to convert config map to DatabaseConfig
	dbConfig := bunsql.DatabaseConfigFromMap(config)

	// Get table name
	tableName := "iam_roles"
	if t, ok := config["table_name"].(string); ok && t != "" {
		tableName = t
	}

	db, err := bunsql.ConnectToDatabase(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	//QUYNGUYEN end

	store := &BunSqlRoleStore{
		db:        db,
		database:  dbConfig.Database,
		driver:    dbConfig.Driver,
		tableName: tableName,
		timeout:   dbConfig.Timeout,
	}

	// Create table if it doesn't exist
	if err := store.createRoleTable(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	return store, nil
}

func (m *BunSqlRoleStore) createRoleTable() error {
	ctx := context.Background()
	_, err := m.db.NewCreateTable().Model((*RoleModel)(nil)).ModelTableExpr(m.tableName).IfNotExists().Exec(ctx)
	return err
}

// StoreRole stores a role definition in MySQL
func (m *BunSqlRoleStore) StoreRole(ctx context.Context, filerAddress string, roleName string, role *RoleDefinition) error {
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

	roleModel := &RoleModel{
		RoleName:  roleName,
		RoleData:  string(roleData), // Convert to string to match postgres implementation
		UpdatedAt: time.Now(),
	}

	// Upsert with database-specific syntax
	if m.driver == "postgresql" {
		// PostgreSQL syntax
		_, err = m.db.NewInsert().Model(roleModel).ModelTableExpr(m.tableName + " AS r").
			On("CONFLICT (role_name) DO UPDATE").
			Set("role_data = EXCLUDED.role_data").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
	} else {
		// MySQL syntax (fallback)
		_, err = m.db.NewInsert().Model(roleModel).ModelTableExpr(m.tableName + " AS r").
			On("DUPLICATE KEY UPDATE").
			Set("role_data = EXCLUDED.role_data").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to store role: %v", err)
	}

	return nil
}

// GetRole retrieves a role definition from MySQL
func (m *BunSqlRoleStore) GetRole(ctx context.Context, filerAddress string, roleName string) (*RoleDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if roleName == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	var roleModel RoleModel
	err := m.db.NewSelect().Model(&roleModel).ModelTableExpr(m.tableName+" AS r").Where("role_name = ?", roleName).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role '%s' not found", roleName)
		}
		return nil, fmt.Errorf("failed to retrieve role: %v", err)
	}

	// Deserialize role from JSON
	var role RoleDefinition

	// Check if the data is double-encoded (JSON string within JSON)
	if strings.HasPrefix(roleModel.RoleData, "\"") && strings.HasSuffix(roleModel.RoleData, "\"") {
		// Data is stored as JSON string, need to unmarshal twice
		var jsonString string
		if err := json.Unmarshal([]byte(roleModel.RoleData), &jsonString); err != nil {
			return nil, fmt.Errorf("failed to deserialize role string: %v, data was: %s", err, roleModel.RoleData)
		}
		// Now unmarshal the actual JSON content
		if err := json.Unmarshal([]byte(jsonString), &role); err != nil {
			return nil, fmt.Errorf("failed to deserialize role content: %v, data was: %s", err, jsonString)
		}
	} else {
		// Data is stored as regular JSON, unmarshal directly
		if err := json.Unmarshal([]byte(roleModel.RoleData), &role); err != nil {
			return nil, fmt.Errorf("failed to deserialize role: %v, data was: %s", err, roleModel.RoleData)
		}
	}

	return &role, nil
}

// ListRoles lists all role names for a given filer address from MySQL
func (m *BunSqlRoleStore) ListRoles(ctx context.Context, filerAddress string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var roleNames []string
	err := m.db.NewSelect().Model((*RoleModel)(nil)).ModelTableExpr(m.tableName+" AS r").Column("role_name").Order("role_name ASC").Scan(ctx, &roleNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %v", err)
	}

	return roleNames, nil
}

// DeleteRole deletes a role definition from MySQL
func (m *BunSqlRoleStore) DeleteRole(ctx context.Context, filerAddress string, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if roleName == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	_, err := m.db.NewDelete().Model((*RoleModel)(nil)).ModelTableExpr(m.tableName+" AS r").Where("role_name = ?", roleName).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete role: %v", err)
	}

	return nil
}

// Shutdown closes the MySQL connection
func (m *BunSqlRoleStore) Shutdown() {
	if m.db != nil {
		m.db.Close()
		m.db = nil
	}
}

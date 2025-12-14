package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/credential/bunsql"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/uptrace/bun"
)

// BunSqlPolicyStore implements PolicyStore interface using Bun
type BunSqlPolicyStore struct {
	mu        sync.RWMutex
	db        *bun.DB
	database  string
	driver    string
	tableName string
}

type PolicyModel struct {
	bun.BaseModel `bun:"table:sts_policies,alias:p"`

	PolicyName string    `bun:"policy_name,pk"`
	PolicyData string    `bun:"policy_data,type:jsonb"` // JSONB content as string
	CreatedAt  time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt  time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

// NewBunSqlPolicyStore creates a new Bun-based policy store
func NewBunSqlPolicyStore(config map[string]interface{}) (PolicyStore, error) {
	if config == nil {
		return nil, fmt.Errorf("policy store config cannot be nil")
	}

	//QUYNGUYEN Use shared helper function to convert config map to DatabaseConfig
	dbConfig := bunsql.DatabaseConfigFromMap(config)

	// Get table name
	tableName := "policies"
	if tn, ok := config["table_name"].(string); ok && tn != "" {
		tableName = tn
	}

	db, err := bunsql.ConnectToDatabase(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	//QUYNGUYEN end

	store := &BunSqlPolicyStore{
		db:        db,
		database:  dbConfig.Database,
		driver:    dbConfig.Driver,
		tableName: tableName,
	}

	// Create table if it doesn't exist
	if err := store.createTable(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table '%s': %v", tableName, err)
	}

	glog.V(0).Infof("BunSQL policy store initialized with database: %s, table: %s", dbConfig.Database, tableName)
	return store, nil
}

// createTable creates the policies table if it doesn't exist
func (m *BunSqlPolicyStore) createTable() error {
	ctx := context.Background()
	_, err := m.db.NewCreateTable().Model((*PolicyModel)(nil)).ModelTableExpr(m.tableName).IfNotExists().Exec(ctx)
	return err
}

// StorePolicy stores a policy document in MySQL
func (m *BunSqlPolicyStore) StorePolicy(ctx context.Context, filerAddress string, policyName string, policy *PolicyDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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

	policyModel := &PolicyModel{
		PolicyName: policyName,
		PolicyData: string(policyData), // Convert to string to match role store implementation
		UpdatedAt:  time.Now(),
	}

	// Debug: Log table name and driver
	glog.V(4).Infof("Storing policy '%s' in table '%s' using driver '%s'", policyName, m.tableName, m.driver)

	// Upsert with database-specific syntax
	if m.driver == "postgresql" {
		// PostgreSQL syntax
		glog.V(4).Infof("Using PostgreSQL syntax with table expression: %s AS p", m.tableName)
		_, err = m.db.NewInsert().Model(policyModel).ModelTableExpr(m.tableName + " AS p").
			On("CONFLICT (policy_name) DO UPDATE").
			Set("policy_data = EXCLUDED.policy_data").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
	} else {
		// MySQL syntax (fallback)
		glog.V(4).Infof("Using MySQL syntax with table expression: %s AS p", m.tableName)
		_, err = m.db.NewInsert().Model(policyModel).ModelTableExpr(m.tableName + " AS p").
			On("DUPLICATE KEY UPDATE").
			Set("policy_data = EXCLUDED.policy_data").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to store policy: %v", err)
	}

	glog.V(2).Infof("Stored policy %s in BunSQL", policyName)
	return nil
}

// GetPolicy retrieves a policy document from MySQL
func (m *BunSqlPolicyStore) GetPolicy(ctx context.Context, filerAddress string, policyName string) (*PolicyDocument, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if policyName == "" {
		return nil, fmt.Errorf("policy name cannot be empty")
	}

	var policyModel PolicyModel
	err := m.db.NewSelect().Model(&policyModel).ModelTableExpr(m.tableName+" AS p").Where("policy_name = ?", policyName).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("policy '%s' not found", policyName)
		}
		return nil, fmt.Errorf("failed to retrieve policy: %v", err)
	}

	// Deserialize policy from JSON
	var policy PolicyDocument

	// Check if the data is double-encoded (JSON string within JSON)
	if strings.HasPrefix(policyModel.PolicyData, "\"") && strings.HasSuffix(policyModel.PolicyData, "\"") {
		// Data is stored as JSON string, need to unmarshal twice
		var jsonString string
		if err := json.Unmarshal([]byte(policyModel.PolicyData), &jsonString); err != nil {
			return nil, fmt.Errorf("failed to deserialize policy string: %v, data was: %s", err, policyModel.PolicyData)
		}
		// Now unmarshal the actual JSON content
		if err := json.Unmarshal([]byte(jsonString), &policy); err != nil {
			return nil, fmt.Errorf("failed to deserialize policy content: %v, data was: %s", err, jsonString)
		}
	} else {
		// Data is stored as regular JSON, unmarshal directly
		if err := json.Unmarshal([]byte(policyModel.PolicyData), &policy); err != nil {
			return nil, fmt.Errorf("failed to deserialize policy: %v, data was: %s", err, policyModel.PolicyData)
		}
	}

	glog.V(2).Infof("Retrieved policy %s from BunSQL", policyName)
	return &policy, nil
}

// DeletePolicy deletes a policy document from MySQL
func (m *BunSqlPolicyStore) DeletePolicy(ctx context.Context, filerAddress string, policyName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if policyName == "" {
		return fmt.Errorf("policy name cannot be empty")
	}

	res, err := m.db.NewDelete().Model((*PolicyModel)(nil)).ModelTableExpr(m.tableName+" AS p").Where("policy_name = ?", policyName).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %v", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		glog.V(2).Infof("Policy %s not found for deletion in BunSQL", policyName)
	} else {
		glog.V(2).Infof("Deleted policy %s from BunSQL", policyName)
	}

	return nil
}

// ListPolicies lists all policy names from MySQL
func (m *BunSqlPolicyStore) ListPolicies(ctx context.Context, filerAddress string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var policyNames []string
	err := m.db.NewSelect().Model((*PolicyModel)(nil)).ModelTableExpr(m.tableName+" AS p").Column("policy_name").Order("policy_name ASC").Scan(ctx, &policyNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %v", err)
	}

	glog.V(2).Infof("Listed %d policies from BunSQL", len(policyNames))
	return policyNames, nil
}

// Close closes the MySQL connection
func (m *BunSqlPolicyStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		err := m.db.Close()
		m.db = nil
		if err != nil {
			glog.V(0).Infof("Error closing BunSQL policy store: %v", err)
			return err
		}
		glog.V(0).Infof("BunSQL policy store closed")
	}
	return nil
}

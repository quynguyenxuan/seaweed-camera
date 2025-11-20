package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/seaweedfs/seaweedfs/weed/s3api/policy_engine"
)

// GetPolicies retrieves all IAM policies from MySQL
func (store *MysqlStore) GetPolicies(ctx context.Context) (map[string]policy_engine.PolicyDocument, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	policies := make(map[string]policy_engine.PolicyDocument)

	rows, err := store.db.QueryContext(ctx, "SELECT name, document FROM policies ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var documentJSON []byte

		if err := rows.Scan(&name, &documentJSON); err != nil {
			return nil, fmt.Errorf("failed to scan policy row: %w", err)
		}

		var document policy_engine.PolicyDocument
		if err := json.Unmarshal(documentJSON, &document); err != nil {
			return nil, fmt.Errorf("failed to unmarshal policy document for %s: %v", name, err)
		}

		policies[name] = document
	}

	return policies, nil
}

// CreatePolicy creates a new IAM policy in MySQL
func (store *MysqlStore) CreatePolicy(ctx context.Context, name string, document policy_engine.PolicyDocument) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	documentJSON, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal policy document: %w", err)
	}

	_, err = store.db.ExecContext(ctx,
		"INSERT INTO policies (name, document) VALUES (?, ?) ON DUPLICATE KEY UPDATE document = VALUES(document), updated_at = CURRENT_TIMESTAMP",
		name, documentJSON)
	if err != nil {
		return fmt.Errorf("failed to insert policy: %w", err)
	}

	return nil
}

// UpdatePolicy updates an existing IAM policy in MySQL
func (store *MysqlStore) UpdatePolicy(ctx context.Context, name string, document policy_engine.PolicyDocument) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	documentJSON, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal policy document: %w", err)
	}

	result, err := store.db.ExecContext(ctx,
		"UPDATE policies SET document = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?",
		documentJSON, name)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy %s not found", name)
	}

	return nil
}

// DeletePolicy deletes an IAM policy from MySQL
func (store *MysqlStore) DeletePolicy(ctx context.Context, name string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	result, err := store.db.ExecContext(ctx, "DELETE FROM policies WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy %s not found", name)
	}

	return nil
}

// GetPolicy retrieves a specific IAM policy by name from MySQL
func (store *MysqlStore) GetPolicy(ctx context.Context, name string) (*policy_engine.PolicyDocument, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	var documentJSON []byte
	err := store.db.QueryRowContext(ctx, "SELECT document FROM policies WHERE name = ?", name).Scan(&documentJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Policy not found
		}
		return nil, fmt.Errorf("failed to query policy: %w", err)
	}

	var document policy_engine.PolicyDocument
	if err := json.Unmarshal(documentJSON, &document); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy document for %s: %v", name, err)
	}

	return &document, nil
}

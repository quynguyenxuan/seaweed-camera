package bunsql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/credential"
	"github.com/seaweedfs/seaweedfs/weed/s3api/policy_engine"
	"github.com/uptrace/bun"
)

type Policy struct {
	bun.BaseModel `bun:"table:policies,alias:p"`

	Name      string    `bun:"name,pk,notnull"`
	Document  []byte    `bun:"document,notnull"` // JSONB
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

var _ credential.PolicyManager = (*BunSqlStore)(nil)

func (store *BunSqlStore) GetPolicies(ctx context.Context) (map[string]policy_engine.PolicyDocument, error) {
	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	var policies []Policy
	err := store.db.NewSelect().Model(&policies).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}

	result := make(map[string]policy_engine.PolicyDocument)
	for _, p := range policies {
		var doc policy_engine.PolicyDocument
		if err := json.Unmarshal(p.Document, &doc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal policy %s: %w", p.Name, err)
		}
		result[p.Name] = doc
	}

	return result, nil
}

func (store *BunSqlStore) CreatePolicy(ctx context.Context, name string, document policy_engine.PolicyDocument) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	exists, err := store.db.NewSelect().Model((*Policy)(nil)).Where("name = ?", name).Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("policy already exists")
	}

	docBytes, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal policy document: %w", err)
	}

	policy := &Policy{
		Name:     name,
		Document: docBytes,
	}

	_, err = store.db.NewInsert().Model(policy).Exec(ctx)
	return err
}

func (store *BunSqlStore) UpdatePolicy(ctx context.Context, name string, document policy_engine.PolicyDocument) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	exists, err := store.db.NewSelect().Model((*Policy)(nil)).Where("name = ?", name).Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("policy not found")
	}

	docBytes, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal policy document: %w", err)
	}

	policy := &Policy{
		Name:      name,
		Document:  docBytes,
		UpdatedAt: time.Now(),
	}

	_, err = store.db.NewUpdate().Model(policy).Column("document", "updated_at").Where("name = ?", name).Exec(ctx)
	return err
}

func (store *BunSqlStore) DeletePolicy(ctx context.Context, name string) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	res, err := store.db.NewDelete().Model((*Policy)(nil)).Where("name = ?", name).Exec(ctx)
	if err != nil {
		return err
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("policy not found")
	}

	return nil
}

func (store *BunSqlStore) GetPolicy(ctx context.Context, name string) (*policy_engine.PolicyDocument, error) {
	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	var policy Policy
	err := store.db.NewSelect().Model(&policy).Where("name = ?", name).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("policy not found")
		}
		return nil, err
	}

	var doc policy_engine.PolicyDocument
	if err := json.Unmarshal(policy.Document, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy document: %w", err)
	}

	return &doc, nil
}

package bunsql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/credential"
	"github.com/seaweedfs/seaweedfs/weed/pb/iam_pb"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	Username    string    `bun:"username,pk,notnull"`
	Email       string    `bun:"email"`
	AccountData []byte    `bun:"account_data"` // JSONB
	Actions     []byte    `bun:"actions"`      // JSONB
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	
	Credentials []*Credential `bun:"rel:has-many,join:username=username"`
}

type Credential struct {
	bun.BaseModel `bun:"table:credentials,alias:c"`

	ID          int64      `bun:"id,pk,autoincrement"`
	Username    string     `bun:"username,notnull"`
	AccessKey   string     `bun:"access_key,unique,notnull"`
	SecretKey   string     `bun:"secret_key,notnull"`
	Permissions string     `bun:"permissions"`
	CreatedAt   time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	Expiration  *time.Time `bun:"expiration"`
}

func (store *BunSqlStore) LoadConfiguration(ctx context.Context) (*iam_pb.S3ApiConfiguration, error) {
	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	config := &iam_pb.S3ApiConfiguration{}
	
	var users []User
	err := store.db.NewSelect().Model(&users).Relation("Credentials").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}

	for _, user := range users {
		identity := &iam_pb.Identity{
			Name: user.Username,
		}
		
		if len(user.AccountData) > 0 {
			if err := json.Unmarshal(user.AccountData, &identity.Account); err != nil {
				return nil, fmt.Errorf("failed to unmarshal account data for user %s: %v", user.Username, err)
			}
		}

		if len(user.Actions) > 0 {
			if err := json.Unmarshal(user.Actions, &identity.Actions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal actions for user %s: %v", user.Username, err)
			}
		}

		for _, cred := range user.Credentials {
			t := int64(0)
			if cred.Expiration != nil {
				t = cred.Expiration.Unix()
			}
			identity.Credentials = append(identity.Credentials, &iam_pb.Credential{
				AccessKey: cred.AccessKey,
				SecretKey: cred.SecretKey,
				Expiration: t,
			})
		}
		
		config.Identities = append(config.Identities, identity)
	}

	return config, nil
}

func (store *BunSqlStore) SaveConfiguration(ctx context.Context, config *iam_pb.S3ApiConfiguration) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	return store.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		// Clear existing
		if _, err := tx.NewDelete().Model((*Credential)(nil)).Where("1=1").Exec(ctx); err != nil { return err }
		if _, err := tx.NewDelete().Model((*User)(nil)).Where("1=1").Exec(ctx); err != nil { return err }

		// Insert new
		for _, identity := range config.Identities {
			var accountData, actions []byte
			var err error
			if identity.Account != nil {
				accountData, err = json.Marshal(identity.Account)
				if err != nil { return err }
			}
			if identity.Actions != nil {
				actions, err = json.Marshal(identity.Actions)
				if err != nil { return err }
			}

			user := &User{
				Username:    identity.Name,
				AccountData: accountData,
				Actions:     actions,
			}
			if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
				return err
			}
			
			for _, cred := range identity.Credentials {
				var expiration *time.Time
				if cred.Expiration > 0 {
					t := time.Unix(cred.Expiration, 0)
					expiration = &t
				}
				c := &Credential{
					Username:   identity.Name,
					AccessKey:  cred.AccessKey,
					SecretKey:  cred.SecretKey,
					Expiration: expiration,
				}
				if _, err := tx.NewInsert().Model(c).Exec(ctx); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (store *BunSqlStore) CreateUser(ctx context.Context, identity *iam_pb.Identity) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	exists, err := store.db.NewSelect().Model((*User)(nil)).Where("username = ?", identity.Name).Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return credential.ErrUserAlreadyExists
	}

	return store.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		var accountData, actions []byte
		var err error
		if identity.Account != nil {
			accountData, err = json.Marshal(identity.Account)
			if err != nil { return err }
		}
		if identity.Actions != nil {
			actions, err = json.Marshal(identity.Actions)
			if err != nil { return err }
		}

		user := &User{
			Username:    identity.Name,
			AccountData: accountData,
			Actions:     actions,
		}
		if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
			return err
		}

		for _, cred := range identity.Credentials {
			var expiration *time.Time
			if cred.Expiration > 0 {
				t := time.Unix(cred.Expiration, 0)
				expiration = &t
			}
			c := &Credential{
				Username:   identity.Name,
				AccessKey:  cred.AccessKey,
				SecretKey:  cred.SecretKey,
				Expiration: expiration,
			}
			if _, err := tx.NewInsert().Model(c).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (store *BunSqlStore) GetUser(ctx context.Context, username string) (*iam_pb.Identity, error) {
	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	var user User
	err := store.db.NewSelect().Model(&user).Relation("Credentials").Where("username = ?", username).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, credential.ErrUserNotFound
		}
		return nil, err
	}

	identity := &iam_pb.Identity{
		Name: username,
	}

	if len(user.AccountData) > 0 {
		if err := json.Unmarshal(user.AccountData, &identity.Account); err != nil {
			return nil, fmt.Errorf("failed to unmarshal account data: %w", err)
		}
	}

	if len(user.Actions) > 0 {
		if err := json.Unmarshal(user.Actions, &identity.Actions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal actions: %w", err)
		}
	}

	for _, cred := range user.Credentials {
		t := int64(0)
		if cred.Expiration != nil {
			t = cred.Expiration.Unix()
		}
		identity.Credentials = append(identity.Credentials, &iam_pb.Credential{
			AccessKey: cred.AccessKey,
			SecretKey: cred.SecretKey,
			Expiration: t,
		})
	}

	return identity, nil
}

func (store *BunSqlStore) UpdateUser(ctx context.Context, username string, identity *iam_pb.Identity) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	return store.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		exists, err := tx.NewSelect().Model((*User)(nil)).Where("username = ?", username).Exists(ctx)
		if err != nil { return err }
		if !exists { return credential.ErrUserNotFound }

		var accountData, actions []byte
		if identity.Account != nil {
			accountData, err = json.Marshal(identity.Account)
			if err != nil { return err }
		}
		if identity.Actions != nil {
			actions, err = json.Marshal(identity.Actions)
			if err != nil { return err }
		}

		user := &User{
			Username:    username,
			AccountData: accountData,
			Actions:     actions,
			UpdatedAt:   time.Now(),
		}
		// Update user fields
		if _, err := tx.NewUpdate().Model(user).Column("account_data", "actions", "updated_at").Where("username = ?", username).Exec(ctx); err != nil {
			return err
		}

		// Replace credentials
		if _, err := tx.NewDelete().Model((*Credential)(nil)).Where("username = ?", username).Exec(ctx); err != nil {
			return err
		}

		for _, cred := range identity.Credentials {
			var expiration *time.Time
			if cred.Expiration > 0 {
				t := time.Unix(cred.Expiration, 0)
				expiration = &t
			}
			c := &Credential{
				Username:   username,
				AccessKey:  cred.AccessKey,
				SecretKey:  cred.SecretKey,
				Expiration: expiration,
			}
			if _, err := tx.NewInsert().Model(c).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (store *BunSqlStore) DeleteUser(ctx context.Context, username string) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	res, err := store.db.NewDelete().Model((*User)(nil)).Where("username = ?", username).Exec(ctx)
	if err != nil {
		return err
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return credential.ErrUserNotFound
	}
	// Credentials should be deleted by cascade if defined in DB schema, but we didn't define ON DELETE CASCADE in struct tags for Bun to handle automatically on DB side unless we use Exec DDL. 
	// Bun struct 'rel:has-many' doesn't enforce DB constraint automatically.
	// But let's assume we want to be safe and delete credentials manually or rely on DB constraint if createTables set it up.
	// In createTables, we just used Model(model).IfNotExists().Exec().
	// To be safe, let's delete credentials first manually if we are not sure about DB constraints.
	// Actually, wait, simpler: The User table has no foreign key to Credential. Credential has foreign key to User.
	// So we should delete credentials first or set up cascade.
	// Let's do manual delete credentials first in transaction if we were doing this properly, but pure DeleteUser usually implies cascading.
	// However, since we returned already if rows==0, let's just leave it there. If we want to be strict, we should wrap in transaction.
	
	// Re-implementing with explicit credential deletion for safety
	return store.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewDelete().Model((*User)(nil)).Where("username = ?", username).Exec(ctx)
		if err != nil { return err }
		rows, err := res.RowsAffected()
		if err != nil { return err }
		if rows == 0 { return credential.ErrUserNotFound }
		
		// Delete credentials
		_, err = tx.NewDelete().Model((*Credential)(nil)).Where("username = ?", username).Exec(ctx)
		return err
	})
}

func (store *BunSqlStore) ListUsers(ctx context.Context) ([]string, error) {
	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	var usernames []string
	err := store.db.NewSelect().Model((*User)(nil)).Column("username").Order("username ASC").Scan(ctx, &usernames)
	if err != nil {
		return nil, err
	}
	return usernames, nil
}

func (store *BunSqlStore) GetUserByAccessKey(ctx context.Context, accessKey string) (*iam_pb.Identity, error) {
	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	var cred Credential
	err := store.db.NewSelect().Model(&cred).Where("access_key = ?", accessKey).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, credential.ErrAccessKeyNotFound
		}
		return nil, err
	}

	return store.GetUser(ctx, cred.Username)
}

func (store *BunSqlStore) CreateAccessKey(ctx context.Context, username string, cred *iam_pb.Credential) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	exists, err := store.db.NewSelect().Model((*User)(nil)).Where("username = ?", username).Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return credential.ErrUserNotFound
	}
	
	var expiration *time.Time
	if cred.Expiration > 0 {
		t := time.Unix(cred.Expiration, 0)
		expiration = &t
	}

	c := &Credential{
		Username:   username,
		AccessKey:  cred.AccessKey,
		SecretKey:  cred.SecretKey,
		Expiration: expiration,
	}

	_, err = store.db.NewInsert().Model(c).Exec(ctx)
	return err
}

func (store *BunSqlStore) DeleteAccessKey(ctx context.Context, username string, accessKey string) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	res, err := store.db.NewDelete().Model((*Credential)(nil)).Where("username = ? AND access_key = ?", username, accessKey).Exec(ctx)
	if err != nil {
		return err
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	
	if rows == 0 {
		// Check user exists to disambiguate error
		exists, err := store.db.NewSelect().Model((*User)(nil)).Where("username = ?", username).Exists(ctx)
		if err != nil { return err }
		if !exists { return credential.ErrUserNotFound }
		return credential.ErrAccessKeyNotFound
	}

	return nil
}

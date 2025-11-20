package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/seaweedfs/seaweedfs/weed/credential"
	"github.com/seaweedfs/seaweedfs/weed/pb/iam_pb"
)

func (store *MysqlStore) LoadConfiguration(ctx context.Context) (*iam_pb.S3ApiConfiguration, error) {
	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	config := &iam_pb.S3ApiConfiguration{}

	// Query all users
	rows, err := store.db.QueryContext(ctx, "SELECT username, email, account_data, actions FROM users ORDER BY username")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var username, email string
		var accountDataJSON, actionsJSON []byte

		if err := rows.Scan(&username, &email, &accountDataJSON, &actionsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}

		identity := &iam_pb.Identity{
			Name: username,
		}

		// Parse account data
		if len(accountDataJSON) > 0 {
			if err := json.Unmarshal(accountDataJSON, &identity.Account); err != nil {
				return nil, fmt.Errorf("failed to unmarshal account data for user %s: %v", username, err)
			}
		}

		// Parse actions
		if len(actionsJSON) > 0 {
			if err := json.Unmarshal(actionsJSON, &identity.Actions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal actions for user %s: %v", username, err)
			}
		}

		// Query credentials for this user
		credRows, err := store.db.QueryContext(ctx, "SELECT access_key, secret_key, expiration FROM credentials WHERE username = ?", username)
		if err != nil {
			return nil, fmt.Errorf("failed to query credentials for user %s: %v", username, err)
		}

		for credRows.Next() {
			var accessKey, secretKey string
			var expirationTs sql.NullTime
			if err := credRows.Scan(&accessKey, &secretKey, &expirationTs); err != nil {
				credRows.Close()
				return nil, fmt.Errorf("failed to scan credential row for user %s: %v", username, err)
			}

			var expiration int64
			if expirationTs.Valid {
				expiration = expirationTs.Time.Unix()
			}
			identity.Credentials = append(identity.Credentials, &iam_pb.Credential{
				AccessKey:  accessKey,
				SecretKey:  secretKey,
				Expiration: expiration,
			})
		}
		credRows.Close()

		config.Identities = append(config.Identities, identity)
	}

	return config, nil
}

func (store *MysqlStore) SaveConfiguration(ctx context.Context, config *iam_pb.S3ApiConfiguration) error {
	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Start transaction
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing data
	if _, err := tx.ExecContext(ctx, "DELETE FROM credentials"); err != nil {
		return fmt.Errorf("failed to clear credentials: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM users"); err != nil {
		return fmt.Errorf("failed to clear users: %w", err)
	}

	// Insert all identities
	for _, identity := range config.Identities {
		// Marshal account data
		var accountDataJSON []byte
		if identity.Account != nil {
			accountDataJSON, err = json.Marshal(identity.Account)
			if err != nil {
				return fmt.Errorf("failed to marshal account data for user %s: %v", identity.Name, err)
			}
		}

		// Marshal actions
		var actionsJSON []byte
		if identity.Actions != nil {
			actionsJSON, err = json.Marshal(identity.Actions)
			if err != nil {
				return fmt.Errorf("failed to marshal actions for user %s: %v", identity.Name, err)
			}
		}

		// Insert user
		_, err := tx.ExecContext(ctx,
			"INSERT INTO users (username, email, account_data, actions) VALUES (?, ?, ?, ?)",
			identity.Name, "", accountDataJSON, actionsJSON)
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %v", identity.Name, err)
		}

		// Insert credentials
		for _, cred := range identity.Credentials {
			_, err := tx.ExecContext(ctx,
				"INSERT INTO credentials (username, access_key, secret_key, expiration) VALUES (?, ?, ?, ?)",
				identity.Name, cred.AccessKey, cred.SecretKey, *int64ToTime(cred.Expiration))
			if err != nil {
				return fmt.Errorf("failed to insert credential for user %s: %v", identity.Name, err)
			}
		}
	}

	return tx.Commit()
}

func (store *MysqlStore) CreateUser(ctx context.Context, identity *iam_pb.Identity) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Check if user already exists
	var count int
	err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = ?", identity.Name).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if count > 0 {
		return credential.ErrUserAlreadyExists
	}

	// Start transaction
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Marshal account data
	var accountDataJSON []byte
	if identity.Account != nil {
		accountDataJSON, err = json.Marshal(identity.Account)
		if err != nil {
			return fmt.Errorf("failed to marshal account data: %w", err)
		}
	}

	// Marshal actions
	var actionsJSON []byte
	if identity.Actions != nil {
		actionsJSON, err = json.Marshal(identity.Actions)
		if err != nil {
			return fmt.Errorf("failed to marshal actions: %w", err)
		}
	}

	// Insert user
	_, err = tx.ExecContext(ctx,
		"INSERT INTO users (username, email, account_data, actions) VALUES (?, ?, ?, ?)",
		identity.Name, "", accountDataJSON, actionsJSON)
	if err != nil {
		if isMysqlDuplicateEntryError(err) {
			return credential.ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}

	// Insert credentials
	for _, cred := range identity.Credentials {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO credentials (username, access_key, secret_key, expiration) VALUES (?, ?, ?, ?)",
			identity.Name, cred.AccessKey, cred.SecretKey, *int64ToTime(cred.Expiration))
		if err != nil {
			return fmt.Errorf("failed to insert credential: %w", err)
		}
	}

	return tx.Commit()
}

func (store *MysqlStore) GetUser(ctx context.Context, username string) (*iam_pb.Identity, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	var email string
	var accountDataJSON, actionsJSON []byte

	err := store.db.QueryRowContext(ctx,
		"SELECT email, account_data, actions FROM users WHERE username = ?",
		username).Scan(&email, &accountDataJSON, &actionsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, credential.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	identity := &iam_pb.Identity{
		Name: username,
	}

	// Parse account data
	if len(accountDataJSON) > 0 {
		if err := json.Unmarshal(accountDataJSON, &identity.Account); err != nil {
			return nil, fmt.Errorf("failed to unmarshal account data: %w", err)
		}
	}

	// Parse actions
	if len(actionsJSON) > 0 {
		if err := json.Unmarshal(actionsJSON, &identity.Actions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal actions: %w", err)
		}
	}

	// Query credentials
	rows, err := store.db.QueryContext(ctx, "SELECT access_key, secret_key, expiration FROM credentials WHERE username = ?", username)
	if err != nil {
		return nil, fmt.Errorf("failed to query credentials: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var accessKey, secretKey string
		var expirationTs sql.NullTime
		if err := rows.Scan(&accessKey, &secretKey, &expirationTs); err != nil {
			return nil, fmt.Errorf("failed to scan credential: %w", err)
		}

		var expiration int64
		if expirationTs.Valid {
			expiration = expirationTs.Time.Unix()
		}
		identity.Credentials = append(identity.Credentials, &iam_pb.Credential{
			AccessKey:  accessKey,
			SecretKey:  secretKey,
			Expiration: expiration,
		})
	}

	return identity, nil
}

func (store *MysqlStore) UpdateUser(ctx context.Context, username string, identity *iam_pb.Identity) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Start transaction
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if user exists
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if count == 0 {
		return credential.ErrUserNotFound
	}

	// Marshal account data
	var accountDataJSON []byte
	if identity.Account != nil {
		accountDataJSON, err = json.Marshal(identity.Account)
		if err != nil {
			return fmt.Errorf("failed to marshal account data: %w", err)
		}
	}

	// Marshal actions
	var actionsJSON []byte
	if identity.Actions != nil {
		actionsJSON, err = json.Marshal(identity.Actions)
		if err != nil {
			return fmt.Errorf("failed to marshal actions: %w", err)
		}
	}

	// Update user
	_, err = tx.ExecContext(ctx,
		"UPDATE users SET email = ?, account_data = ?, actions = ?, updated_at = CURRENT_TIMESTAMP WHERE username = ?",
		"", accountDataJSON, actionsJSON, username)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Delete existing credentials
	_, err = tx.ExecContext(ctx, "DELETE FROM credentials WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete existing credentials: %w", err)
	}

	// Insert new credentials
	for _, cred := range identity.Credentials {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO credentials (username, access_key, secret_key, expiration) VALUES (?, ?, ?, ?)",
			username, cred.AccessKey, cred.SecretKey, *int64ToTime(cred.Expiration))
		if err != nil {
			return fmt.Errorf("failed to insert credential: %w", err)
		}
	}

	return tx.Commit()
}

func (store *MysqlStore) DeleteUser(ctx context.Context, username string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return credential.ErrUserNotFound
		}
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM credentials WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete user credentials: %w", err)
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return tx.Commit()
}

func (store *MysqlStore) ListUsers(ctx context.Context) ([]string, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	rows, err := store.db.QueryContext(ctx, "SELECT username FROM users ORDER BY username")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, fmt.Errorf("failed to scan username: %w", err)
		}
		usernames = append(usernames, username)
	}

	return usernames, nil
}

func (store *MysqlStore) GetUserByAccessKey(ctx context.Context, accessKey string) (*iam_pb.Identity, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	var username string
	err := store.db.QueryRowContext(ctx, "SELECT username FROM credentials WHERE access_key = ?", accessKey).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, credential.ErrAccessKeyNotFound
		}
		return nil, fmt.Errorf("failed to query access key: %w", err)
	}

	return store.GetUser(ctx, username)
}

func (store *MysqlStore) CreateAccessKey(ctx context.Context, username string, cred *iam_pb.Credential) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Check if user exists
	var count int
	err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if count == 0 {
		return credential.ErrUserNotFound
	}

	// Insert credential
	_, err = store.db.ExecContext(ctx,
		"INSERT INTO credentials (username, access_key, secret_key, expiration) VALUES (?, ?, ?, ?)",
		username, cred.AccessKey, cred.SecretKey, *int64ToTime(cred.Expiration))
	if err != nil {
		return fmt.Errorf("failed to insert credential: %w", err)
	}

	return nil
}

func (store *MysqlStore) DeleteAccessKey(ctx context.Context, username string, accessKey string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	result, err := store.db.ExecContext(ctx,
		"DELETE FROM credentials WHERE username = ? AND access_key = ?",
		username, accessKey)
	if err != nil {
		return fmt.Errorf("failed to delete access key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Check if user exists
		var count int
		err = store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check user existence: %w", err)
		}
		if count == 0 {
			return credential.ErrUserNotFound
		}
		return credential.ErrAccessKeyNotFound
	}

	return nil
}

func int64ToTime(timestamp int64) *sql.NullTime {
	if timestamp <= 0 {
		return &sql.NullTime{
			Time:  time.Time{},
			Valid: false,
		}
	}
	t := time.Unix(timestamp, 0)
	return &sql.NullTime{Time: t, Valid: true}
}

func isMysqlDuplicateEntryError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}

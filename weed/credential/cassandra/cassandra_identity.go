package cassandra

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/seaweedfs/seaweedfs/weed/credential"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/iam_pb"
)

// batchWithContext wraps a batch operation with context
func batchWithContext(ctx context.Context, batch *gocql.Batch) *gocql.Batch {
	// Note: gocql batch doesn't directly support context,
	// but we can set the context on individual queries within the batch
	return batch
}

func (store *CassandraStore) LoadConfiguration(ctx context.Context) (*iam_pb.S3ApiConfiguration, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	query := fmt.Sprintf(`
		SELECT config_data FROM %s.%s_config
		WHERE config_key = ? LIMIT 1`, store.keyspace, store.tableName)

	var configData string
	err := store.session.Query(query, "s3_api_config").
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&configData)

	if err != nil {
		if err == gocql.ErrNotFound {
			// Return empty config if not found
			return &iam_pb.S3ApiConfiguration{}, nil
		}
		return nil, fmt.Errorf("failed to load configuration: %v", err)
	}

	var config iam_pb.S3ApiConfiguration
	err = json.Unmarshal([]byte(configData), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %v", err)
	}

	return &config, nil
}

func (store *CassandraStore) SaveConfiguration(ctx context.Context, config *iam_pb.S3ApiConfiguration) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Serialize configuration
	configData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %v", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.%s_config (config_key, config_data, updated_at)
		VALUES (?, ?, ?)`, store.keyspace, store.tableName)

	err = store.session.Query(query, "s3_api_config", string(configData), time.Now()).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	glog.V(2).Infof("Saved S3 API configuration to Cassandra")
	return nil
}

func (store *CassandraStore) CreateUser(ctx context.Context, identity *iam_pb.Identity) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Start batch operation
	batch := store.session.NewBatch(gocql.LoggedBatch)

	// Marshal account data
	var accountDataJSON []byte
	var err error
	if identity.Account != nil {
		accountDataJSON, err = json.Marshal(identity.Account)
		if err != nil {
			return fmt.Errorf("failed to marshal account data: %v", err)
		}
	} else {
		accountDataJSON = []byte{}
	}

	// Marshal actions
	var actionsJSON []byte
	if identity.Actions != nil {
		actionsJSON, err = json.Marshal(identity.Actions)
		if err != nil {
			return fmt.Errorf("failed to marshal actions: %v", err)
		}
	} else {
		actionsJSON = []byte{}
	}

	// Insert user
	userQuery := fmt.Sprintf(`
		INSERT INTO %s.%s_users (username, email, account_data, actions, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`, store.keyspace, store.tableName)
	batch.Query(userQuery, identity.Name, string(accountDataJSON), string(actionsJSON), time.Now(), time.Now())

	// Insert credentials
	for _, cred := range identity.Credentials {
		credQuery := fmt.Sprintf(`
			INSERT INTO %s.%s_credentials (id, username, access_key, secret_key, created_at, updated_at, expiration)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, store.keyspace, store.tableName)
		batch.Query(credQuery, gocql.TimeUUID(), identity.Name, cred.AccessKey, cred.SecretKey, time.Now(), time.Now(), time.Unix(cred.Expiration, 0))
	}

	// Execute batch
	err = store.session.ExecuteBatch(batchWithContext(ctx, batch))
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	glog.V(2).Infof("Created user %s in Cassandra", identity.Name)
	return nil
}

func (store *CassandraStore) GetUser(ctx context.Context, username string) (*iam_pb.Identity, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	return store.getUser(ctx, username)
}

func (store *CassandraStore) getUser(ctx context.Context, username string) (*iam_pb.Identity, error) {
	query := fmt.Sprintf(`
		SELECT email, account_data, actions FROM %s.%s_users
		WHERE username = ? LIMIT 1`, store.keyspace, store.tableName)

	var email string
	var accountDataJSON, actionsJSON []byte
	err := store.session.Query(query, username).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&email, &accountDataJSON, &actionsJSON)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, credential.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	identity := &iam_pb.Identity{
		Name: username,
	}

	// Unmarshal account data
	if len(accountDataJSON) > 0 {
		var account iam_pb.Account
		if err := json.Unmarshal(accountDataJSON, &account); err != nil {
			return nil, fmt.Errorf("failed to unmarshal account data: %v", err)
		}
		identity.Account = &account
	}

	// Unmarshal actions
	if len(actionsJSON) > 0 {
		var actions map[string]interface{}
		if err := json.Unmarshal(actionsJSON, &actions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal actions: %v", err)
		}
		// Convert to protobuf format if needed
	}

	// Get credentials
	credQuery := fmt.Sprintf(`
		SELECT access_key, secret_key, expiration FROM %s.%s_credentials
		WHERE username = ?`, store.keyspace, store.tableName)

	iter := store.session.Query(credQuery, username).WithContext(ctx).Iter()
	defer iter.Close()

	var accessKey, secretKey string
	var expiration time.Time
	var credentials []*iam_pb.Credential

	for iter.Scan(&accessKey, &secretKey, &expiration) {
		cred := &iam_pb.Credential{
			AccessKey:  accessKey,
			SecretKey:  secretKey,
			Expiration: expiration.Unix(),
		}
		credentials = append(credentials, cred)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to iterate credentials: %v", err)
	}

	identity.Credentials = credentials
	return identity, nil
}

func (store *CassandraStore) UpdateUser(ctx context.Context, username string, identity *iam_pb.Identity) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Check if user exists
	_, err := store.getUser(ctx, username)
	if err != nil {
		return err
	}

	// Start batch operation
	batch := store.session.NewBatch(gocql.LoggedBatch)

	// Marshal account data
	var accountDataJSON []byte
	if identity.Account != nil {
		accountDataJSON, err = json.Marshal(identity.Account)
		if err != nil {
			return fmt.Errorf("failed to marshal account data: %v", err)
		}
	} else {
		accountDataJSON = []byte{}
	}

	// Marshal actions
	var actionsJSON []byte
	if identity.Actions != nil {
		actionsJSON, err = json.Marshal(identity.Actions)
		if err != nil {
			return fmt.Errorf("failed to marshal actions: %v", err)
		}
	} else {
		actionsJSON = []byte{}
	}

	// Update user
	userQuery := fmt.Sprintf(`
		UPDATE %s.%s_users SET email = ?, account_data = ?, actions = ?, updated_at = ?
		WHERE username = ?`, store.keyspace, store.tableName)
	batch.Query(userQuery, string(accountDataJSON), string(actionsJSON), time.Now(), username)

	// Delete existing credentials
	deleteCredQuery := fmt.Sprintf(`
		DELETE FROM %s.%s_credentials WHERE username = ?`, store.keyspace, store.tableName)
	batch.Query(deleteCredQuery, username)

	// Insert new credentials
	for _, cred := range identity.Credentials {
		credQuery := fmt.Sprintf(`
			INSERT INTO %s.%s_credentials (id, username, access_key, secret_key, created_at, updated_at, expiration)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, store.keyspace, store.tableName)
		batch.Query(credQuery, gocql.TimeUUID(), username, cred.AccessKey, cred.SecretKey, time.Now(), time.Now(), time.Unix(cred.Expiration, 0))
	}

	// Execute batch
	err = store.session.ExecuteBatch(batchWithContext(ctx, batch))
	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}

	glog.V(2).Infof("Updated user %s in Cassandra", username)
	return nil
}

func (store *CassandraStore) DeleteUser(ctx context.Context, username string) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	return store.deleteUser(ctx, username)
}

func (store *CassandraStore) deleteUser(ctx context.Context, username string) error {
	// Start batch operation
	batch := store.session.NewBatch(gocql.LoggedBatch)

	// Delete user
	userQuery := fmt.Sprintf(`
		DELETE FROM %s.%s_users WHERE username = ?`, store.keyspace, store.tableName)
	batch.Query(userQuery, username)

	// Delete credentials
	credQuery := fmt.Sprintf(`
		DELETE FROM %s.%s_credentials WHERE username = ?`, store.keyspace, store.tableName)
	batch.Query(credQuery, username)

	// Execute batch
	err := store.session.ExecuteBatch(batchWithContext(ctx, batch))
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	glog.V(2).Infof("Deleted user %s from Cassandra", username)
	return nil
}

func (store *CassandraStore) ListUsers(ctx context.Context) ([]string, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	query := fmt.Sprintf(`
		SELECT username FROM %s.%s_users`, store.keyspace, store.tableName)

	iter := store.session.Query(query).WithContext(ctx).Iter()
	defer iter.Close()

	var username string
	var usernames []string

	for iter.Scan(&username) {
		usernames = append(usernames, username)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %v", err)
	}

	return usernames, nil
}

func (store *CassandraStore) GetUserByAccessKey(ctx context.Context, accessKey string) (*iam_pb.Identity, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	query := fmt.Sprintf(`
		SELECT username FROM %s.%s_credentials
		WHERE access_key = ? LIMIT 1`, store.keyspace, store.tableName)

	var username string
	err := store.session.Query(query, accessKey).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&username)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, credential.ErrAccessKeyNotFound
		}
		return nil, fmt.Errorf("failed to query access key: %v", err)
	}

	return store.getUser(ctx, username)
}

func (store *CassandraStore) CreateAccessKey(ctx context.Context, username string, credential *iam_pb.Credential) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	return store.createAccessKey(ctx, username, credential)
}

func (store *CassandraStore) createAccessKey(ctx context.Context, username string, credential *iam_pb.Credential) error {
	// Check if user exists
	_, err := store.getUser(ctx, username)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.%s_credentials (id, username, access_key, secret_key, created_at, updated_at, expiration)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, store.keyspace, store.tableName)

	err = store.session.Query(query, gocql.TimeUUID(), username, credential.AccessKey, credential.SecretKey, time.Now(), time.Now(), time.Unix(credential.Expiration, 0)).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to create access key: %v", err)
	}

	glog.V(2).Infof("Created access key for user %s in Cassandra", username)
	return nil
}

func (store *CassandraStore) DeleteAccessKey(ctx context.Context, username string, accessKey string) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	return store.deleteAccessKey(ctx, username, accessKey)
}

func (store *CassandraStore) deleteAccessKey(ctx context.Context, username string, accessKey string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.%s_credentials WHERE username = ? AND access_key = ?`, store.keyspace, store.tableName)

	err := store.session.Query(query, username, accessKey).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to delete access key: %v", err)
	}

	glog.V(2).Infof("Deleted access key for user %s in Cassandra", username)
	return nil
}

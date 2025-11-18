package cassandra

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/iam_pb"
)

// CreateUser creates a new user with the given identity
func (store *CassandraStore) CreateUser(ctx context.Context, identity *iam_pb.Identity) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Check if user already exists
	existingUser, err := store.getUser(ctx, identity.Name)
	if err == nil && existingUser != nil {
		return fmt.Errorf("user %s already exists", identity.Name)
	}

	// Serialize identity to JSON
	identityData, err := json.Marshal(identity)
	if err != nil {
		return fmt.Errorf("failed to marshal identity: %v", err)
	}

	// Insert user
	query := fmt.Sprintf(`
		INSERT INTO %s.%s_users (username, identity_data, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, store.keyspace, store.tableName)

	err = store.session.Query(query, identity.Name, string(identityData), time.Now(), time.Now()).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	// Create access keys
	for _, cred := range identity.Credentials {
		err = store.createAccessKey(ctx, identity.Name, cred)
		if err != nil {
			// Clean up user on failure
			store.deleteUser(ctx, identity.Name)
			return fmt.Errorf("failed to create access key: %v", err)
		}
	}

	glog.V(2).Infof("Created user %s in Cassandra", identity.Name)
	return nil
}

// GetUser retrieves a user by username
func (store *CassandraStore) GetUser(ctx context.Context, username string) (*iam_pb.Identity, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	return store.getUser(ctx, username)
}

// getUser retrieves a user by username (internal method, without locking)
func (store *CassandraStore) getUser(ctx context.Context, username string) (*iam_pb.Identity, error) {
	query := fmt.Sprintf(`
		SELECT identity_data FROM %s.%s_users
		WHERE username = ? LIMIT 1`, store.keyspace, store.tableName)

	var identityData string
	err := store.session.Query(query, username).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&identityData)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, fmt.Errorf("user %s not found", username)
		}
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	// Deserialize identity
	var identity iam_pb.Identity
	err = json.Unmarshal([]byte(identityData), &identity)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal identity: %v", err)
	}

	return &identity, nil
}

// UpdateUser updates an existing user
func (store *CassandraStore) UpdateUser(ctx context.Context, username string, identity *iam_pb.Identity) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Check if user exists
	existingUser, err := store.getUser(ctx, username)
	if err != nil {
		return fmt.Errorf("user %s not found", username)
	}

	// Ensure username matches
	if identity.Name != username {
		return fmt.Errorf("username mismatch: expected %s, got %s", username, identity.Name)
	}

	// Delete existing access keys
	for _, cred := range existingUser.Credentials {
		err = store.deleteAccessKey(ctx, username, cred.AccessKey)
		if err != nil {
			glog.V(1).Infof("Warning: failed to delete access key %s: %v", cred.AccessKey, err)
		}
	}

	// Serialize updated identity
	identityData, err := json.Marshal(identity)
	if err != nil {
		return fmt.Errorf("failed to marshal identity: %v", err)
	}

	// Update user
	query := fmt.Sprintf(`
		UPDATE %s.%s_users
		SET identity_data = ?, updated_at = ?
		WHERE username = ?`, store.keyspace, store.tableName)

	err = store.session.Query(query, string(identityData), time.Now(), username).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}

	// Store new access keys
	for _, cred := range identity.Credentials {
		err = store.createAccessKey(ctx, username, cred)
		if err != nil {
			return fmt.Errorf("failed to create access key during update: %v", err)
		}
	}

	glog.V(2).Infof("Updated user %s in Cassandra", username)
	return nil
}

// DeleteUser removes a user by username
func (store *CassandraStore) DeleteUser(ctx context.Context, username string) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	// Get user to delete access keys first
	identity, err := store.getUser(ctx, username)
	if err != nil {
		return fmt.Errorf("user %s not found", username)
	}

	// Delete all access keys
	for _, cred := range identity.Credentials {
		err = store.deleteAccessKey(ctx, username, cred.AccessKey)
		if err != nil {
			glog.V(1).Infof("Warning: failed to delete access key %s: %v", cred.AccessKey, err)
		}
	}

	// Delete user
	err = store.deleteUser(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	glog.V(2).Infof("Deleted user %s from Cassandra", username)
	return nil
}

// deleteUser removes a user by username (internal method, without locking)
func (store *CassandraStore) deleteUser(ctx context.Context, username string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.%s_users
		WHERE username = ?`, store.keyspace, store.tableName)

	err := store.session.Query(query, username).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	return nil
}

// ListUsers returns all usernames
func (store *CassandraStore) ListUsers(ctx context.Context) ([]string, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	query := fmt.Sprintf(`
		SELECT username FROM %s.%s_users`, store.keyspace, store.tableName)

	iter := store.session.Query(query).
		WithContext(ctx).
		Iter()

	var usernames []string
	var username string

	for iter.Scan(&username) {
		usernames = append(usernames, username)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to list users: %v", err)
	}

	return usernames, nil
}

// GetUserByAccessKey retrieves a user by access key
func (store *CassandraStore) GetUserByAccessKey(ctx context.Context, accessKey string) (*iam_pb.Identity, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return nil, fmt.Errorf("store not configured")
	}

	query := fmt.Sprintf(`
		SELECT username FROM %s.%s_access_keys
		WHERE access_key = ? LIMIT 1`, store.keyspace, store.tableName)

	var username string
	err := store.session.Query(query, accessKey).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&username)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, fmt.Errorf("access key %s not found", accessKey)
		}
		return nil, fmt.Errorf("failed to get user by access key: %v", err)
	}

	return store.getUser(ctx, username)
}

// CreateAccessKey creates a new access key for a user
func (store *CassandraStore) CreateAccessKey(ctx context.Context, username string, credential *iam_pb.Credential) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	return store.createAccessKey(ctx, username, credential)
}

// createAccessKey creates a new access key for a user (internal method, without locking)
func (store *CassandraStore) createAccessKey(ctx context.Context, username string, credential *iam_pb.Credential) error {
	// Serialize credential
	credentialData, err := json.Marshal(credential)
	if err != nil {
		return fmt.Errorf("failed to marshal credential: %v", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.%s_access_keys (access_key, username, credential_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, store.keyspace, store.tableName)

	err = store.session.Query(query, credential.AccessKey, username, string(credentialData), time.Now(), time.Now()).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to create access key: %v", err)
	}

	return nil
}

// DeleteAccessKey removes an access key for a user
func (store *CassandraStore) DeleteAccessKey(ctx context.Context, username string, accessKey string) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if !store.configured {
		return fmt.Errorf("store not configured")
	}

	return store.deleteAccessKey(ctx, username, accessKey)
}

// deleteAccessKey removes an access key for a user (internal method, without locking)
func (store *CassandraStore) deleteAccessKey(ctx context.Context, username string, accessKey string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.%s_access_keys
		WHERE access_key = ?`, store.keyspace, store.tableName)

	err := store.session.Query(query, accessKey).
		WithContext(ctx).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to delete access key: %v", err)
	}

	return nil
}

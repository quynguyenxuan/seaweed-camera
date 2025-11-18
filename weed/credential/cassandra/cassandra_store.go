package cassandra

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/seaweedfs/seaweedfs/weed/credential"
	"github.com/seaweedfs/seaweedfs/weed/filer/cassandra2"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/iam_pb"
	"github.com/seaweedfs/seaweedfs/weed/util"
)

func init() {
	credential.Stores = append(credential.Stores, NewCassandraStore())
}

// CassandraStore implements CredentialStore interface using Cassandra
type CassandraStore struct {
	mu         sync.RWMutex
	session    *gocql.Session
	keyspace   string
	tableName  string
	configured bool
}

// NewCassandraStore creates a new Cassandra-backed credential store
func NewCassandraStore() *CassandraStore {
	return &CassandraStore{}
}

func (store *CassandraStore) GetName() credential.CredentialStoreTypeName {
	return credential.StoreTypeCassandra
}

func (store *CassandraStore) Initialize(configuration util.Configuration, prefix string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.configured {
		return nil
	}

	// Get keyspace
	keyspace := "sunfs"
	if ks := configuration.GetString(prefix + "keyspace"); ks != "" {
		keyspace = ks
	}

	// Get table name prefix
	tableName := "credentials"
	if tn := configuration.GetString(prefix + "table_prefix"); tn != "" {
		tableName = tn
	}

	// Get Cassandra cluster configuration
	filerStore := cassandra2.GetInstance()
	cluster := filerStore.GetCluster()
	cluster.Keyspace = keyspace

	// Create session
	session, err := cluster.CreateSession()
	if err != nil {
		return fmt.Errorf("failed to create Cassandra session: %v", err)
	}

	// Create tables
	err = store.createTables(session, keyspace, tableName)
	if err != nil {
		session.Close()
		return fmt.Errorf("failed to create tables: %v", err)
	}

	store.session = session
	store.keyspace = keyspace
	store.tableName = tableName
	store.configured = true

	glog.V(0).Infof("Cassandra credential store initialized with keyspace: %s, table prefix: %s", keyspace, tableName)
	return nil
}

func (store *CassandraStore) createTables(session *gocql.Session, keyspace, tableName string) error {
	// Create users table
	err := session.Query(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s_users (
			username text PRIMARY KEY,
			identity_data text,
			created_at timestamp,
			updated_at timestamp
		)`, keyspace, tableName)).WithContext(context.Background()).Exec()

	if err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	// Create access keys table
	err = session.Query(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s_access_keys (
			access_key text PRIMARY KEY,
			username text,
			credential_data text,
			created_at timestamp,
			updated_at timestamp
		)`, keyspace, tableName)).WithContext(context.Background()).Exec()

	if err != nil {
		return fmt.Errorf("failed to create access keys table: %v", err)
	}

	// Create policies table
	err = session.Query(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s_policies (
			policy_name text PRIMARY KEY,
			policy_data text,
			created_at timestamp,
			updated_at timestamp
		)`, keyspace, tableName)).WithContext(context.Background()).Exec()

	if err != nil {
		return fmt.Errorf("failed to create policies table: %v", err)
	}

	// Create config table
	err = session.Query(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s_config (
			config_key text PRIMARY KEY,
			config_data text,
			updated_at timestamp
		)`, keyspace, tableName)).WithContext(context.Background()).Exec()

	if err != nil {
		return fmt.Errorf("failed to create config table: %v", err)
	}

	return nil
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

func (store *CassandraStore) Shutdown() {
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.session != nil {
		store.session.Close()
		store.session = nil
	}
	store.configured = false
	glog.V(2).Infof("Cassandra credential store shut down")
}

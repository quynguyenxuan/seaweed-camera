package cassandra

import (
	"context"
	"fmt"
	"sync"

	"github.com/gocql/gocql"
	"github.com/seaweedfs/seaweedfs/weed/credential"
	"github.com/seaweedfs/seaweedfs/weed/filer/cassandra2"
	"github.com/seaweedfs/seaweedfs/weed/glog"
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
	// cluster := filerStore.GetCluster()
	// cluster.Keyspace = keyspace

	// Create session
	// session, err := cluster.CreateSession()
	session := filerStore.GetSession()

	if session == nil {
		return fmt.Errorf("failed to create Cassandra session: %v")
	}

	// Create tables
	err := store.createTables(session, keyspace, tableName)
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
			email text,
			account_data text,
			actions text,
			created_at timestamp,
			updated_at timestamp
		)`, keyspace, tableName)).WithContext(context.Background()).Exec()

	if err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	// Create access keys table
	err = session.Query(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s_credentials (
			id timeuuid PRIMARY KEY,
			username text,
			access_key text,
			secret_key text,
			created_at timestamp,
			updated_at timestamp,
			expiration timestamp
		)`, keyspace, tableName)).WithContext(context.Background()).Exec()

	if err != nil {
		return fmt.Errorf("failed to create access keys table: %v", err)
	}

	// Create policies table
	err = session.Query(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s_policies (
			name text PRIMARY KEY,
			document text,
			created_at timestamp,
			updated_at timestamp
		)`, keyspace, tableName)).WithContext(context.Background()).Exec()

	if err != nil {
		return fmt.Errorf("failed to create policies table: %v", err)
	}

	// Create secondary indexes for efficient querying
	// Index for users email
	err = session.Query(fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_users_email ON %s.%s_users (email)
	`, tableName, keyspace, tableName)).WithContext(context.Background()).Exec()
	if err != nil {
		return fmt.Errorf("failed to create users email index: %v", err)
	}

	// Index for credentials username
	err = session.Query(fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_credentials_username ON %s.%s_credentials (username)
	`, tableName, keyspace, tableName)).WithContext(context.Background()).Exec()
	if err != nil {
		return fmt.Errorf("failed to create credentials username index: %v", err)
	}

	// Index for credentials access_key
	err = session.Query(fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_credentials_access_key ON %s.%s_credentials (access_key)
	`, tableName, keyspace, tableName)).WithContext(context.Background()).Exec()
	if err != nil {
		return fmt.Errorf("failed to create credentials access_key index: %v", err)
	}

	// Index for policies name
	err = session.Query(fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_policies_name ON %s.%s_policies (name)
	`, tableName, keyspace, tableName)).WithContext(context.Background()).Exec()
	if err != nil {
		// return fmt.Errorf("failed to create policies name index: %v", err)
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

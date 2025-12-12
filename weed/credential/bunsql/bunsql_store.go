package bunsql

import (
	"context"
	"fmt"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/credential"
	"github.com/seaweedfs/seaweedfs/weed/util"
	"github.com/uptrace/bun"
)

func init() {
	credential.Stores = append(credential.Stores, &BunSqlStore{})
}

type BunSqlStore struct {
	db         *bun.DB
	configured bool
}

func (store *BunSqlStore) GetName() credential.CredentialStoreTypeName {
	return credential.StoreTypeBunSql
}

func (store *BunSqlStore) Initialize(configuration util.Configuration, prefix string) error {
	if store.configured {
		return nil
	}

	hostname := configuration.GetString(prefix + "hostname")
	port := configuration.GetInt(prefix + "port")
	username := configuration.GetString(prefix + "username")
	password := configuration.GetString(prefix + "password")
	database := configuration.GetString(prefix + "database")
	driver := configuration.GetString(prefix + "driver")
	schema := configuration.GetString(prefix + "schema")
	sslmode := configuration.GetString(prefix + "sslmode")
	connectionMaxIdle := configuration.GetInt(prefix + "connection_max_idle")
	connectionMaxOpen := configuration.GetInt(prefix + "connection_max_open")
	connectionMaxLifetime := configuration.GetInt(prefix + "connection_max_lifetime_seconds")
	interpolateParams := configuration.GetBool(prefix + "interpolateParams")
	enableUpsert := configuration.GetBool(prefix + "enableUpsert")

	//QUYNGUYEN Use shared database connection function
	config := NewDatabaseConfig()
	config.Driver = driver
	config.Hostname = hostname
	config.Port = port
	config.Username = username
	config.Password = password
	config.Database = database
	config.Schema = schema
	config.SSLMode = sslmode
	config.ConnectionMaxIdle = connectionMaxIdle
	config.ConnectionMaxOpen = connectionMaxOpen
	config.ConnectionMaxLifetime = time.Duration(connectionMaxLifetime) * time.Second
	config.InterpolateParams = interpolateParams
	config.EnableUpsert = enableUpsert

	db, err := ConnectToDatabase(config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	//QUYNGUYEN end

	store.db = db

	if err := store.createTables(); err != nil {
		db.Close()
		return fmt.Errorf("failed to create tables: %w", err)
	}

	store.configured = true
	return nil
}

func (store *BunSqlStore) createTables() error {
	// Register models for table creation
	models := []interface{}{
		(*User)(nil),
		(*Credential)(nil),
		(*Policy)(nil),
	}

	for _, model := range models {
		_, err := store.db.NewCreateTable().Model(model).IfNotExists().Exec(context.Background())
		if err != nil {
			return err
		}
	}
	return nil
}

func (store *BunSqlStore) Shutdown() {
	if store.db != nil {
		store.db.Close()
		store.db = nil
	}
	store.configured = false
}

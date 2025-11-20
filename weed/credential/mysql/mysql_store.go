package mysql

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/credential"
	"github.com/seaweedfs/seaweedfs/weed/util"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	credential.Stores = append(credential.Stores, &MysqlStore{})
}

// MysqlStore implements CredentialStore using MySQL
type MysqlStore struct {
	mu        sync.RWMutex
	db         *sql.DB
	configured bool
}

func (store *MysqlStore) GetName() credential.CredentialStoreTypeName {
	return credential.StoreTypeMysql
}

func (store *MysqlStore) Initialize(configuration util.Configuration, prefix string) error {
	if store.configured {
		return nil
	}

	hostname := configuration.GetString(prefix + "hostname")
	port := configuration.GetInt(prefix + "port")
	username := configuration.GetString(prefix + "username")
	password := configuration.GetString(prefix + "password")
	database := configuration.GetString(prefix + "database")
	sslmode := configuration.GetString(prefix + "sslmode")

	// Set defaults
	if hostname == "" {
		hostname = "localhost"
	}
	if port == 0 {
		port = 3306
	}
	if sslmode == "" {
		sslmode = "false"
	}

	// Build MySQL connection string
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=5s&parseTime=true&tls=%s",
		username, password, hostname, port, database, sslmode)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	store.db = db

	// Create tables if they don't exist
	if err := store.createTables(); err != nil {
		db.Close()
		return fmt.Errorf("failed to create tables: %w", err)
	}

	store.configured = true
	return nil
}

func (store *MysqlStore) createTables() error {
	// Create users table
	usersTable := `
		CREATE TABLE IF NOT EXISTS users (
			username VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255),
			account_data JSON,
			actions JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			KEY idx_users_email (email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	// Create credentials table
	credentialsTable := `
		CREATE TABLE IF NOT EXISTS credentials (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			access_key VARCHAR(255) UNIQUE NOT NULL,
			secret_key VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			expiration TIMESTAMP NULL,
			KEY idx_credentials_username (username),
			CONSTRAINT fk_credentials_user FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	// Create policies table
	policiesTable := `
		CREATE TABLE IF NOT EXISTS policies (
			name VARCHAR(255) PRIMARY KEY,
			document JSON NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	// Execute table creation
	if _, err := store.db.Exec(usersTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if _, err := store.db.Exec(credentialsTable); err != nil {
		return fmt.Errorf("failed to create credentials table: %w", err)
	}

	if _, err := store.db.Exec(policiesTable); err != nil {
		return fmt.Errorf("failed to create policies table: %w", err)
	}

	return nil
}

func (store *MysqlStore) Shutdown() {
	if store.db != nil {
		store.db.Close()
		store.db = nil
	}
	store.configured = false
}

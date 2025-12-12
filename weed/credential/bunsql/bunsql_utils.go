package bunsql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	_ "github.com/uptrace/bun/driver/sqliteshim" // For consistency
)

// DatabaseConfig holds the configuration for database connection
type DatabaseConfig struct {
	Driver                string
	Hostname              string
	Port                  int
	Username              string
	Password              string
	Database              string
	Schema                string
	SSLMode               string
	Timeout               time.Duration
	ConnectionMaxIdle     int
	ConnectionMaxOpen     int
	ConnectionMaxLifetime time.Duration
	InterpolateParams     bool
	EnableUpsert          bool
}

// NewDatabaseConfig creates a new DatabaseConfig with default values
func NewDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Driver:                "postgresql",
		Hostname:              "localhost",
		Port:                  5432,
		Username:              "sunfs",
		Password:              "",
		Database:              "",
		Schema:                "public",
		SSLMode:               "disable",
		Timeout:               30 * time.Second,
		ConnectionMaxIdle:     2,
		ConnectionMaxOpen:     25,
		ConnectionMaxLifetime: 0,
		InterpolateParams:     false,
		EnableUpsert:          true,
	}
}

// SetDefaults sets appropriate defaults based on driver
func (config *DatabaseConfig) SetDefaults() {
	if config.Hostname == "" {
		config.Hostname = "localhost"
	}

	if config.Driver == "" {
		config.Driver = "postgresql"
	}

	if config.Port == 0 {
		if config.Driver == "postgresql" {
			config.Port = 5432
		} else {
			config.Port = 3306
		}
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
}

// DatabaseConfigFromMap creates a DatabaseConfig from a map[string]interface{}
// QUYNGUYEN Helper function to convert config map to DatabaseConfig
func DatabaseConfigFromMap(config map[string]interface{}) DatabaseConfig {
	dbConfig := NewDatabaseConfig()

	// Get driver
	if d, ok := config["driver"].(string); ok && d != "" {
		dbConfig.Driver = d
	}

	// Get hostname
	if h, ok := config["hostname"].(string); ok && h != "" {
		dbConfig.Hostname = h
	}

	// Get port
	if p, ok := config["port"].(int); ok && p > 0 {
		dbConfig.Port = p
	} else if p, ok := config["port"].(float64); ok && p > 0 {
		dbConfig.Port = int(p)
	}

	// Get username
	if u, ok := config["username"].(string); ok && u != "" {
		dbConfig.Username = u
	}

	// Get password
	if p, ok := config["password"].(string); ok {
		dbConfig.Password = p
	}

	// Get database
	if db, ok := config["database"].(string); ok && db != "" {
		dbConfig.Database = db
	}

	// Get timeout
	if t, ok := config["timeout"].(time.Duration); ok && t > 0 {
		dbConfig.Timeout = t
	} else if t, ok := config["timeout"].(string); ok && t != "" {
		if parsed, err := time.ParseDuration(t); err == nil {
			dbConfig.Timeout = parsed
		}
	}

	// Get schema
	if s, ok := config["schema"].(string); ok && s != "" {
		dbConfig.Schema = s
	}

	// Get sslmode
	if ssl, ok := config["sslmode"].(string); ok && ssl != "" {
		dbConfig.SSLMode = ssl
	}

	// Get connection pool settings
	if idle, ok := config["connection_max_idle"].(int); ok && idle > 0 {
		dbConfig.ConnectionMaxIdle = idle
	} else if idle, ok := config["connection_max_idle"].(float64); ok && idle > 0 {
		dbConfig.ConnectionMaxIdle = int(idle)
	}

	if open, ok := config["connection_max_open"].(int); ok && open > 0 {
		dbConfig.ConnectionMaxOpen = open
	} else if open, ok := config["connection_max_open"].(float64); ok && open > 0 {
		dbConfig.ConnectionMaxOpen = int(open)
	}

	if lifetime, ok := config["connection_max_lifetime_seconds"].(int); ok && lifetime >= 0 {
		dbConfig.ConnectionMaxLifetime = time.Duration(lifetime) * time.Second
	} else if lifetime, ok := config["connection_max_lifetime_seconds"].(float64); ok && lifetime >= 0 {
		dbConfig.ConnectionMaxLifetime = time.Duration(lifetime) * time.Second
	}

	// Get interpolate params
	if interp, ok := config["interpolateParams"].(bool); ok {
		dbConfig.InterpolateParams = interp
	}

	// Get enable upsert
	// Note: This option should be used by individual store implementations
	// to control whether to use UPSERT operations (INSERT ... ON CONFLICT UPDATE)
	// or separate INSERT/UPDATE operations. Currently, all stores use MySQL dialect
	// so the actual implementation would need to handle this at the query level.
	if upsert, ok := config["enableUpsert"].(bool); ok {
		dbConfig.EnableUpsert = upsert
	}

	// Set defaults after parsing all values
	dbConfig.SetDefaults()

	return dbConfig
}

//QUYNGUYEN end

// ConnectToDatabase creates a database connection using the provided configuration
// QUYNGUYEN Configure database connection based on driver
func ConnectToDatabase(config DatabaseConfig) (*bun.DB, error) {
	config.SetDefaults()

	var sqldb *sql.DB
	var db *bun.DB
	var err error

	switch config.Driver {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.Username, config.Password, config.Hostname, config.Port, config.Database)
		if config.InterpolateParams {
			dsn += "&interpolateParams=true"
		}
		sqldb, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open mysql database: %w", err)
		}
		db = bun.NewDB(sqldb, mysqldialect.New())
	case "postgresql":
		// For PostgreSQL, build connection string and use pgx pool with proper dialect
		connectTimeoutSeconds := int(config.Timeout.Seconds())
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&search_path=%s&connect_timeout=%d",
			config.Username, config.Password, config.Hostname, config.Port, config.Database,
			config.SSLMode, config.Schema, connectTimeoutSeconds)

		pgxConfig, err := pgxpool.ParseConfig(connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse postgresql pool config: %w", err)
		}
		pgxConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

		pool, err := pgxpool.NewWithConfig(context.Background(), pgxConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgresql connection pool: %w", err)
		}

		sqldb := stdlib.OpenDBFromPool(pool)
		db = bun.NewDB(sqldb, pgdialect.New())

	case "sqlite":
		// For SQLite, database is the file path
		sqldb, err = sql.Open("sqlite", config.Database)
		if err != nil {
			return nil, fmt.Errorf("failed to open sqlite database: %w", err)
		}
		// Use MySQL dialect as fallback since SQLite dialect is not available
		// Note: This may have limitations for SQLite-specific features
		db = bun.NewDB(sqldb, mysqldialect.New())
	default:
		return nil, fmt.Errorf("unsupported database driver: %s. Supported drivers: mysql, postgresql, sqlite", config.Driver)
	}
	//QUYNGUYEN end

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	if config.ConnectionMaxOpen > 0 {
		db.SetMaxOpenConns(config.ConnectionMaxOpen)
	}
	if config.ConnectionMaxIdle > 0 {
		db.SetMaxIdleConns(config.ConnectionMaxIdle)
	}
	if config.ConnectionMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnectionMaxLifetime)
	}

	return db, nil
}

// ConnectToDatabaseWithTimeout creates a database connection with custom timeout settings
// This is useful for stores that need different timeout configurations
func ConnectToDatabaseWithTimeout(config DatabaseConfig, maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) (*bun.DB, error) {
	db, err := ConnectToDatabase(config)
	if err != nil {
		return nil, err
	}

	// Override connection pool settings if provided
	if maxOpenConns > 0 {
		db.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 0 {
		db.SetMaxIdleConns(maxIdleConns)
	}
	if connMaxLifetime > 0 {
		db.SetConnMaxLifetime(connMaxLifetime)
	}

	return db, nil
}

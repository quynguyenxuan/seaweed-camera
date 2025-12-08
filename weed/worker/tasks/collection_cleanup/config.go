package collection_cleanup

import (
	"github.com/seaweedfs/seaweedfs/weed/admin/config"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/worker/tasks/base"
)

// Config extends BaseConfig with collection cleanup-specific settings
type Config struct {
	base.BaseConfig

	// DefaultTtlDays specifies the default TTL in days for collection cleanup
	DefaultTtlDays int32 `json:"default_ttl_days" yaml:"default_ttl_days"`
}

// DefaultConfig returns the default configuration for collection cleanup
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: base.BaseConfig{
			Enabled:             true,        // Disabled by default
			ScanIntervalSeconds: 60 * 60, // 24 hours
			MaxConcurrent:       1,            // Only one cleanup at a time
		},
		DefaultTtlDays: 30*12*10, // Default 30 days
	}
}

// NewDefaultConfig creates a new configuration with default values
func NewDefaultConfig() *Config {
	return DefaultConfig()
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return DefaultConfig()
}

// GetDefaultTtlDays returns the default TTL in days
func (c *Config) GetDefaultTtlDays() int32 {
	if c.DefaultTtlDays <= 0 {
		return 30
	}
	return c.DefaultTtlDays
}

// SetDefaultTtlDays sets the default TTL in days
func (c *Config) SetDefaultTtlDays(days int32) {
	c.DefaultTtlDays = days
}

// IsEnabled returns whether collection cleanup is enabled
func (c *Config) IsEnabled() bool {
	return c.Enabled
}

// SetEnabled sets whether collection cleanup is enabled
func (c *Config) SetEnabled(enabled bool) {
	c.Enabled = enabled
}

// LoadConfigFromPersistence loads configuration from persistence
func LoadConfigFromPersistence(configPersistence interface{}) *Config {
	// In real implementation, this would load from actual persistence
	// For now, return default config
	glog.V(1).Infof("Loading collection cleanup config from persistence")
	return DefaultConfig()
}

// GetConfigSpec returns the configuration specification for this task
func GetConfigSpec() base.ConfigSpec {
	return base.ConfigSpec{
		Fields: []*config.Field{
			// Schema definition would go here
		},
	}
}

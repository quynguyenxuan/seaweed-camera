package collection_cleanup

import (
	"fmt"

	"github.com/seaweedfs/seaweedfs/weed/admin/config"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/worker_pb"
	"github.com/seaweedfs/seaweedfs/weed/worker/tasks/base"
)

// Config extends BaseConfig with collection cleanup-specific settings
type Config struct {
	base.BaseConfig

	// CollectionPattern specifies regex pattern to match collections for cleanup
	CollectionPattern string `json:"collection_pattern" yaml:"collection_pattern"`

	// DryRun specifies whether to run in dry-run mode (no actual deletion)
	DryRun bool `json:"dry_run" yaml:"dry_run"`
}

// DefaultConfig returns the default configuration for collection cleanup
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: base.BaseConfig{
			Enabled:             true,    // Enabled by default
			ScanIntervalSeconds: 60 * 60, // 24 hours
			MaxConcurrent:       1,       // Only one cleanup at a time
		},
		CollectionPattern: ".*",  // Match all collections
		DryRun:            false, // Actually delete files by default
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

// GetCollectionPattern returns the collection pattern regex
func (c *Config) GetCollectionPattern() string {
	if c.CollectionPattern == "" {
		return ".*"
	}
	return c.CollectionPattern
}

// SetCollectionPattern sets the collection pattern regex
func (c *Config) SetCollectionPattern(pattern string) {
	c.CollectionPattern = pattern
}

// IsDryRun returns whether to run in dry-run mode
func (c *Config) IsDryRun() bool {
	return c.DryRun
}

// SetDryRun sets whether to run in dry-run mode
func (c *Config) SetDryRun(dryRun bool) {
	c.DryRun = dryRun
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
			{
				Name:         "enabled",
				JSONName:     "enabled",
				Type:         config.FieldTypeBool,
				DefaultValue: true,
				Required:     false,
				DisplayName:  "Enable Collection Cleanup Tasks",
				Description:  "Whether collection cleanup tasks should be automatically created",
				HelpText:     "Toggle this to enable or disable automatic collection cleanup task generation",
				InputType:    "checkbox",
				CSSClasses:   "form-check-input",
			},
			{
				Name:         "scan_interval_seconds",
				JSONName:     "scan_interval_seconds",
				Type:         config.FieldTypeInterval,
				DefaultValue: 60 * 60,
				MinValue:     5 * 60,
				MaxValue:     24 * 60 * 60,
				Required:     true,
				DisplayName:  "Scan Interval",
				Description:  "How often to scan for collections needing cleanup",
				HelpText:     "The system will check for collections requiring cleanup at this interval",
				Placeholder:  "3600 (1 hour)",
				Unit:         config.UnitMinutes,
				InputType:    "interval",
				CSSClasses:   "form-control",
			},
			{
				Name:         "max_concurrent",
				JSONName:     "max_concurrent",
				Type:         config.FieldTypeInt,
				DefaultValue: 1,
				MinValue:     1,
				MaxValue:     3,
				Required:     true,
				DisplayName:  "Max Concurrent Tasks",
				Description:  "Maximum number of collection cleanup tasks that can run simultaneously",
				HelpText:     "Limits the number of cleanup operations running at the same time",
				Placeholder:  "1 (default)",
				Unit:         config.UnitCount,
				InputType:    "number",
				CSSClasses:   "form-control",
			},
			{
				Name:         "collection_pattern",
				JSONName:     "collection_pattern",
				Type:         config.FieldTypeString,
				DefaultValue: ".*",
				Required:     false,
				DisplayName:  "Collection Pattern",
				Description:  "Regex pattern to match collection names for cleanup",
				HelpText:     "Only collections matching this pattern will be processed",
				Placeholder:  ".* (all collections)",
				InputType:    "text",
				CSSClasses:   "form-control",
			},
			{
				Name:         "dry_run",
				JSONName:     "dry_run",
				Type:         config.FieldTypeBool,
				DefaultValue: false,
				Required:     false,
				DisplayName:  "Dry Run Mode",
				Description:  "Run in dry-run mode without actually deleting files",
				HelpText:     "Enable this to test cleanup operations without file deletion",
				InputType:    "checkbox",
				CSSClasses:   "form-check-input",
			},
		},
	}
}

// ToTaskPolicy converts configuration to a TaskPolicy protobuf message
func (c *Config) ToTaskPolicy() *worker_pb.TaskPolicy {
	// For now, use generic task config since CollectionCleanupTaskConfig is not yet defined in protobuf
	return &worker_pb.TaskPolicy{
		Enabled:               c.Enabled,
		MaxConcurrent:         int32(c.MaxConcurrent),
		RepeatIntervalSeconds: int32(c.ScanIntervalSeconds),
		CheckIntervalSeconds:  int32(c.ScanIntervalSeconds),
		// TaskConfig will be added when CollectionCleanupTaskConfig is defined in protobuf
	}
}

// FromTaskPolicy loads configuration from a TaskPolicy protobuf message
func (c *Config) FromTaskPolicy(policy *worker_pb.TaskPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy is nil")
	}

	// Set general TaskPolicy fields
	c.Enabled = policy.Enabled
	c.MaxConcurrent = int(policy.MaxConcurrent)
	c.ScanIntervalSeconds = int(policy.RepeatIntervalSeconds)

	// Collection cleanup specific fields will be loaded when protobuf is updated
	// For now, keep existing values

	return nil
}

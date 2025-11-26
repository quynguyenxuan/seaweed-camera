package collection_cleanup

import (
	"fmt"

	"github.com/seaweedfs/seaweedfs/weed/pb/worker_pb"
	"github.com/seaweedfs/seaweedfs/weed/worker/tasks"
	"github.com/seaweedfs/seaweedfs/weed/worker/types"
)

// CollectionCleanupUIProvider provides UI configuration for collection cleanup tasks
type CollectionCleanupUIProvider struct {
	*tasks.BaseUIProvider
}

// NewCollectionCleanupUIProvider creates a new UI provider for collection cleanup
func NewCollectionCleanupUIProvider() *CollectionCleanupUIProvider {
	return &CollectionCleanupUIProvider{
		BaseUIProvider: tasks.NewBaseUIProvider(
			types.TaskTypeCollectionCleanup,
			"Collection Cleanup",
			"Cleans up expired files in collections based on TTL configuration",
			"cleanup",
			func() *tasks.TaskConfigSchema {
				return &tasks.TaskConfigSchema{
					// Schema implementation would go here
				}
			},
			func() types.TaskConfig {
				return DefaultConfig()
			},
			func(policy *worker_pb.TaskPolicy) error {
				// Apply task policy implementation
				return nil
			},
			func(config types.TaskConfig) error {
				// Apply task config implementation
				return nil
			},
		),
	}
}

// GetSchema returns the configuration schema for collection cleanup
func (ui *CollectionCleanupUIProvider) GetSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"default_ttl_days": map[string]interface{}{
				"type":        "integer",
				"title":       "Default TTL Days",
				"description": "Default time-to-live in days for collection cleanup",
				"minimum":     1,
				"maximum":     365,
				"default":     30,
			},
			"max_concurrent_tasks": map[string]interface{}{
				"type":        "integer",
				"title":       "Max Concurrent Tasks",
				"description": "Maximum number of concurrent cleanup tasks",
				"minimum":     1,
				"maximum":     10,
				"default":     1,
			},
			"repeat_interval": map[string]interface{}{
				"type":        "string",
				"title":       "Repeat Interval",
				"description": "How often to run cleanup tasks (e.g., '24h', '1d')",
				"pattern":     "^[0-9]+[smhd]$",
				"default":     "24h",
			},
			"enabled": map[string]interface{}{
				"type":        "boolean",
				"title":       "Enabled",
				"description": "Whether collection cleanup is enabled",
				"default":     false,
			},
		},
		"required": []string{"enabled"},
	}
}

// GetCurrentConfig returns the current configuration
func (ui *CollectionCleanupUIProvider) GetCurrentConfig() interface{} {
	config := DefaultConfig()

	return map[string]interface{}{
		"ttl_days":       config.GetDefaultTtlDays(),
		"max_concurrent": config.MaxConcurrent,
		"scan_interval":  config.ScanIntervalSeconds,
		"enabled":        config.Enabled,
	}
}

// ValidateConfig validates the provided configuration
func (ui *CollectionCleanupUIProvider) ValidateConfig(config interface{}) error {
	// For now, just basic validation
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	// In real implementation, would validate specific fields
	return nil
}

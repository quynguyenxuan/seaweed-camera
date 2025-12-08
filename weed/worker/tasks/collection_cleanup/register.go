package collection_cleanup

import (
	"fmt"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/worker_pb"
	"github.com/seaweedfs/seaweedfs/weed/worker/tasks"
	"github.com/seaweedfs/seaweedfs/weed/worker/tasks/base"
	"github.com/seaweedfs/seaweedfs/weed/worker/types"
)

// Global variable to hold the task definition for configuration updates
var globalTaskDef *base.TaskDefinition

// Auto-register this task when the package is imported
func init() {
	RegisterCollectionCleanupTask()

	// Register config updater
	tasks.AutoRegisterConfigUpdater(types.TaskTypeCollectionCleanup, UpdateConfigFromPersistence)
}

// RegisterCollectionCleanupTask registers the collection cleanup task with the new architecture
func RegisterCollectionCleanupTask() {
	// Create configuration instance
	config := NewDefaultConfig()

	// Create complete task definition
	taskDef := &base.TaskDefinition{
		Type:         types.TaskTypeCollectionCleanup,
		Name:         "collection_cleanup",
		DisplayName:  "Collection Cleanup",
		Description:  "Cleans up expired files in collections based on TTL configuration",
		Icon:         "fas fa-trash-alt text-danger",
		Capabilities: []string{"collection_cleanup", "cleanup"},

		Config:     config,
		ConfigSpec: GetConfigSpec(),
		CreateTask: func(params *worker_pb.TaskParams) (types.Task, error) {
			if params == nil {
				return nil, fmt.Errorf("task parameters are required")
			}

			// Get collection name from params or use default
			collection := params.Collection
			if collection == "" {
				collection = "default_collection"
			}

			// Get server address or use default
			server := "localhost:9333"
			if len(params.Sources) > 0 {
				server = params.Sources[0].Node
			}

			return NewCollectionCleanupTask(
				fmt.Sprintf("collection_cleanup_%s", collection),
				server,
				collection,
			), nil
		},
		DetectionFunc:  Detection,
		ScanInterval:   1 * time.Hour,
		SchedulingFunc: Scheduling,
		MaxConcurrent:  1,
		RepeatInterval: 24 * time.Hour,
	}

	// Store task definition globally for configuration updates
	globalTaskDef = taskDef

	// Register everything with a single function call!
	base.RegisterTask(taskDef)
}

// UpdateConfigFromPersistence updates the collection cleanup configuration from persistence
func UpdateConfigFromPersistence(configPersistence interface{}) error {
	if globalTaskDef == nil {
		return fmt.Errorf("collection cleanup task not registered")
	}

	// Load configuration from persistence
	newConfig := LoadConfigFromPersistence(configPersistence)
	if newConfig == nil {
		return fmt.Errorf("failed to load configuration from persistence")
	}

	// Update the task definition's config
	globalTaskDef.Config = newConfig

	glog.V(1).Infof("Updated collection cleanup task configuration from persistence")
	return nil
}

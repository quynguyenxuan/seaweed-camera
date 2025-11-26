package collection_cleanup

import (
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/worker_pb"
	"github.com/seaweedfs/seaweedfs/weed/worker/tasks"
	"github.com/seaweedfs/seaweedfs/weed/worker/types"
)

// CollectionCleanupFactory implements TaskFactory interface
type CollectionCleanupFactory struct {
	*tasks.BaseTaskFactory
}

// NewCollectionCleanupFactory creates a new factory
func NewCollectionCleanupFactory() *CollectionCleanupFactory {
	return &CollectionCleanupFactory{
		BaseTaskFactory: tasks.NewBaseTaskFactory(
			types.TaskTypeCollectionCleanup,
			[]string{string(types.TaskTypeCollectionCleanup)},
			"Factory for creating collection cleanup tasks",
		),
	}
}

// Create implements TaskFactory interface
func (f *CollectionCleanupFactory) Create(params *worker_pb.TaskParams) (types.Task, error) {
	// Get server address (would typically come from worker configuration)
	server := "localhost:9333" // Default master address
	
	// Create task instance
	task := NewCollectionCleanupTask("collection_cleanup_task", server, "default_collection")
	
	glog.V(3).Infof("Created collection cleanup task")
	return task, nil
}

// Type implements TaskFactory interface
func (f *CollectionCleanupFactory) Type() string {
	return string(types.TaskTypeCollectionCleanup)
}

func init() {
	// Auto-register config updater
	tasks.AutoRegisterConfigUpdater(types.TaskTypeCollectionCleanup, UpdateConfigFromPersistence)

	// Create and register factory
	factory := NewCollectionCleanupFactory()

	// Register with global registries
	tasks.AutoRegister(types.TaskTypeCollectionCleanup, factory)

	glog.V(1).Infof("Collection cleanup task registered successfully")
}

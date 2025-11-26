package collection_cleanup

import (
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/worker/tasks/base"
	"github.com/seaweedfs/seaweedfs/weed/worker/types"
)

// Scheduling determines if a collection cleanup task can be scheduled
func Scheduling(task *types.TaskInput, runningTasks []*types.TaskInput, availableWorkers []*types.WorkerData, config base.TaskConfig) bool {
	cleanupConfig := config.(*Config)

	glog.V(4).Infof("Collection cleanup scheduling check: enabled=%v, max_concurrent=%d",
		cleanupConfig.Enabled, cleanupConfig.MaxConcurrent)

	// Check if collection cleanup is enabled
	if !cleanupConfig.Enabled {
		glog.V(4).Infof("Collection cleanup is disabled")
		return false
	}

	// Check if we have available workers
	if len(availableWorkers) == 0 {
		glog.V(4).Infof("No available workers for collection cleanup")
		return false
	}

	// Count running collection cleanup tasks
	runningCleanupCount := 0
	for _, runningTask := range runningTasks {
		if runningTask.Type == types.TaskTypeCollectionCleanup {
			runningCleanupCount++
		}
	}

	// Check if we've reached the maximum concurrent tasks
	if runningCleanupCount >= cleanupConfig.MaxConcurrent {
		glog.V(4).Infof("Collection cleanup limit reached: %d/%d",
			runningCleanupCount, cleanupConfig.MaxConcurrent)
		return false
	}

	// Check if any available worker has collection cleanup capability
	for _, worker := range availableWorkers {
		for _, capability := range worker.Capabilities {
			if capability == types.TaskTypeCollectionCleanup {
				glog.V(4).Infof("Collection cleanup can be scheduled on worker %s", worker.ID)
				return true
			}
		}
	}

	glog.V(4).Infof("No workers with collection cleanup capability available")
	return false
}

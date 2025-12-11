package collection_cleanup

import (
	"fmt"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/worker_pb"
	"github.com/seaweedfs/seaweedfs/weed/worker/tasks/base"
	"github.com/seaweedfs/seaweedfs/weed/worker/types"
)

// Detection implements the detection logic for collection cleanup tasks
//QUYNGUYEN: Refactored to create ONE task per pattern instead of individual tasks per collection
func Detection(metrics []*types.VolumeHealthMetrics, clusterInfo *types.ClusterInfo, config base.TaskConfig) ([]*types.TaskDetectionResult, error) {
	if !config.IsEnabled() {
		return nil, nil
	}

	cleanupConfig := config.(*Config)
	var results []*types.TaskDetectionResult

	glog.V(3).Infof("Starting collection cleanup detection")

	// Get collection pattern from config
	pattern := cleanupConfig.GetCollectionPattern()
	if pattern == "" {
		pattern = ".*" // Default: match all collections
	}

	// Get admin server address from cluster info for task execution
	adminServer := ""
	if clusterInfo != nil && len(clusterInfo.Servers) > 0 {
		adminServer = clusterInfo.Servers[0].Address
	}

	if adminServer == "" {
		glog.Errorf("No admin server available in cluster info")
		return nil, fmt.Errorf("no admin server available")
	}

	// Create ONE task with the pattern
	taskID := fmt.Sprintf("collection_cleanup_pattern_%d", time.Now().Unix())
	
	result := &types.TaskDetectionResult{
		TaskID:     taskID,
		TaskType:   types.TaskTypeCollectionCleanup,
		Collection: pattern, // Store pattern in Collection field
		Priority:   types.TaskPriorityNormal,
		Reason:     fmt.Sprintf("Collection cleanup for pattern '%s'", pattern),
		ScheduleAt: time.Now(),
	}

	// Create typed parameters for collection cleanup task
	result.TypedParams = createCollectionCleanupTaskParams(result, pattern, adminServer)
	results = append(results, result)

	glog.V(3).Infof("Collection cleanup detection completed: created 1 task for pattern '%s'", pattern)
	return results, nil
}
//QUYNGUYEN end

// createCollectionCleanupTaskParams creates typed parameters for collection cleanup tasks
func createCollectionCleanupTaskParams(task *types.TaskDetectionResult, pattern string, adminServer string) *worker_pb.TaskParams {
	return &worker_pb.TaskParams{
		TaskId:     task.TaskID,
		VolumeId:   0, // Collection cleanup doesn't target specific volume
		Collection: pattern, // Store pattern for the task

		Sources: []*worker_pb.TaskSource{
			{
				Node:          adminServer, // Admin server address for RPC call
				VolumeId:      0,
				EstimatedSize: 0,
				DataCenter:    "default",
				Rack:          "default",
			},
		},

		TaskParams: &worker_pb.TaskParams_VacuumParams{
			VacuumParams: &worker_pb.VacuumTaskParams{
				// Use existing VacuumTaskParams as placeholder
				// In real implementation, CollectionCleanupTaskParams should be added to protobuf
				GarbageThreshold: 0.3,
				ForceVacuum:      false,
			},
		},
	}
}

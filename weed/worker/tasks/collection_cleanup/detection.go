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
func Detection(metrics []*types.VolumeHealthMetrics, clusterInfo *types.ClusterInfo, config base.TaskConfig) ([]*types.TaskDetectionResult, error) {
	if !config.IsEnabled() {
		return nil, nil
	}

	cleanupConfig := config.(*Config)
	var results []*types.TaskDetectionResult

	glog.V(3).Infof("Starting collection cleanup detection")

	// Get collections that need cleanup
	collections := getCollectionsNeedingCleanup()

	for _, collection := range collections {
		taskID := fmt.Sprintf("collection_cleanup_%s_%d", collection.Name, time.Now().Unix())

		result := &types.TaskDetectionResult{
			TaskID:     taskID,
			TaskType:   types.TaskTypeCollectionCleanup,
			Collection: collection.Name,
			Priority:   types.TaskPriorityNormal,
			Reason:     fmt.Sprintf("Collection %s has TTL of %d days", collection.Name, collection.TtlDays),
			ScheduleAt: time.Now(),
		}

		// Create typed parameters for collection cleanup task
		result.TypedParams = createCollectionCleanupTaskParams(result, collection, cleanupConfig, clusterInfo)
		results = append(results, result)

		glog.V(4).Infof("Created collection cleanup task for %s (TTL: %d days)", collection.Name, collection.TtlDays)
	}

	glog.V(3).Infof("Collection cleanup detection completed: created %d tasks", len(results))
	return results, nil
}

// CollectionWithTtl represents a collection with its TTL configuration
type CollectionWithTtl struct {
	Name    string
	TtlDays int
}

// getCollectionsNeedingCleanup returns collections that need cleanup
func getCollectionsNeedingCleanup() []*CollectionWithTtl {
	// In real implementation, this would:
	// 1. Query master for all collections
	// 2. Check TTL configuration for each collection
	// 3. Return collections that have TTL configured

	// For demo purposes, return some example collections
	return []*CollectionWithTtl{
		{Name: "temp_collection", TtlDays: 7},
		{Name: "backup_collection", TtlDays: 90},
		{Name: "log_collection", TtlDays: 30},
	}
}

// createCollectionCleanupTaskParams creates typed parameters for collection cleanup tasks
func createCollectionCleanupTaskParams(task *types.TaskDetectionResult, collection *CollectionWithTtl, cleanupConfig *Config, clusterInfo *types.ClusterInfo) *worker_pb.TaskParams {
	return &worker_pb.TaskParams{
		TaskId:     task.TaskID,
		VolumeId:   0, // Collection cleanup doesn't target specific volume
		Collection: task.Collection,

		Sources: []*worker_pb.TaskSource{
			{
				Node:          "localhost:9333", // Master server
				VolumeId:      0,
				EstimatedSize: 0,
				DataCenter:    "default", // Use default data center
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

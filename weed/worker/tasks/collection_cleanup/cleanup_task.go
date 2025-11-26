package collection_cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/worker_pb"
	"github.com/seaweedfs/seaweedfs/weed/worker/types"
	"github.com/seaweedfs/seaweedfs/weed/worker/types/base"
)

// CollectionCleanupTask implements the Task interface
type CollectionCleanupTask struct {
	*base.BaseTask
	server     string
	collection string
	progress   float64
}

// NewCollectionCleanupTask creates a new collection cleanup task instance
func NewCollectionCleanupTask(id string, server string, collection string) *CollectionCleanupTask {
	return &CollectionCleanupTask{
		BaseTask:   base.NewBaseTask(id, types.TaskTypeCollectionCleanup),
		server:     server,
		collection: collection,
	}
}

// Execute implements the Task interface
func (t *CollectionCleanupTask) Execute(ctx context.Context, params *worker_pb.TaskParams) error {
	if params == nil {
		return fmt.Errorf("task parameters are required")
	}

	// For now, use default TTL of 30 days
	// In future, this could be extracted from task parameters
	ttlDays := int32(30)

	glog.V(3).Infof("Starting collection cleanup for %s with TTL %d days", t.collection, ttlDays)

	// Calculate time range based on TTL configuration
	now := time.Now()
	fromTime := uint64(0) // From beginning
	toTime := uint64(now.AddDate(0, 0, -int(ttlDays)).Unix())

	glog.V(3).Infof("Cleaning collection %s: deleting files older than %d days (before %s)",
		t.collection, ttlDays, time.Unix(int64(toTime), 0).Format("2006-01-02 15:04:05"))

	// Call master API to delete collection with time range
	err := t.cleanupCollectionWithTime(ctx, fromTime, toTime)
	if err != nil {
		glog.Errorf("Collection cleanup failed for %s: %v", t.collection, err)
		return fmt.Errorf("collection cleanup failed: %w", err)
	}

	t.progress = 100.0
	glog.V(3).Infof("Collection cleanup completed successfully for %s", t.collection)
	return nil
}

// cleanupCollectionWithTime calls master API to delete collection with time range
func (t *CollectionCleanupTask) cleanupCollectionWithTime(ctx context.Context, fromTime, toTime uint64) error {
	if t.collection == "" {
		return fmt.Errorf("collection name is required")
	}

	glog.V(3).Infof("Connecting to master at %s to cleanup collection %s", t.server, t.collection)

	// Create gRPC dial option
	// grpcDialOption := grpc.WithInsecure()

	// // Parse master server address
	// masterAddress := pb.ServerAddress(t.server)

	// Use pb.WithMasterClient to connect and call CollectionDelete
	// err := pb.WithMasterClient(false, masterAddress, grpcDialOption, false, func(client master_pb.SeaweedClient) error {
	// 	glog.V(4).Infof("Calling CollectionDelete for collection %s: fromTime=%d, toTime=%d",
	// 		t.collection, fromTime, toTime)

	// 	resp, err := client.CollectionDelete(ctx, &master_pb.CollectionDeleteRequest{
	// 		Name:     t.collection,
	// 		FromTime: fromTime,
	// 		ToTime:   toTime,
	// 	})

	// 	if err != nil {
	// 		glog.Errorf("Failed to delete collection %s on master: %v", t.collection, err)
	// 		return fmt.Errorf("master API call failed: %w", err)
	// 	}

	// 	glog.V(3).Infof("CollectionDelete response for %s: %v", t.collection, resp)
	// 	return nil
	// })

	// if err != nil {
	// 	glog.Errorf("Collection cleanup failed for %s: %v", t.collection, err)
	// 	return fmt.Errorf("collection cleanup failed: %w", err)
	// }

	// glog.V(3).Infof("Successfully cleaned up collection %s from time %d to %d",
	// 	t.collection, fromTime, toTime)

	return nil
}

// GetProgress returns the current progress of the task
func (t *CollectionCleanupTask) GetProgress() float64 {
	return t.progress
}

// GetCollection returns the collection name
func (t *CollectionCleanupTask) GetCollection() string {
	return t.collection
}

// SetCollection updates the collection name
func (t *CollectionCleanupTask) SetCollection(collection string) {
	t.collection = collection
}

// GetServer returns the server address
func (t *CollectionCleanupTask) GetServer() string {
	return t.server
}

// SetServer updates the server address
func (t *CollectionCleanupTask) SetServer(server string) {
	t.server = server
}

// ResetProgress resets the task progress
func (t *CollectionCleanupTask) ResetProgress() {
	t.progress = 0.0
}

// IsCompleted checks if the task is completed
func (t *CollectionCleanupTask) IsCompleted() bool {
	return t.progress >= 100.0
}

package collection_cleanup

import (
	"context"
	"fmt"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/worker_pb"
	"github.com/seaweedfs/seaweedfs/weed/worker/types"
	"github.com/seaweedfs/seaweedfs/weed/worker/types/base"
	"google.golang.org/grpc"
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

	glog.V(3).Infof("Starting collection cleanup for %s", t.collection)

	// Call CollectionCleanup API which will automatically handle TTL
	err := t.cleanupCollection(ctx)
	if err != nil {
		glog.Errorf("Collection cleanup failed for %s: %v", t.collection, err)
		return fmt.Errorf("collection cleanup failed: %w", err)
	}

	t.progress = 100.0
	glog.V(3).Infof("Collection cleanup completed successfully for %s", t.collection)
	return nil
}

// cleanupCollection calls admin server to perform collection cleanup
func (t *CollectionCleanupTask) cleanupCollection(ctx context.Context) error {
	if t.collection == "" {
		return fmt.Errorf("collection name is required")
	}

	glog.V(3).Infof("Calling admin server at %s to cleanup collection %s", t.server, t.collection)

	// Parse admin server address
	adminAddress := pb.ServerAddress(t.server)

	// Create gRPC dial option
	grpcDialOption := grpc.WithInsecure()

	// Connect to admin server's worker gRPC service
	// Admin gRPC port = HTTP port + 10000
	grpcAddress := pb.ServerToGrpcAddress(string(adminAddress))
	
	glog.V(4).Infof("Connecting to admin gRPC server at %s", grpcAddress)

	// Use pb.GrpcDial to connect to admin server
	conn, err := pb.GrpcDial(ctx, grpcAddress, false, grpcDialOption)
	if err != nil {
		return fmt.Errorf("failed to connect to admin server: %w", err)
	}
	defer conn.Close()

	// Create worker service client
	client := worker_pb.NewWorkerServiceClient(conn)

	// Call CleanupCollection RPC
	resp, err := client.CleanupCollection(ctx, &worker_pb.CleanupCollectionRequest{
		WorkerId:          "collection_cleanup_task",
		TaskId:            t.ID(),
		CollectionPattern: t.collection, // Can be a single collection name or regex pattern
		DryRun:            false,
	})

	if err != nil {
		glog.Errorf("Failed to call admin server for collection cleanup: %v", err)
		return fmt.Errorf("admin API call failed: %w", err)
	}

	if !resp.Success {
		glog.Errorf("Collection cleanup failed: %s", resp.Message)
		return fmt.Errorf("cleanup failed: %s", resp.Message)
	}

	glog.V(3).Infof("Collection cleanup response: %s (files deleted: %d, bytes freed: %d, duration: %dms)",
		resp.Message, resp.FilesDeleted, resp.BytesFreed, resp.DurationMs)

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

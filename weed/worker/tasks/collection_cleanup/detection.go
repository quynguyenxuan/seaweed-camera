package collection_cleanup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/master_pb"
	"github.com/seaweedfs/seaweedfs/weed/worker/types"
)

// CollectionCleanupDetector detects collections that need cleanup
type CollectionCleanupDetector struct {
	// No base detector needed for now
}

// NewCollectionCleanupDetector creates a new collection cleanup detector
func NewCollectionCleanupDetector() *CollectionCleanupDetector {
	return &CollectionCleanupDetector{}
}

// DetectTasks implements the Detector interface
func (d *CollectionCleanupDetector) DetectTasks(ctx context.Context, masterClient master_pb.SeaweedClient) ([]*types.TaskDetectionResult, error) {
	glog.V(3).Infof("Starting collection cleanup detection")

	var tasks []*types.TaskDetectionResult

	// Get all collections with TTL configuration
	collections, err := d.getCollectionsWithTtl(ctx, masterClient)
	if err != nil {
		glog.Errorf("Failed to get collections with TTL: %v", err)
		return nil, fmt.Errorf("failed to get collections with TTL: %w", err)
	}

	glog.V(3).Infof("Found %d collections with TTL configuration", len(collections))

	// Create cleanup task for each collection
	for _, collection := range collections {
		task := &types.TaskDetectionResult{
			TaskID:     fmt.Sprintf("collection_cleanup_%s_%d", collection.Name, time.Now().Unix()),
			TaskType:   types.TaskTypeCollectionCleanup,
			Collection: collection.Name,
			Priority:   types.TaskPriorityNormal,
			Reason:     fmt.Sprintf("Collection %s has TTL of %d days", collection.Name, collection.TtlDays),
			ScheduleAt: time.Now(),
		}
		tasks = append(tasks, task)
		glog.V(4).Infof("Created collection cleanup task for %s (TTL: %d days)", collection.Name, collection.TtlDays)
	}

	glog.V(3).Infof("Collection cleanup detection completed: created %d tasks", len(tasks))
	return tasks, nil
}

// CollectionWithTtl represents a collection with its TTL configuration
type CollectionWithTtl struct {
	Name    string
	TtlDays int
}

// getCollectionsWithTtl gets all collections that have TTL configuration
func (d *CollectionCleanupDetector) getCollectionsWithTtl(ctx context.Context, masterClient master_pb.SeaweedClient) ([]*CollectionWithTtl, error) {
	// This would typically involve:
	// 1. Getting all collections from master
	// 2. Getting TTL configuration from filer configuration
	// 3. Filtering collections that have TTL configured

	// For now, we'll simulate with some example collections
	// In real implementation, this would call actual master APIs

	var collections []*CollectionWithTtl

	// Simulate some collections with TTL
	collections = append(collections, &CollectionWithTtl{
		Name:    "temp_collection",
		TtlDays: 7,
	})
	collections = append(collections, &CollectionWithTtl{
		Name:    "backup_collection",
		TtlDays: 90,
	})
	collections = append(collections, &CollectionWithTtl{
		Name:    "log_collection",
		TtlDays: 30,
	})

	return collections, nil
}

// getCollectionTtl gets the TTL configuration for a collection
// This is a placeholder - in real implementation, this would read from filer configuration
func (d *CollectionCleanupDetector) getCollectionTtl(collectionName string) int {
	// Placeholder implementation
	// In real scenario, this would:
	// 1. Read filer configuration
	// 2. Find TTL settings for this collection
	// 3. Parse TTL string (e.g., "30d", "7d") to days

	// For demo purposes, return different TTLs for different collections
	if strings.Contains(collectionName, "temp") {
		return 7 // 7 days for temp collections
	}
	if strings.Contains(collectionName, "backup") {
		return 90 // 90 days for backup collections
	}
	if strings.Contains(collectionName, "log") {
		return 30 // 30 days for log collections
	}

	// Default TTL for collections that match certain patterns
	if strings.HasPrefix(collectionName, "user_") {
		return 30
	}

	return 0 // No TTL configured
}

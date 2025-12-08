package collection_cleanup

import (
	"context"
	"testing"

	"github.com/seaweedfs/seaweedfs/weed/pb/worker_pb"
	"github.com/seaweedfs/seaweedfs/weed/worker/types"
)

// TestCollectionCleanupTaskCreation tests creating a new collection cleanup task
func TestCollectionCleanupTaskCreation(t *testing.T) {
	taskID := "test-cleanup-1"
	server := "localhost:9333"
	collection := "test_collection"

	task := NewCollectionCleanupTask(taskID, server, collection)

	// Verify task properties
	if task.GetCollection() != collection {
		t.Errorf("Expected collection %s, got %s", collection, task.GetCollection())
	}

	if task.GetServer() != server {
		t.Errorf("Expected server %s, got %s", server, task.GetServer())
	}

	if task.GetProgress() != 0.0 {
		t.Errorf("Expected initial progress 0.0, got %f", task.GetProgress())
	}

	if task.IsCompleted() {
		t.Error("Task should not be completed initially")
	}
}

// TestCollectionCleanupTaskSetters tests the setter methods
func TestCollectionCleanupTaskSetters(t *testing.T) {
	task := NewCollectionCleanupTask("test", "localhost:9333", "original")

	// Test collection setter
	newCollection := "new_collection"
	task.SetCollection(newCollection)
	if task.GetCollection() != newCollection {
		t.Errorf("Expected collection %s, got %s", newCollection, task.GetCollection())
	}

	// Test server setter
	newServer := "localhost:9334"
	task.SetServer(newServer)
	if task.GetServer() != newServer {
		t.Errorf("Expected server %s, got %s", newServer, task.GetServer())
	}

	// Test progress reset
	task.ResetProgress()
	if task.GetProgress() != 0.0 {
		t.Errorf("Expected progress 0.0 after reset, got %f", task.GetProgress())
	}
}

// TestCollectionCleanupTaskType tests the task type
func TestCollectionCleanupTaskType(t *testing.T) {
	task := NewCollectionCleanupTask("test", "localhost:9333", "test")

	// Verify task type
	if task.Type() != types.TaskTypeCollectionCleanup {
		t.Errorf("Expected task type %s, got %s",
			types.TaskTypeCollectionCleanup, task.Type())
	}
}

// TestCollectionCleanupTaskExecution tests task execution (mock)
func TestCollectionCleanupTaskExecution(t *testing.T) {
	// This test would require a running master server to work fully
	// For now, we just test the task creation and basic properties

	task := NewCollectionCleanupTask("test-cleanup", "localhost:9333", "mock_collection")

	// Test with nil parameters (should fail)
	ctx := context.Background()
	err := task.Execute(ctx, nil)
	if err == nil {
		t.Error("Expected error with nil parameters")
	}

	// Test with empty parameters (should work but fail to connect to master)
	params := &worker_pb.TaskParams{}
	err = task.Execute(ctx, params)
	// This will fail because there's no master server running, which is expected
	if err == nil {
		t.Error("Expected connection error without running master server")
	}

	t.Logf("Expected error (no master server): %v", err)
}

// TestCollectionCleanupFactory tests the factory creation
func TestCollectionCleanupFactory(t *testing.T) {
	factory := NewCollectionCleanupFactory()

	// Verify factory properties
	if factory.Type() != string(types.TaskTypeCollectionCleanup) {
		t.Errorf("Expected factory type %s, got %s",
			string(types.TaskTypeCollectionCleanup), factory.Type())
	}

	if len(factory.Capabilities()) == 0 {
		t.Error("Factory should have at least one capability")
	}

	if factory.Description() == "" {
		t.Error("Factory should have a description")
	}
}

// TestCollectionCleanupFactoryCreate tests creating tasks via factory
func TestCollectionCleanupFactoryCreate(t *testing.T) {
	factory := NewCollectionCleanupFactory()
	params := &worker_pb.TaskParams{}

	task, err := factory.Create(params)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if task == nil {
		t.Fatal("Created task is nil")
	}

	// Verify task type
	if task.Type() != types.TaskTypeCollectionCleanup {
		t.Errorf("Expected task type %s, got %s",
			types.TaskTypeCollectionCleanup, task.Type())
	}
}

// TestConfigDefaults tests the default configuration
func TestConfigDefaults(t *testing.T) {
	config := DefaultConfig()

	if config.GetDefaultTtlDays() <= 0 {
		t.Error("Default TTL days should be positive")
	}

	if config.ScanIntervalSeconds <= 0 {
		t.Error("Scan interval should be positive")
	}

	if config.MaxConcurrent <= 0 {
		t.Error("Max concurrent should be positive")
	}
}

// BenchmarkCollectionCleanupTaskCreation benchmarks task creation
func BenchmarkCollectionCleanupTaskCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewCollectionCleanupTask("bench-task", "localhost:9333", "bench_collection")
	}
}

// ExampleCollectionCleanupTask demonstrates how to use the collection cleanup task
func ExampleCollectionCleanupTask() {
	// Create a new collection cleanup task
	task := NewCollectionCleanupTask("cleanup-1", "localhost:9333", "temp_files")

	// Set task properties
	task.SetCollection("backup_data")
	task.SetServer("master.example.com:9333")

	// Get task information
	collection := task.GetCollection()
	server := task.GetServer()
	progress := task.GetProgress()
	completed := task.IsCompleted()

	// Output task information
	println("Collection:", collection)
	println("Server:", server)
	println("Progress:", progress)
	println("Completed:", completed)

	// Reset progress if needed
	task.ResetProgress()

	// Output: Collection: backup_data
	// Output: Server: master.example.com:9333
	// Output: Progress: 0
	// Output: Completed: false
}

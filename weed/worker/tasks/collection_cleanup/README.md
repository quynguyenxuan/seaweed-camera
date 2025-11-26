# Collection Cleanup Task

## Overview

The Collection Cleanup task is a maintenance worker task that automatically cleans up expired files in collections based on TTL (Time-To-Live) configuration. It follows the same architectural pattern as other worker tasks like Vacuum, Balance, and Erasure Coding.

## Features

- **Automatic Detection**: Scans for collections with TTL configuration
- **Time-based Cleanup**: Deletes files older than configured TTL
- **Master API Integration**: Uses `CollectionDelete` API with time range parameters
- **Configurable**: Supports TTL settings, concurrent limits, and scheduling
- **Monitoring**: Provides progress tracking and detailed logging

## Architecture

### Components

1. **CollectionCleanupTask** (`cleanup_task.go`)
   - Main task implementation
   - Executes cleanup logic using master API
   - Handles progress tracking and error handling

2. **Config** (`config.go`)
   - Configuration management
   - TTL settings, concurrent tasks, scan intervals
   - Extends `base.BaseConfig`

3. **Detection** (`detection.go`)
   - Detects collections needing cleanup
   - Reads TTL configuration from filer settings
   - Creates task definitions for each collection

4. **Scheduling** (`scheduling.go`)
   - Controls when cleanup tasks can run
   - Enforces concurrent task limits
   - Validates worker capabilities

5. **Factory** (`register.go`)
   - Task creation and registration
   - Auto-registers with worker system
   - Implements `TaskFactory` interface

6. **UI Provider** (`ui_provider.go`)
   - Admin interface configuration
   - Schema validation and defaults
   - Configuration persistence

## Configuration

```yaml
collection_cleanup:
  enabled: false                    # Disabled by default
  scan_interval_seconds: 86400      # 24 hours
  max_concurrent: 1                 # Only one cleanup at a time
  default_ttl_days: 30              # Default 30 days TTL
```

## Workflow

1. **Detection Phase**
   - Worker scans for collections with TTL configuration
   - Reads TTL settings from filer configuration
   - Creates cleanup tasks for eligible collections

2. **Scheduling Phase**
   - Checks if cleanup is enabled
   - Validates concurrent task limits
   - Ensures worker has required capabilities

3. **Execution Phase**
   - Calculates time range based on TTL
   - Calls master `CollectionDelete` API with time range
   - Monitors progress and logs results

## TTL Configuration

Collections can have TTL configured in filer configuration:

```yaml
# Example filer configuration
path_conf:
  - collection: "temp_files"
    location_prefix: "/temp/"
    ttl: "7d"                    # 7 days
  - collection: "backup_data"
    location_prefix: "/backup/"
    ttl: "90d"                   # 90 days
```

## Integration

### Auto-Registration
The task is automatically registered when the worker starts:

```go
import _ "github.com/seaweedfs/seaweedfs/weed/worker/tasks/collection_cleanup"
```

### Task Type
- **Type**: `TaskTypeCollectionCleanup`
- **String**: `"collection_cleanup"`
- **Category**: Maintenance

## API Usage

The task uses the master `CollectionDelete` API:

```protobuf
CollectionDeleteRequest {
    string name = 1;
    uint64 from_time = 2;
    uint64 to_time = 3;
}
```

## Logging

The task provides detailed logging at different verbosity levels:

- `V(1)`: Task registration and major events
- `V(3)`: Detection results and execution details
- `V(4)`: Scheduling decisions and worker assignments
- `Error`: Failure conditions and error details

## Future Enhancements

1. **Real TTL Integration**: Connect to actual filer configuration
2. **Advanced Scheduling**: Support cron expressions
3. **Dry Run Mode**: Preview what would be deleted
4. **Metrics**: Prometheous metrics for cleanup operations
5. **Selective Cleanup**: Exclude specific files or patterns

## Usage

1. Enable the task in worker configuration
2. Configure TTL settings in filer configuration
3. Monitor logs for cleanup operations
4. Use admin UI to view task status and configuration

package dash

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/master_pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/volume_server_pb"
	"github.com/seaweedfs/seaweedfs/weed/storage/erasure_coding"
)

// GetClusterVolumes retrieves cluster volumes data with pagination, sorting, and filtering
func (s *AdminServer) GetClusterVolumes(page int, pageSize int, sortBy string, sortOrder string, collection string) (*ClusterVolumesData, error) {
	// Set defaults
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 100
	}
	if sortBy == "" {
		sortBy = "id"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}
	var volumes []VolumeWithTopology
	var totalSize int64
	var cachedTopologyInfo *master_pb.TopologyInfo

	// Get detailed volume information via gRPC
	err := s.WithMasterClient(func(client master_pb.SeaweedClient) error {
		resp, err := client.VolumeList(context.Background(), &master_pb.VolumeListRequest{})
		if err != nil {
			return err
		}

		// Cache the topology info for reuse
		cachedTopologyInfo = resp.TopologyInfo

		if resp.TopologyInfo != nil {
			for _, dc := range resp.TopologyInfo.DataCenterInfos {
				for _, rack := range dc.RackInfos {
					for _, node := range rack.DataNodeInfos {
						for _, diskInfo := range node.DiskInfos {
							// Process regular volumes
							for _, volInfo := range diskInfo.VolumeInfos {
								volume := VolumeWithTopology{
									VolumeInformationMessage: volInfo,
									Server:                   node.Id,
									DataCenter:               dc.Id,
									Rack:                     rack.Id,
								}
								volumes = append(volumes, volume)
								totalSize += int64(volInfo.Size)
							}

							// Process EC shards in the same loop
							for _, ecShardInfo := range diskInfo.EcShardInfos {
								// Add all shard sizes for this EC volume
								for _, shardSize := range ecShardInfo.ShardSizes {
									totalSize += shardSize
								}
							}
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Filter by collection if specified
	if collection != "" {
		var filteredVolumes []VolumeWithTopology
		var filteredTotalSize int64
		var filteredEcTotalSize int64

		for _, volume := range volumes {
			if matchesCollection(volume.Collection, collection) {
				filteredVolumes = append(filteredVolumes, volume)
				filteredTotalSize += int64(volume.Size)
			}
		}

		// Filter EC shard sizes by collection using already processed data
		// This reuses the topology traversal done above (lines 43-71) to avoid a second pass
		if cachedTopologyInfo != nil {
			for _, dc := range cachedTopologyInfo.DataCenterInfos {
				for _, rack := range dc.RackInfos {
					for _, node := range rack.DataNodeInfos {
						for _, diskInfo := range node.DiskInfos {
							for _, ecShardInfo := range diskInfo.EcShardInfos {
								if matchesCollection(ecShardInfo.Collection, collection) {
									// Add all shard sizes for this EC volume
									for _, shardSize := range ecShardInfo.ShardSizes {
										filteredEcTotalSize += shardSize
									}
								}
							}
						}
					}
				}
			}
		}

		volumes = filteredVolumes
		totalSize = filteredTotalSize + filteredEcTotalSize
	}

	// Calculate unique data center, rack, disk type, collection, and version counts from filtered volumes
	dataCenterMap := make(map[string]bool)
	rackMap := make(map[string]bool)
	diskTypeMap := make(map[string]bool)
	collectionMap := make(map[string]bool)
	versionMap := make(map[string]bool)
	for _, volume := range volumes {
		if volume.DataCenter != "" {
			dataCenterMap[volume.DataCenter] = true
		}
		if volume.Rack != "" {
			rackMap[volume.Rack] = true
		}
		diskType := volume.DiskType
		if diskType == "" {
			diskType = "hdd" // Default to hdd if not specified
		}
		diskTypeMap[diskType] = true

		// Handle collection for display purposes
		collectionName := volume.Collection
		if collectionName == "" {
			collectionName = "default"
		}
		collectionMap[collectionName] = true

		versionMap[fmt.Sprintf("%d", volume.Version)] = true
	}
	dataCenterCount := len(dataCenterMap)
	rackCount := len(rackMap)
	diskTypeCount := len(diskTypeMap)
	collectionCount := len(collectionMap)
	versionCount := len(versionMap)

	// Sort volumes
	s.sortVolumes(volumes, sortBy, sortOrder)

	// Get volume size limit from master
	var volumeSizeLimit uint64
	err = s.WithMasterClient(func(client master_pb.SeaweedClient) error {
		resp, err := client.GetMasterConfiguration(context.Background(), &master_pb.GetMasterConfigurationRequest{})
		if err != nil {
			return err
		}
		volumeSizeLimit = uint64(resp.VolumeSizeLimitMB) * 1024 * 1024 // Convert MB to bytes
		return nil
	})
	if err != nil {
		// If we can't get the limit, set a default
		volumeSizeLimit = 30 * 1024 * 1024 * 1024 // 30GB default
	}

	// Calculate pagination
	totalVolumes := len(volumes)
	totalPages := (totalVolumes + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	// Apply pagination
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize
	if startIndex >= totalVolumes {
		volumes = []VolumeWithTopology{}
	} else {
		if endIndex > totalVolumes {
			endIndex = totalVolumes
		}
		volumes = volumes[startIndex:endIndex]
	}

	// Determine conditional display flags and extract single values
	showDataCenterColumn := dataCenterCount > 1
	showRackColumn := rackCount > 1
	showDiskTypeColumn := diskTypeCount > 1
	showCollectionColumn := collectionCount > 1 && collection == "" // Hide column when filtering by collection
	showVersionColumn := versionCount > 1

	var singleDataCenter, singleRack, singleDiskType, singleCollection, singleVersion string
	var allVersions, allDiskTypes []string

	if dataCenterCount == 1 {
		for dc := range dataCenterMap {
			singleDataCenter = dc
			break
		}
	}
	if rackCount == 1 {
		for rack := range rackMap {
			singleRack = rack
			break
		}
	}
	if diskTypeCount == 1 {
		for diskType := range diskTypeMap {
			singleDiskType = diskType
			break
		}
	} else {
		// Collect all disk types and sort them
		for diskType := range diskTypeMap {
			allDiskTypes = append(allDiskTypes, diskType)
		}
		sort.Strings(allDiskTypes)
	}
	if collectionCount == 1 {
		for collection := range collectionMap {
			singleCollection = collection
			break
		}
	}
	if versionCount == 1 {
		for version := range versionMap {
			singleVersion = "v" + version
			break
		}
	} else {
		// Collect all versions and sort them
		for version := range versionMap {
			allVersions = append(allVersions, "v"+version)
		}
		sort.Strings(allVersions)
	}

	return &ClusterVolumesData{
		Volumes:              volumes,
		TotalVolumes:         totalVolumes,
		TotalSize:            totalSize,
		VolumeSizeLimit:      volumeSizeLimit,
		LastUpdated:          time.Now(),
		CurrentPage:          page,
		TotalPages:           totalPages,
		PageSize:             pageSize,
		SortBy:               sortBy,
		SortOrder:            sortOrder,
		DataCenterCount:      dataCenterCount,
		RackCount:            rackCount,
		DiskTypeCount:        diskTypeCount,
		CollectionCount:      collectionCount,
		VersionCount:         versionCount,
		ShowDataCenterColumn: showDataCenterColumn,
		ShowRackColumn:       showRackColumn,
		ShowDiskTypeColumn:   showDiskTypeColumn,
		ShowCollectionColumn: showCollectionColumn,
		ShowVersionColumn:    showVersionColumn,
		SingleDataCenter:     singleDataCenter,
		SingleRack:           singleRack,
		SingleDiskType:       singleDiskType,
		SingleCollection:     singleCollection,
		SingleVersion:        singleVersion,
		AllVersions:          allVersions,
		AllDiskTypes:         allDiskTypes,
		FilterCollection:     collection,
	}, nil
}

// sortVolumes sorts the volumes slice based on the specified field and order
func (s *AdminServer) sortVolumes(volumes []VolumeWithTopology, sortBy string, sortOrder string) {
	sort.Slice(volumes, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "id":
			less = volumes[i].Id < volumes[j].Id
		case "server":
			less = volumes[i].Server < volumes[j].Server
		case "datacenter":
			less = volumes[i].DataCenter < volumes[j].DataCenter
		case "rack":
			less = volumes[i].Rack < volumes[j].Rack
		case "collection":
			less = volumes[i].Collection < volumes[j].Collection
		case "size":
			less = volumes[i].Size < volumes[j].Size
		case "filecount":
			less = volumes[i].FileCount < volumes[j].FileCount
		case "replication":
			less = volumes[i].ReplicaPlacement < volumes[j].ReplicaPlacement
		case "disktype":
			less = volumes[i].DiskType < volumes[j].DiskType
		case "version":
			less = volumes[i].Version < volumes[j].Version
		//QUYNGUYEN add
		case "ttl":
			less = volumes[i].Ttl < volumes[j].Ttl
		default:
			less = volumes[i].Id < volumes[j].Id
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

// GetVolumeDetails retrieves detailed information about a specific volume
func (s *AdminServer) GetVolumeDetails(volumeID int, server string) (*VolumeDetailsData, error) {
	var primaryVolume VolumeWithTopology
	var replicas []VolumeWithTopology
	var volumeSizeLimit uint64
	var found bool

	// Find the volume and all its replicas in the cluster
	err := s.WithMasterClient(func(client master_pb.SeaweedClient) error {
		resp, err := client.VolumeList(context.Background(), &master_pb.VolumeListRequest{})
		if err != nil {
			return err
		}

		if resp.TopologyInfo != nil {
			for _, dc := range resp.TopologyInfo.DataCenterInfos {
				for _, rack := range dc.RackInfos {
					for _, node := range rack.DataNodeInfos {
						for _, diskInfo := range node.DiskInfos {
							for _, volInfo := range diskInfo.VolumeInfos {
								if int(volInfo.Id) == volumeID {
									diskType := volInfo.DiskType
									if diskType == "" {
										diskType = "hdd"
									}

									volume := VolumeWithTopology{
										VolumeInformationMessage: volInfo,
										Server:                   node.Id,
										DataCenter:               dc.Id,
										Rack:                     rack.Id,
									}

									// If this is the requested server, it's the primary volume
									if node.Id == server {
										primaryVolume = volume
										found = true
									} else {
										// This is a replica on another server
										replicas = append(replicas, volume)
									}
								}
							}
						}
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("volume %d not found on server %s", volumeID, server)
	}

	// Get volume size limit from master
	err = s.WithMasterClient(func(client master_pb.SeaweedClient) error {
		resp, err := client.GetMasterConfiguration(context.Background(), &master_pb.GetMasterConfigurationRequest{})
		if err != nil {
			return err
		}
		volumeSizeLimit = uint64(resp.VolumeSizeLimitMB) * 1024 * 1024 // Convert MB to bytes
		return nil
	})

	if err != nil {
		// If we can't get the limit, set a default
		volumeSizeLimit = 30 * 1024 * 1024 * 1024 // 30GB default
	}

	return &VolumeDetailsData{
		Volume:           primaryVolume,
		Replicas:         replicas,
		VolumeSizeLimit:  volumeSizeLimit,
		ReplicationCount: len(replicas) + 1, // Include the primary volume
		LastUpdated:      time.Now(),
	}, nil
}

// VacuumVolume performs a vacuum operation on a specific volume
func (s *AdminServer) VacuumVolume(volumeID int, server string) error {
	// Validate volumeID range before converting to uint32
	if volumeID < 0 || uint64(volumeID) > math.MaxUint32 {
		return fmt.Errorf("volume ID out of range: %d", volumeID)
	}
	return s.WithMasterClient(func(client master_pb.SeaweedClient) error {
		_, err := client.VacuumVolume(context.Background(), &master_pb.VacuumVolumeRequest{
			// lgtm[go/incorrect-integer-conversion]
			// Safe conversion: volumeID has been validated to be in range [0, 0xFFFFFFFF] above
			VolumeId:         uint32(volumeID),
			GarbageThreshold: 0.0001, // A very low threshold to ensure all garbage is collected
			Collection:       "",     // Empty for all collections
		})
		return err
	})
}

// QUYNGUYEN: Delete volume files from filer servers
func (s *AdminServer) DeleteVolumeFiles(volumeID int, server string, collection string) (int, error) {
	// Validate volumeID range before converting to uint32
	if volumeID < 0 || uint64(volumeID) > math.MaxUint32 {
		return 0, fmt.Errorf("volume ID out of range: %d", volumeID)
	}

	glog.V(2).Infof("QUYNGUYEN: AdminServer.DeleteVolumeFiles - volumeID: %d, server: %s, collection: %s", volumeID, server, collection)

	// Step 1: Delete volume from volume server directly
	err := s.DeleteVolume(volumeID, server, false) // Delete even if not empty
	if err != nil {
		glog.V(2).Infof("QUYNGUYEN: Failed to delete volume: %v", err)
		return 0, fmt.Errorf("failed to delete volume: %v", err)
	}

	glog.V(2).Infof("QUYNGUYEN: Successfully deleted volume %d from server %s", volumeID, server)

	// Step 2: Send volume ID to filer servers to delete entries directly
	var deletedEntries int

	// Get all filer servers using existing method
	filerServers := s.GetAllFilers()
	if len(filerServers) == 0 {
		glog.V(2).Infof("QUYNGUYEN: No filer servers found")
		// Volume was deleted but no filers to clean up - still consider success
		return -1, nil
	}

	// Call each filer server directly to delete entries
	volumeIds := []uint32{uint32(volumeID)}
	for _, filerServer := range filerServers {
		err = pb.WithFilerClient(false, 0, pb.ServerAddress(filerServer), s.grpcDialOption, func(client filer_pb.SeaweedFilerClient) error {
			_, err := client.DeleteCollection(context.Background(), &filer_pb.DeleteCollectionRequest{
				Collection: collection,
				VolumeIds:  volumeIds,
			})
			return err
		})

		if err != nil {
			glog.V(2).Infof("QUYNGUYEN: Failed to delete entries from filer %s: %v", filerServer, err)
			// Continue with other filers even if one fails
		} else {
			glog.V(2).Infof("QUYNGUYEN: Successfully sent delete request to filer %s", filerServer)
			deletedEntries = -1 // At least one filer succeeded
		}
	}

	if err != nil {
		glog.V(2).Infof("QUYNGUYEN: Failed to delete volume files from filers: %v", err)
		// Volume was deleted but filer cleanup failed - still consider success
		glog.V(2).Infof("QUYNGUYEN: Volume deleted but filer cleanup incomplete")
	}

	glog.V(2).Infof("QUYNGUYEN: DeleteVolumeFiles completed - volume: %d, filer cleanup: %v", volumeID, err == nil)

	return deletedEntries, nil
}

// QUYNGUYEN: Cleanup collection (delete expired files)
func (s *AdminServer) CleanupCollection(collection string) (int, error) {
	glog.V(2).Infof("QUYNGUYEN: AdminServer.CleanupCollection - collection: %s", collection)

	// Call master server to delete expired files
	var deletedEntries int
	err := s.WithMasterClient(func(client master_pb.SeaweedClient) error {
		_, err := client.CollectionCleanup(context.Background(), &master_pb.CollectionCleanupRequest{
			Name:     collection,
		})
		if err != nil {
			return err
		}

		// CollectionDelete doesn't return deleted count, so we return -1
		deletedEntries = -1
		return nil
	})

	if err != nil {
		glog.V(2).Infof("QUYNGUYEN: Failed to cleanup collection via master: %v", err)
		return 0, err
	}

	glog.V(2).Infof("QUYNGUYEN: CleanupCollection completed - collection: %s, fromTime: %d, toTime: %d", collection)

	return deletedEntries, nil
}

//QUYNGUYEN: Cleanup collections matching regex pattern
// CleanupCollectionsByPattern cleans up all collections whose names match the given regex pattern
func (s *AdminServer) CleanupCollectionsByPattern(pattern string) (map[string]int, error) {
	glog.V(2).Infof("QUYNGUYEN: AdminServer.CleanupCollectionsByPattern - pattern: %s", pattern)

	// Compile regex pattern
	regex, err := regexp.Compile(pattern)
	if err != nil {
		glog.V(2).Infof("QUYNGUYEN: Invalid regex pattern: %v", err)
		return nil, fmt.Errorf("invalid regex pattern: %v", err)
	}

	// Get all collections from master
	var allCollections []string
	err = s.WithMasterClient(func(client master_pb.SeaweedClient) error {
		resp, err := client.CollectionList(context.Background(), &master_pb.CollectionListRequest{
			IncludeNormalVolumes: true,
			IncludeEcVolumes:     true,
		})
		if err != nil {
			return err
		}

		for _, c := range resp.Collections {
			allCollections = append(allCollections, c.Name)
		}

		return nil
	})

	if err != nil {
		glog.V(2).Infof("QUYNGUYEN: Failed to get collections from master: %v", err)
		return nil, fmt.Errorf("failed to get collections: %v", err)
	}

	// Filter collections by regex pattern
	var matchedCollections []string
	for _, collection := range allCollections {
		if regex.MatchString(collection) {
			matchedCollections = append(matchedCollections, collection)
		}
	}

	glog.V(2).Infof("QUYNGUYEN: Found %d collections matching pattern '%s': %v", len(matchedCollections), pattern, matchedCollections)

	// Cleanup each matched collection
	results := make(map[string]int)
	for _, collection := range matchedCollections {
		deletedCount, err := s.CleanupCollection(collection)
		if err != nil {
			glog.V(2).Infof("QUYNGUYEN: Failed to cleanup collection %s: %v", collection, err)
			results[collection] = -2 // Error marker
		} else {
			results[collection] = deletedCount
			glog.V(2).Infof("QUYNGUYEN: Successfully cleaned up collection %s: %d entries", collection, deletedCount)
		}
	}

	glog.V(2).Infof("QUYNGUYEN: CleanupCollectionsByPattern completed - pattern: %s, matched: %d, results: %v", pattern, len(matchedCollections), results)

	return results, nil
}
//QUYNGUYEN end

// GetClusterVolumeServers retrieves cluster volume servers data including EC shard information
func (s *AdminServer) GetClusterVolumeServers() (*ClusterVolumeServersData, error) {
	var volumeServerMap map[string]*VolumeServer

	// Make only ONE VolumeList call and use it for both topology building AND EC shard processing
	err := s.WithMasterClient(func(client master_pb.SeaweedClient) error {
		resp, err := client.VolumeList(context.Background(), &master_pb.VolumeListRequest{})
		if err != nil {
			return err
		}

		// Get volume size limit from response, default to 30GB if not set
		volumeSizeLimitMB := resp.VolumeSizeLimitMb
		if volumeSizeLimitMB == 0 {
			volumeSizeLimitMB = 30000 // default to 30000MB (30GB)
		}

		// Build basic topology from the VolumeList response (replaces GetClusterTopology call)
		volumeServerMap = make(map[string]*VolumeServer)

		if resp.TopologyInfo != nil {
			// Process topology to build basic volume server info (similar to cluster_topology.go logic)
			for _, dc := range resp.TopologyInfo.DataCenterInfos {
				for _, rack := range dc.RackInfos {
					for _, node := range rack.DataNodeInfos {
						// Initialize volume server if not exists
						if volumeServerMap[node.Id] == nil {
							volumeServerMap[node.Id] = &VolumeServer{
								Address:        node.Id,
								DataCenter:     dc.Id,
								Rack:           rack.Id,
								Volumes:        0,
								DiskUsage:      0,
								DiskCapacity:   0,
								EcVolumes:      0,
								EcShards:       0,
								EcShardDetails: []VolumeServerEcInfo{},
							}
						}
						vs := volumeServerMap[node.Id]

						// Process EC shard information for this server at volume server level (not per-disk)
						ecVolumeMap := make(map[uint32]*VolumeServerEcInfo)
						// Temporary map to accumulate shard info across disks
						ecShardAccumulator := make(map[uint32][]*master_pb.VolumeEcShardInformationMessage)

						// Process disk information
						for _, diskInfo := range node.DiskInfos {
							vs.DiskCapacity += int64(diskInfo.MaxVolumeCount) * int64(volumeSizeLimitMB) * 1024 * 1024 // Use actual volume size limit

							// Count regular volumes and calculate disk usage
							for _, volInfo := range diskInfo.VolumeInfos {
								vs.Volumes++
								vs.DiskUsage += int64(volInfo.Size)
							}

							// Accumulate EC shard information across all disks for this volume server
							for _, ecShardInfo := range diskInfo.EcShardInfos {
								volumeId := ecShardInfo.Id
								ecShardAccumulator[volumeId] = append(ecShardAccumulator[volumeId], ecShardInfo)
							}
						}

						// Process accumulated EC shard information per volume
						for volumeId, ecShardInfos := range ecShardAccumulator {
							if len(ecShardInfos) == 0 {
								continue
							}

							// Initialize EC volume info
							ecInfo := &VolumeServerEcInfo{
								VolumeID:     volumeId,
								Collection:   ecShardInfos[0].Collection,
								ShardCount:   0,
								EcIndexBits:  0,
								ShardNumbers: []int{},
								ShardSizes:   make(map[int]int64),
								TotalSize:    0,
							}

							// Merge EcIndexBits from all disks and collect shard sizes
							allShardSizes := make(map[erasure_coding.ShardId]int64)
							for _, ecShardInfo := range ecShardInfos {
								ecInfo.EcIndexBits |= ecShardInfo.EcIndexBits

								// Collect shard sizes from this disk
								shardBits := erasure_coding.ShardBits(ecShardInfo.EcIndexBits)
								shardBits.EachSetIndex(func(shardId erasure_coding.ShardId) {
									if size, found := erasure_coding.GetShardSize(ecShardInfo, shardId); found {
										allShardSizes[shardId] = size
									}
								})
							}

							// Process final merged shard information
							finalShardBits := erasure_coding.ShardBits(ecInfo.EcIndexBits)
							finalShardBits.EachSetIndex(func(shardId erasure_coding.ShardId) {
								ecInfo.ShardCount++
								ecInfo.ShardNumbers = append(ecInfo.ShardNumbers, int(shardId))
								vs.EcShards++

								// Add shard size if available
								if shardSize, exists := allShardSizes[shardId]; exists {
									ecInfo.ShardSizes[int(shardId)] = shardSize
									ecInfo.TotalSize += shardSize
									vs.DiskUsage += shardSize // Add EC shard size to total disk usage
								}
							})

							ecVolumeMap[volumeId] = ecInfo
						}

						// Convert EC volume map to slice and update volume server (after processing all disks)
						for _, ecInfo := range ecVolumeMap {
							vs.EcShardDetails = append(vs.EcShardDetails, *ecInfo)
							vs.EcVolumes++
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map back to slice
	var volumeServers []VolumeServer
	for _, vs := range volumeServerMap {
		volumeServers = append(volumeServers, *vs)
	}

	// Sort volume servers by address for consistent ordering on page refresh
	sort.Slice(volumeServers, func(i, j int) bool {
		return volumeServers[i].GetDisplayAddress() < volumeServers[j].GetDisplayAddress()
	})

	var totalCapacity int64
	var totalVolumes int
	for _, vs := range volumeServers {
		totalCapacity += vs.DiskCapacity
		totalVolumes += vs.Volumes
	}

	return &ClusterVolumeServersData{
		VolumeServers:      volumeServers,
		TotalVolumeServers: len(volumeServers),
		TotalVolumes:       totalVolumes,
		TotalCapacity:      totalCapacity,
		LastUpdated:        time.Now(),
	}, nil
}

// DeleteVolume deletes a volume by ID from a specific server
// QUYNGUYEN
func (s *AdminServer) DeleteVolume(volumeID int, server string, onlyEmpty bool) error {
	// Validate volumeID range before converting to uint32
	if volumeID < 0 || uint64(volumeID) > math.MaxUint32 {
		return fmt.Errorf("volume ID out of range: %d", volumeID)
	}

	return s.WithVolumeServerClient(pb.ServerAddress(server), func(client volume_server_pb.VolumeServerClient) error {
		_, err := client.VolumeDelete(context.Background(), &volume_server_pb.VolumeDeleteRequest{
			// lgtm[go/incorrect-integer-conversion]
			// Safe conversion: volumeID has been validated to be in range [0, 0xFFFFFFFF] above
			VolumeId:  uint32(volumeID),
			OnlyEmpty: onlyEmpty,
		})
		return err
	})
}

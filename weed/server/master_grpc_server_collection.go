package weed_server

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/seaweedfs/raft"

	"github.com/seaweedfs/seaweedfs/weed/cluster"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/operation"
	"github.com/seaweedfs/seaweedfs/weed/pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/master_pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/volume_server_pb"
)

func (ms *MasterServer) CollectionList(ctx context.Context, req *master_pb.CollectionListRequest) (*master_pb.CollectionListResponse, error) {

	if !ms.Topo.IsLeader() {
		return nil, raft.NotLeaderError
	}

	resp := &master_pb.CollectionListResponse{}
	collections := ms.Topo.ListCollections(req.IncludeNormalVolumes, req.IncludeEcVolumes)
	for _, c := range collections {
		resp.Collections = append(resp.Collections, &master_pb.Collection{
			Name: c,
		})
	}

	return resp, nil
}

func (ms *MasterServer) CollectionDelete(ctx context.Context, req *master_pb.CollectionDeleteRequest) (*master_pb.CollectionDeleteResponse, error) {

	if !ms.Topo.IsLeader() {
		return nil, raft.NotLeaderError
	}
	log.Println("QUYNGUYEN: CollectionDelete master receive request: ", req.FromTime, req.ToTime, req.Name)
	resp := &master_pb.CollectionDeleteResponse{}

	// QUYNGUYEN: Phase 1 - Delete metadata trên TẤT CẢ filers
	err := ms.deleteCollectionOnAllFilers(ctx, req.Name, req.FromTime, req.ToTime)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection metadata on filers: %v", err)
	}

	// QUYNGUYEN: Phase 2 - Delete volume data trên volume servers
	err = ms.doDeleteNormalCollection(req.Name, req.FromTime, req.ToTime)
	if err != nil {
		return nil, fmt.Errorf("failed to delete normal collection: %v", err)
	}

	err = ms.doDeleteEcCollection(req.Name, req.FromTime, req.ToTime)
	if err != nil {
		return nil, fmt.Errorf("failed to delete EC collection: %v", err)
	}

	return resp, nil
}

// QUYNGUYEN: Delete collection metadata trên tất cả filers
func (ms *MasterServer) deleteCollectionOnAllFilers(ctx context.Context, collectionName string, fromTime, toTime uint64) error {
	// Lấy danh sách tất cả filers
	filerNodes, err := ms.getFilerNodes()
	if err != nil {
		return fmt.Errorf("failed to get filer nodes: %v", err)
	}

	glog.V(2).Infof("QUYNGUYEN: Deleting collection %s on %d filers", collectionName, len(filerNodes))

	// Parallel delete trên tất cả filers
	var wg sync.WaitGroup
	errChan := make(chan error, len(filerNodes))

	for _, node := range filerNodes {
		wg.Add(1)
		go func(filerAddress string) {
			defer wg.Done()
			err := ms.deleteCollectionOnFiler(ctx, filerAddress, collectionName, fromTime, toTime, nil)
			if err != nil {
				errChan <- fmt.Errorf("failed to delete on filer %s: %v", filerAddress, err)
			}
		}(node.Address)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("filer deletion errors: %s", strings.Join(errors, "; "))
	}

	glog.V(2).Infof("QUYNGUYEN: Successfully deleted collection %s on all filers", collectionName)
	return nil
}

// QUYNGUYEN: Lấy danh sách tất cả filers
func (ms *MasterServer) getFilerNodes() ([]*master_pb.ListClusterNodesResponse_ClusterNode, error) {
	var filerNodes []*master_pb.ListClusterNodesResponse_ClusterNode

	// Sử dụng Cluster.ListClusterNode để lấy filers
	clusterNodes := ms.Cluster.ListClusterNode("", cluster.FilerType)
	for _, node := range clusterNodes {
		filerNodes = append(filerNodes, &master_pb.ListClusterNodesResponse_ClusterNode{
			Address:     string(node.Address),
			Version:     node.Version,
			CreatedAtNs: node.CreatedTs.UnixNano(),
			DataCenter:  string(node.DataCenter),
			Rack:        string(node.Rack),
		})
	}

	return filerNodes, nil
}

// QUYNGUYEN: Delete collection trên một filer cụ thể
func (ms *MasterServer) deleteCollectionOnFiler(ctx context.Context, filerAddress, collectionName string, fromTime, toTime uint64, volumeIds []uint32) error {
	glog.V(3).Infof("QUYNGUYEN: Deleting collection %s on filer %s with %d volume IDs", collectionName, filerAddress, len(volumeIds))

	return pb.WithFilerClient(false, 0, pb.ServerAddress(filerAddress), ms.grpcDialOption, func(client filer_pb.SeaweedFilerClient) error {
		_, err := client.DeleteCollection(ctx, &filer_pb.DeleteCollectionRequest{
			Collection: collectionName,
			FromTime:   fromTime,
			ToTime:     toTime,
			VolumeIds:  volumeIds,
		})
		return err
	})
}

// QUYNGUYEN: Common function to collect deleted volume IDs from volume servers
func (ms *MasterServer) collectDeletedVolumeIdsFromServers(servers []string, collectionName string, fromTime, toTime uint64) ([]uint32, error) {
	allDeletedVolumeIds := make([]uint32, 0)

	for _, server := range servers {
		err := operation.WithVolumeServerClient(false, pb.ServerAddress(server), ms.grpcDialOption, func(client volume_server_pb.VolumeServerClient) error {
			resp, deleteErr := client.DeleteCollection(context.Background(), &volume_server_pb.DeleteCollectionRequest{
				Collection: collectionName,
				FromTime:   fromTime,
				ToTime:     toTime,
			})
			if deleteErr != nil {
				return deleteErr
			}
			// Collect volume IDs from response
			if resp != nil && len(resp.VolumeIds) > 0 {
				allDeletedVolumeIds = append(allDeletedVolumeIds, resp.VolumeIds...)
				glog.V(2).Infof("QUYNGUYEN: Collected %d volume IDs from server %s: %v", len(resp.VolumeIds), server, resp.VolumeIds)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return allDeletedVolumeIds, nil
}

// QUYNGUYEN: Common function to send deleted volume IDs to all filer servers
func (ms *MasterServer) sendDeletedVolumeIdsToFilers(ctx context.Context, collectionName string, fromTime, toTime uint64, volumeIds []uint32) error {
	if len(volumeIds) == 0 {
		return nil
	}

	glog.V(2).Infof("QUYNGUYEN: Sending %d deleted volume IDs to filers: %v", len(volumeIds), volumeIds)

	filerNodes := ms.Cluster.ListClusterNode("", cluster.FilerType)
	for _, node := range filerNodes {
		err := pb.WithFilerClient(false, 0, node.Address, ms.grpcDialOption, func(client filer_pb.SeaweedFilerClient) error {
			_, deleteErr := client.DeleteCollection(ctx, &filer_pb.DeleteCollectionRequest{
				Collection: collectionName,
				FromTime:   fromTime,
				ToTime:     toTime,
				VolumeIds:  volumeIds,
			})
			if deleteErr != nil {
				glog.V(2).Infof("QUYNGUYEN: Failed to delete entries from filer %s: %v", node.Address, deleteErr)
				return deleteErr
			}
			glog.V(2).Infof("QUYNGUYEN: Successfully deleted entries for %d volumes from filer %s", len(volumeIds), node.Address)
			return nil
		})
		if err != nil {
			glog.V(2).Infof("QUYNGUYEN: Error connecting to filer %s: %v", node.Address, err)
			// Continue with other filers even if one fails
		}
	}
	return nil
}

// QUYNGUYEN:add fromTime, toTime
func (ms *MasterServer) doDeleteNormalCollection(collectionName string, fromTime, toTime uint64) error {
	collection, ok := ms.Topo.FindCollection(collectionName)
	if !ok {
		return nil
	}

	// Get list of normal volume servers
	serverAddresses := make([]string, 0)
	for _, server := range collection.ListVolumeServers() {
		serverAddresses = append(serverAddresses, string(server.ServerAddress()))
	}

	// Collect deleted volume IDs from all normal volume servers
	allDeletedVolumeIds, err := ms.collectDeletedVolumeIdsFromServers(serverAddresses, collectionName, fromTime, toTime)
	if err != nil {
		return err
	}

	// Send deleted volume IDs to all filer servers using common function
	err = ms.sendDeletedVolumeIdsToFilers(context.Background(), collectionName, fromTime, toTime, allDeletedVolumeIds)
	if err != nil {
		return err
	}

	//QUYNGUYEN: delete entry in ec collection
	if fromTime != 0 && toTime != 0 {
		return nil
	}
	ms.Topo.DeleteCollection(collectionName)

	return nil
}

//QUYNGUYEN:add fromTime, toTime

func (ms *MasterServer) doDeleteEcCollection(collectionName string, fromTime, toTime uint64) error {
	//QUYNGUYEN
	listOfEcServers := ms.Topo.ListEcServersByCollection(collectionName)
	
	// Convert []pb.ServerAddress to []string
	serverAddresses := make([]string, len(listOfEcServers))
	for i, server := range listOfEcServers {
		serverAddresses[i] = string(server)
	}

	// Collect deleted volume IDs from all EC volume servers using common function
	allDeletedVolumeIds, err := ms.collectDeletedVolumeIdsFromServers(serverAddresses, collectionName, fromTime, toTime)
	if err != nil {
		return err
	}

	// Send deleted volume IDs to all filer servers using common function
	err = ms.sendDeletedVolumeIdsToFilers(context.Background(), collectionName, fromTime, toTime, allDeletedVolumeIds)
	if err != nil {
		return err
	}
	//QUYNGUYEN: delete entry in ec collection
	if fromTime != 0 && toTime != 0 {
		return nil
	}
	ms.Topo.DeleteEcCollection(collectionName)

	return nil
}

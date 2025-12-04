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
			err := ms.deleteCollectionOnFiler(ctx, filerAddress, collectionName, fromTime, toTime)
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
func (ms *MasterServer) deleteCollectionOnFiler(ctx context.Context, filerAddress, collectionName string, fromTime, toTime uint64) error {
	glog.V(3).Infof("QUYNGUYEN: Deleting collection %s on filer %s", collectionName, filerAddress)

	return pb.WithFilerClient(false, 0, pb.ServerAddress(filerAddress), ms.grpcDialOption, func(client filer_pb.SeaweedFilerClient) error {
		_, err := client.DeleteCollection(ctx, &filer_pb.DeleteCollectionRequest{
			Collection: collectionName,
			FromTime:   fromTime,
			ToTime:     toTime,
		})
		return err
	})
}

// QUYNGUYEN:add fromTime, toTime
func (ms *MasterServer) doDeleteNormalCollection(collectionName string, fromTime, toTime uint64) error {
	collection, ok := ms.Topo.FindCollection(collectionName)
	if !ok {
		return nil
	}

	for _, server := range collection.ListVolumeServers() {
		err := operation.WithVolumeServerClient(false, server.ServerAddress(), ms.grpcDialOption, func(client volume_server_pb.VolumeServerClient) error {
			_, deleteErr := client.DeleteCollection(context.Background(), &volume_server_pb.DeleteCollectionRequest{
				Collection: collectionName,
				FromTime:   fromTime,
				ToTime:     toTime,
			})
			return deleteErr
		})
		if err != nil {
			return err
		}
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

	listOfEcServers := ms.Topo.ListEcServersByCollection(collectionName)

	for _, server := range listOfEcServers {
		err := operation.WithVolumeServerClient(false, server, ms.grpcDialOption, func(client volume_server_pb.VolumeServerClient) error {
			_, deleteErr := client.DeleteCollection(context.Background(), &volume_server_pb.DeleteCollectionRequest{
				Collection: collectionName,
				FromTime:   fromTime,
				ToTime:     toTime,
			})
			return deleteErr
		})
		if err != nil {
			return err
		}
	}
	//QUYNGUYEN: delete entry in ec collection
	if fromTime != 0 && toTime != 0 {
		return nil
	}
	ms.Topo.DeleteEcCollection(collectionName)

	return nil
}

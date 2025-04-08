package filer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/cluster"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/master_pb"
	"github.com/seaweedfs/seaweedfs/weed/util"
)

const (
	MsgFailDelNonEmptyFolder = "fail to delete non-empty folder"
)

type OnChunksFunc func([]*filer_pb.FileChunk) error
type OnHardLinkIdsFunc func([]HardLinkId) error

func (f *Filer) DeleteEntryMetaAndData(ctx context.Context, p util.FullPath, isRecursive, ignoreRecursiveError, shouldDeleteChunks, isFromOtherCluster bool, signatures []int32, ifNotModifiedAfter int64) (err error) {
	if p == "/" {
		return nil
	}

	entry, findErr := f.FindEntry(ctx, p)
	if findErr != nil {
		return findErr
	}
	if ifNotModifiedAfter > 0 && entry.Attr.Mtime.Unix() > ifNotModifiedAfter {
		return nil
	}
	isDeleteCollection := f.isBucket(entry)
	if entry.IsDirectory() {
		// delete the folder children, not including the folder itself
		err = f.doBatchDeleteFolderMetaAndData(ctx, entry, isRecursive, ignoreRecursiveError, shouldDeleteChunks && !isDeleteCollection, isDeleteCollection, isFromOtherCluster, signatures, func(hardLinkIds []HardLinkId) error {
			// A case not handled:
			// what if the chunk is in a different collection?
			if shouldDeleteChunks {
				f.maybeDeleteHardLinks(hardLinkIds)
			}
			return nil
		})
		if err != nil {
			glog.V(2).Infof("delete directory %s: %v", p, err)
			return fmt.Errorf("delete directory %s: %v", p, err)
		}
	}

	// delete the file or folder
	err = f.doDeleteEntryMetaAndData(ctx, entry, shouldDeleteChunks, isFromOtherCluster, signatures)
	if err != nil {
		return fmt.Errorf("delete file %s: %v", p, err)
	}

	if shouldDeleteChunks && !isDeleteCollection {
		f.DeleteChunks(p, entry.GetChunks())
	}

	if isDeleteCollection {
		collectionName := entry.Name()
		f.DoDeleteCollection(collectionName)
	}

	return nil
}

func (f *Filer) doBatchDeleteFolderMetaAndData(ctx context.Context, entry *Entry, isRecursive, ignoreRecursiveError, shouldDeleteChunks, isDeletingBucket, isFromOtherCluster bool, signatures []int32, onHardLinkIdsFn OnHardLinkIdsFunc) (err error) {

	//collect all the chunks of this layer and delete them together at the end
	var chunksToDelete []*filer_pb.FileChunk
	lastFileName := ""
	includeLastFile := false
	if !isDeletingBucket || !f.Store.CanDropWholeBucket() {
		for {
			entries, _, err := f.ListDirectoryEntries(ctx, entry.FullPath, lastFileName, includeLastFile, PaginationSize, "", "", "")
			if err != nil {
				glog.Errorf("list folder %s: %v", entry.FullPath, err)
				return fmt.Errorf("list folder %s: %v", entry.FullPath, err)
			}
			if lastFileName == "" && !isRecursive && len(entries) > 0 {
				// only for first iteration in the loop
				glog.V(2).Infof("deleting a folder %s has children: %+v ...", entry.FullPath, entries[0].Name())
				return fmt.Errorf("%s: %s", MsgFailDelNonEmptyFolder, entry.FullPath)
			}

			for _, sub := range entries {
				lastFileName = sub.Name()
				if sub.IsDirectory() {
					subIsDeletingBucket := f.isBucket(sub)
					err = f.doBatchDeleteFolderMetaAndData(ctx, sub, isRecursive, ignoreRecursiveError, shouldDeleteChunks, subIsDeletingBucket, false, nil, onHardLinkIdsFn)
				} else {
					f.NotifyUpdateEvent(ctx, sub, nil, shouldDeleteChunks, isFromOtherCluster, nil)
					if len(sub.HardLinkId) != 0 {
						// hard link chunk data are deleted separately
						err = onHardLinkIdsFn([]HardLinkId{sub.HardLinkId})
					} else {
						if shouldDeleteChunks {
							chunksToDelete = append(chunksToDelete, sub.GetChunks()...)
						}
					}
				}
				if err != nil && !ignoreRecursiveError {
					return err
				}
			}

			if len(entries) < PaginationSize {
				break
			}
		}
	}

	glog.V(3).Infof("deleting directory %v delete chunks: %v", entry.FullPath, shouldDeleteChunks)

	if storeDeletionErr := f.Store.DeleteFolderChildren(ctx, entry.FullPath); storeDeletionErr != nil {
		return fmt.Errorf("filer store delete: %v", storeDeletionErr)
	}

	f.NotifyUpdateEvent(ctx, entry, nil, shouldDeleteChunks, isFromOtherCluster, signatures)
	f.DeleteChunks(entry.FullPath, chunksToDelete)
	return nil
}

func (f *Filer) doDeleteEntryMetaAndData(ctx context.Context, entry *Entry, shouldDeleteChunks bool, isFromOtherCluster bool, signatures []int32) (err error) {

	glog.V(3).Infof("deleting entry %v, delete chunks: %v", entry.FullPath, shouldDeleteChunks)

	if storeDeletionErr := f.Store.DeleteOneEntry(ctx, entry); storeDeletionErr != nil {
		return fmt.Errorf("filer store delete: %v", storeDeletionErr)
	}
	if !entry.IsDirectory() {
		f.NotifyUpdateEvent(ctx, entry, nil, shouldDeleteChunks, isFromOtherCluster, signatures)
	}

	return nil
}

func (f *Filer) DoDeleteCollection(collectionName string) (err error) {

	return f.MasterClient.WithClient(false, func(client master_pb.SeaweedClient) error {
		_, err := client.CollectionDelete(context.Background(), &master_pb.CollectionDeleteRequest{
			Name: collectionName,
		})
		if err != nil {
			glog.Infof("delete collection %s: %v", collectionName, err)
		}
		return err
	})
}
func isValidKeyInTime(key string, fromDate, toDate int) bool {
	keyParts := strings.Split(key, "_")

	if fromDate == 0 || toDate == 0 {
		return true
	}
	if len(keyParts) < 3 {
		return false
	}
	key = keyParts[2]
	keyDate, _ := strconv.Atoi(key)

	if keyDate > 0 && keyDate >= fromDate && keyDate <= toDate {
		return true
	}
	return false
}
func isEntryCreatedInTime(entry *Entry, fromDate, toDate uint64) bool {
	createdTime := uint64(entry.Attr.Crtime.Unix())
	// log.Println("QUYNGUYEN Get Created Time  ", createdTime, fromDate, toDate)
	if createdTime > 0 && createdTime >= fromDate && createdTime <= toDate {
		return true
	}
	return false
}
func timestamptoDate(timestamp int64) int {
	dateNum, err := strconv.Atoi(strings.ReplaceAll(time.Unix(timestamp, 0).Format("06-01-02-15-04-05"), "-", ""))
	if err != nil {
		return 0
	}
	return dateNum
}

func (f *Filer) doDeleteFilerEntryWithTime(ctx context.Context, fullPath util.FullPath, fromDate, toDate uint64) (err error) {
	// TODO dele record in current filer

	lastFileName := ""
	includeLastFile := false

	// glog.V(2).Infoln("QUYNGUYEN delete collection  ", fromDate, toDate, fullPath)
	for {
		entries, _, err := f.ListDirectoryEntries(ctx, fullPath, lastFileName, includeLastFile, PaginationSize, "", "", "")
		if err != nil {
			glog.Errorf("QUYNGUYEN list folder %s: %v", fullPath, err)
			return fmt.Errorf("QUYNGUYEN list folder %s: %v", fullPath, err)
		}
		entryCount := 0
		for _, entry := range entries {
			lastFileName = entry.Name()
			// glog.V(2).Infof("QUYNGUYEN deleting 1 %s %b %b %d %d", entry.FullPath, entry.IsDirectory(), fromDate, toDate)
			// if !isValidKeyInTime(fmt.Sprintf("%s", entry.FullPath), fromDate, toDate) {
			// 	continue
			// }
			if entry.IsDirectory() {
				f.doDeleteFilerEntryWithTime(ctx, entry.FullPath, fromDate, toDate)
			} else if isEntryCreatedInTime(entry, fromDate, toDate) {
				// if len(entry.HardLinkId) != 0 {
				// 	// hard link chunk data are deleted separately
				// 	f.maybeDeleteHardLinks([]HardLinkId{entry.HardLinkId})
				// glog.Errorf("QUYNGUYEN list isEntryCreatedInTime %s: %v", fullPath, err)
				// }
				entryCount++
				storeDeletionErr := f.Store.DeleteOneEntry(ctx, entry)
				glog.V(2).Infof("QUYNGUYEN delete collection DeleteOneEntry  %s: %v", entry.FullPath, storeDeletionErr)
			}
		}
		// glog.V(2).Infof("QUYNGUYEN deleting directory 1 %d ", entryCount)

		if len(entries) < PaginationSize {
			break
		}
	}

	// glog.V(2).Infof("QUYNGUYEN deleting directory %s %d ", fullPath)
	// if !isValidKeyInTime(string(fullPath), fromDate, toDate) {
	// 	return nil
	// }
	//Keep folder but delete children
	// if storeDeletionErr := f.Store.DeleteFolderChildren(ctx, fullPath); storeDeletionErr != nil {
	// 	return fmt.Errorf("filer store delete: %v", storeDeletionErr)
	// }

	// f.StreamListDirectoryEntries(ctx, p, "", true, int64(math.MaxInt64), "", "", "", func(entry *Entry) bool {
	// 	glog.V(3).Infof("QUYNGUYEN delete collection2 %s: %s %b %b", collectionName, entry.FullPath.Name(), isValidKeyInTime(entry.FullPath.Name(), fromDate, toDate), entry.IsDirectory())
	// 	if isValidKeyInTime(entry.FullPath.Name(), fromDate, toDate) {
	// 		if entry.IsDirectory() {
	// 			folderDeleteErr := f.Store.DeleteFolderChildren(ctx, entry.FullPath)
	// 			if folderDeleteErr != nil {
	// 				glog.V(2).Infof("QUYNGUYEN DeleteFolderChildren error %s: %v", collectionName, entry.FullPath.Name())
	// 			}
	// 		}
	// 		if storeDeletionErr := f.Store.DeleteOneEntry(ctx, entry); storeDeletionErr != nil {
	// 			glog.V(2).Infof("QUYNGUYEN delete collection DeleteOneEntry %s: %v", collectionName, entry.FullPath.Name())
	// 		}
	// 	}
	// 	glog.V(2).Infof("QUYNGUYEN delete collection2 %s: %v", collectionName, entry.FullPath.Name())

	// 	return true
	// })
	return nil
}

func (f *Filer) DoDeleteFilerEntryWithTime(ctx context.Context, collectionName string, fromTime, toTime uint64) (err error) {
	// TODO dele record in current filer

	locations := f.FilerConf.GetCollectionLocations(collectionName)
	glog.V(2).Infoln("QUYNGUYEN GetCollectionLocations ", collectionName, len(locations), locations)
	if len(locations) == 0 {
		return fmt.Errorf("QUYNGUYEN delete  collection error1 %s: %v", collectionName, err)
	}

	for _, location := range locations {
		p := util.FullPath(location)
		f.doDeleteFilerEntryWithTime(ctx, p, fromTime, toTime)
	}
	return nil
}

func (f *Filer) DoDeleteCollectionWithTime(ctx context.Context, p util.FullPath, collectionName string, fromTime, toTime uint64) (err error) {
	// glog.V(2).Infof("QUYNGUYEN: DoDeleteCollectionWithTime    delete collection %s", collectionName)
	//  // 1. Get the list of all filers in the cluster.
	existingNodes := f.ListExistingPeerUpdates(ctx)
	// 2. Call each filer to delete collection in time
	for _, node := range existingNodes {
		if node.NodeType != cluster.FilerType {
			continue
		}
		glog.V(2).Infof("QUYNGUYEN: call other filer %s to delete collection %s", node.Address, collectionName)
		// 	 // Launch a goroutine for each filer call to avoid blocking
		go func(filerAddress pb.ServerAddress) {
			pb.WithFilerClient(false, f.UniqueFilerId, filerAddress, f.GrpcDialOption, func(client filer_pb.SeaweedFilerClient) error {
				_, err := client.DeleteCollection(context.Background(), &filer_pb.DeleteCollectionRequest{
					Collection: collectionName,
					FromTime:   fromTime,
					ToTime:     toTime,
				})
				if err != nil {
					glog.Errorf("failed to delete collection %s on filer %s: %v", collectionName, filerAddress, err)
				}
				return err
			})
		}(pb.ServerAddress(node.Address))
	}

	return f.MasterClient.WithClient(false, func(client master_pb.SeaweedClient) error {
		_, err := client.CollectionDelete(context.Background(), &master_pb.CollectionDeleteRequest{
			Name:     collectionName,
			FromTime: fromTime,
			ToTime:   toTime,
		})
		if err != nil {
			glog.Infof("delete collection %s: %v", collectionName, err)
		}
		return err
	})
	// return nil
}

func (f *Filer) maybeDeleteHardLinks(hardLinkIds []HardLinkId) {
	for _, hardLinkId := range hardLinkIds {
		if err := f.Store.DeleteHardLink(context.Background(), hardLinkId); err != nil {
			glog.Errorf("delete hard link id %d : %v", hardLinkId, err)
		}
	}
}

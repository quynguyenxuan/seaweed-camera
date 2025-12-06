package filer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/master_pb"
	"github.com/seaweedfs/seaweedfs/weed/storage/needle"
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
				f.maybeDeleteHardLinks(ctx, hardLinkIds)
			}
			return nil
		})
		if err != nil {
			glog.V(2).InfofCtx(ctx, "delete directory %s: %v", p, err)
			return fmt.Errorf("delete directory %s: %v", p, err)
		}
	}

	// delete the file or folder
	err = f.doDeleteEntryMetaAndData(ctx, entry, shouldDeleteChunks, isFromOtherCluster, signatures)
	if err != nil {
		return fmt.Errorf("delete file %s: %v", p, err)
	}

	if shouldDeleteChunks && !isDeleteCollection {
		f.DeleteChunks(ctx, p, entry.GetChunks())
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
				glog.ErrorfCtx(ctx, "list folder %s: %v", entry.FullPath, err)
				return fmt.Errorf("list folder %s: %v", entry.FullPath, err)
			}
			if lastFileName == "" && !isRecursive && len(entries) > 0 {
				// only for first iteration in the loop
				glog.V(2).InfofCtx(ctx, "deleting a folder %s has children: %+v ...", entry.FullPath, entries[0].Name())
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

	glog.V(3).InfofCtx(ctx, "deleting directory %v delete chunks: %v", entry.FullPath, shouldDeleteChunks)

	if storeDeletionErr := f.Store.DeleteFolderChildren(ctx, entry.FullPath); storeDeletionErr != nil {
		return fmt.Errorf("filer store delete: %w", storeDeletionErr)
	}

	f.NotifyUpdateEvent(ctx, entry, nil, shouldDeleteChunks, isFromOtherCluster, signatures)
	f.DeleteChunks(ctx, entry.FullPath, chunksToDelete)

	return nil
}

func (f *Filer) doDeleteEntryMetaAndData(ctx context.Context, entry *Entry, shouldDeleteChunks bool, isFromOtherCluster bool, signatures []int32) (err error) {

	glog.V(3).InfofCtx(ctx, "deleting entry %v, delete chunks: %v", entry.FullPath, shouldDeleteChunks)

	if storeDeletionErr := f.Store.DeleteOneEntry(ctx, entry); storeDeletionErr != nil {
		return fmt.Errorf("filer store delete: %w", storeDeletionErr)
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
				//Chunks được xóa khi gọi tới mastermaster server
				// Delete chunks before deleting metadata no
				// f.DeleteChunks(ctx, entry.FullPath, entry.GetChunks())
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

func (f *Filer) DoDeleteCollectionWithTime(ctx context.Context, collectionName string, fromTime, toTime uint64) (err error) {
	glog.V(2).Infof("QUYNGUYEN: DoDeleteCollectionWithTime delete collection %s", collectionName)
	//QUYNGUYEN: Filer chỉ gọi Master, Master sẽ điều phối tất cả
	return f.MasterClient.WithClient(false, func(client master_pb.SeaweedClient) error {
		_, err := client.CollectionDelete(ctx, &master_pb.CollectionDeleteRequest{
			Name:     collectionName,
			FromTime: fromTime,
			ToTime:   toTime,
		})
		if err != nil {
			glog.Errorf("Failed to delete collection %s on master: %v", collectionName, err)
			return err
		}
		glog.V(3).Infof("Successfully deleted collection %s via master", collectionName)
		return nil
	})
	//Quynguyen end
}

func (f *Filer) maybeDeleteHardLinks(ctx context.Context, hardLinkIds []HardLinkId) {
	for _, hardLinkId := range hardLinkIds {
		if err := f.Store.DeleteHardLink(ctx, hardLinkId); err != nil {
			glog.ErrorfCtx(ctx, "delete hard link id %d : %v", hardLinkId, err)
		}
	}
}

// QUYNGUYEN: Delete filer entries by volume IDs
func (f *Filer) DoDeleteFilerEntryByVolumeIds(ctx context.Context, collectionName string, volumeIds []uint32) (err error) {
	glog.V(2).Infof("QUYNGUYEN: DoDeleteFilerEntryByVolumeIds deleting %d volume IDs in collection %s", len(volumeIds), collectionName)

	// Get locations for the collection
	locations := f.FilerConf.GetCollectionLocations(collectionName)
	glog.V(2).Infoln("QUYNGUYEN GetCollectionLocations ", collectionName, len(locations), locations)
	if len(locations) == 0 {
		return fmt.Errorf("QUYNGUYEN delete collection error1 %s: no locations found", collectionName)
	}

	// Convert volume IDs to needle.VolumeId slice
	vids := make([]needle.VolumeId, len(volumeIds))
	for i, vid := range volumeIds {
		vids[i] = needle.VolumeId(vid)
	}

	// Delete entries in each location
	for _, location := range locations {
		p := util.FullPath(location)
		err = f.doDeleteFilerEntryByVolumeIds(ctx, p, vids)
		if err != nil {
			glog.V(2).Infof("QUYNGUYEN: Error deleting entries in location %s: %v", location, err)
			return err
		}
	}

	glog.V(2).Infof("QUYNGUYEN: Successfully deleted entries for %d volume IDs in collection %s", len(volumeIds), collectionName)
	return nil
}

// QUYNGUYEN: Helper function to delete entries by volume IDs in a specific path
func (f *Filer) doDeleteFilerEntryByVolumeIds(ctx context.Context, fullPath util.FullPath, volumeIds []needle.VolumeId) (err error) {
	glog.V(4).Infof("QUYNGUYEN: doDeleteFilerEntryByVolumeIds in path %s for %d volume IDs", fullPath, len(volumeIds))
	
	// Create a map for faster lookup
	volumeIdMap := make(map[needle.VolumeId]bool)
	for _, vid := range volumeIds {
		volumeIdMap[vid] = true
	}

	// Optimized: Use larger batch size and early termination
	const batchSize = 4096 // Increased from 1024 for better performance
	deletedCount := 0
	
	lastFileName := ""
	includeLastFile := false
	
	for {
		entries, hasMore, err := f.ListDirectoryEntries(ctx, fullPath, lastFileName, includeLastFile, batchSize, "", "", "")
		if err != nil {
			glog.V(2).Infof("QUYNGUYEN: Error listing directory %s: %v", fullPath, err)
			return err
		}
		
		if len(entries) == 0 {
			break
		}

		// Process entries with optimized chunk checking
		batchDeleted := 0
		for _, entry := range entries {
			lastFileName = entry.Name()
			includeLastFile = false
			
			// Quick check: skip if no chunks
			if entry.Chunks == nil || len(entry.Chunks) == 0 {
				continue
			}
			
			// Optimized: early termination on first matching volume ID
			shouldDelete := false
			for _, chunk := range entry.Chunks {
				if chunk.Fid != nil && volumeIdMap[needle.VolumeId(chunk.Fid.VolumeId)] {
					shouldDelete = true
					break // Found matching volume ID, no need to check other chunks
				}
			}
			
			if shouldDelete {
				glog.V(4).Infof("QUYNGUYEN: Deleting entry %s in path %s", entry.Name(), fullPath)
				err = f.DeleteEntryMetaAndData(ctx, fullPath, true, false, false, false, nil, 0)
				if err != nil {
					glog.V(2).Infof("QUYNGUYEN: Error deleting entry %s: %v", entry.Name(), err)
					// Continue with other entries even if one fails
				} else {
					batchDeleted++
				}
			}
		}
		
		deletedCount += batchDeleted
		includeLastFile = true
		
		// Log progress for large directories
		if deletedCount > 0 && deletedCount%100 == 0 {
			glog.V(3).Infof("QUYNGUYEN: Deleted %d entries so far in path %s", deletedCount, fullPath)
		}
		
		if !hasMore {
			break
		}
	}
	
	glog.V(4).Infof("QUYNGUYEN: Completed deletion in path %s: deleted %d entries", fullPath, deletedCount)
	return nil
}

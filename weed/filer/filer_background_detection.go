package filer

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
)

func (f *Filer) loopBackgroundDetection() {
	const (
		scanInterval = 1 * time.Hour
		enabled      = true
	)

	if !enabled {
		glog.V(1).Infof("Background detection is disabled")
		return
	}

	glog.V(0).Infof("Starting background detection with interval %v", scanInterval)
	time.Sleep(30 * time.Second)

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.deletionQuit:
			glog.V(0).Infof("Background detection shutting down")
			return
		case <-ticker.C:
			f.scanAndDeleteExpiredCollections()
		}
	}
}

func (f *Filer) scanAndDeleteExpiredCollections() {
	glog.V(2).Infof("Starting background scan for expired collections")

	startTime := time.Now()
	totalExpired := int64(0)
	totalScanned := int64(0)
	totalProcessed := int64(0)

	// Get all collections from configuration
	collections := f.getAllCollections()

	for _, collection := range collections {
		select {
		case <-f.deletionQuit:
			return
		default:
		}

		totalScanned++

		// Get TTL configuration for this collection
		collectionTtls := f.FilerConf.GetCollectionTtls(collection)

		// Find the shortest TTL in this collection (most aggressive cleanup)
		minTtlDays := 0
		for _, ttlStr := range collectionTtls {
			if ttlDays := f.parseTtlToDays(ttlStr); ttlDays > 0 && (minTtlDays == 0 || ttlDays < minTtlDays) {
				minTtlDays = ttlDays
			}
		}

		// Skip collection if no TTL configured
		if minTtlDays == 0 {
			glog.V(3).Infof("Background scan skipping collection %s: no TTL configured", collection)
			continue
		}

		totalProcessed++

		// Calculate time range for expired files based on collection TTL
		now := time.Now()
		toTime := now.AddDate(0, 0, -minTtlDays).Unix() // Delete files older than TTL
		fromTime := int64(0)                            // From beginning

		glog.V(3).Infof("Background scan collection %s: using TTL %d days, deleting files before %s",
			collection, minTtlDays, time.Unix(toTime, 0).Format("2006-01-02 15:04:05"))

		// Use DoDeleteCollectionWithTime to delete expired files in collection
		err := f.DoDeleteCollectionWithTime(context.Background(), collection, uint64(fromTime), uint64(toTime))
		if err != nil {
			glog.Errorf("Background detection failed to delete expired files in collection %s: %v", collection, err)
		} else {
			totalExpired++
			glog.V(3).Infof("Background detection successfully deleted expired files in collection %s", collection)
		}
	}

	duration := time.Since(startTime)
	glog.V(1).Infof("Background detection completed: %d total collections, %d processed (with TTL), %d had expired files deleted in %v",
		totalScanned, totalProcessed, totalExpired, duration)
}

func (f *Filer) getAllCollections() []string {
	var collections []string
	seen := make(map[string]bool)

	// Get collections from configuration by walking through all rules
	f.FilerConf.rules.Walk(func(key []byte, value *filer_pb.FilerConf_PathConf) bool {
		if value.Collection != "" && !seen[value.Collection] {
			collections = append(collections, value.Collection)
			seen[value.Collection] = true
		}
		return true
	})

	glog.V(3).Infof("Found %d collections for background detection: %v", len(collections), collections)
	return collections
}

// parseTtlToDays parses TTL string (e.g., "30d", "24h", "7d") to days
func (f *Filer) parseTtlToDays(ttlStr string) int {
	if ttlStr == "" {
		return 0
	}

	ttlStr = strings.ToLower(strings.TrimSpace(ttlStr))

	if strings.HasSuffix(ttlStr, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(ttlStr, "d"))
		if err == nil && days > 0 {
			return days
		}
	} else if strings.HasSuffix(ttlStr, "h") {
		hours, err := strconv.Atoi(strings.TrimSuffix(ttlStr, "h"))
		if err == nil && hours > 0 {
			return (hours + 23) / 24 // Round up to days
		}
	} else if strings.HasSuffix(ttlStr, "m") {
		minutes, err := strconv.Atoi(strings.TrimSuffix(ttlStr, "m"))
		if err == nil && minutes > 0 {
			return (minutes + 1439) / 1440 // Round up to days
		}
	} else if strings.HasSuffix(ttlStr, "s") {
		seconds, err := strconv.Atoi(strings.TrimSuffix(ttlStr, "s"))
		if err == nil && seconds > 0 {
			return (seconds + 86399) / 86400 // Round up to days
		}
	}

	// Try to parse as plain number of days
	days, err := strconv.Atoi(ttlStr)
	if err == nil && days > 0 {
		return days
	}

	return 0
}

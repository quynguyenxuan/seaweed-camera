package weed_server

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/seaweedfs/seaweedfs/weed/filer"
	"github.com/seaweedfs/seaweedfs/weed/filer/redis3"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
)

var redisStore *redis3.Redis3Store

var fc *filer.FilerConf

var locationPrefixesMap = []*filer_pb.FilerConf_PathConf{
	{
		LocationPrefix: "/buckets/camera2009/001",
		Collection:     "001",
	},
	{
		LocationPrefix: "/buckets/camera2009/007",
		Collection:     "007",
	},
	{
		LocationPrefix: "/buckets/camera2009/015",
		Collection:     "015",
	},
	{
		LocationPrefix: "/buckets/camera2009/030",
		Collection:     "030",
	},

	{
		LocationPrefix: "/buckets/camera2010/001",
		Collection:     "001",
	},
	{
		LocationPrefix: "/buckets/camera2010/007",
		Collection:     "007",
	},
	{
		LocationPrefix: "/buckets/camera2010/015",
		Collection:     "015",
	},
	{
		LocationPrefix: "/buckets/camera2010/030",
		Collection:     "030",
	},
	{
		LocationPrefix: "/buckets/camera2011/001",
		Collection:     "001",
	},
	{
		LocationPrefix: "/buckets/camera2011/007",
		Collection:     "007",
	},
	{
		LocationPrefix: "/buckets/camera2011/015",
		Collection:     "015",
	},
	{
		LocationPrefix: "/buckets/camera2011/030",
		Collection:     "030",
	},

	{
		LocationPrefix: "/buckets/camera2012/001",
		Collection:     "001",
	},
	{
		LocationPrefix: "/buckets/camera2012/007",
		Collection:     "007",
	},
	{
		LocationPrefix: "/buckets/camera2012/015",
		Collection:     "015",
	},
	{
		LocationPrefix: "/buckets/camera2012/030",
		Collection:     "030",
	},

	{
		LocationPrefix: "/buckets/camera2013/001",
		Collection:     "001",
	},
	{
		LocationPrefix: "/buckets/camera2013/007",
		Collection:     "007",
	},
	{
		LocationPrefix: "/buckets/camera2013/015",
		Collection:     "015",
	},
	{
		LocationPrefix: "/buckets/camera2013/030",
		Collection:     "030",
	},
}

func InitializeRedis() {
	redisAddress := os.Getenv("REDIS_ADDRESS")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDatabase, _ := strconv.Atoi(os.Getenv("REDIS_DATABASES"))
	redisStore, _ = redis3.New(redisAddress, redisPassword, redisDatabase)
}
func KeyDeleteHandler(keysCh <-chan string, fromTime int, toTime int) error {
	var wg sync.WaitGroup
	taskCount := 0
	maxEntries := 1000
	finish := false
	for {
		if finish {
			break
		}
		count := 0
		var batch []string
		for key := range keysCh {
			taskCount++

			if !isValidKey(key, fromTime, toTime) {
				continue
			}
			batch = append(batch, key)
			if count++; count >= maxEntries {
				break
			}
		}
		if count == 0 {
			// Multi Objects Delete API doesn't accept empty object list, quit immediately
			break
		}
		if count < maxEntries {
			// We didn't have 1000 entries, so this is the last batch
			finish = true
		}
		taskCount++
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			// Queue all in batch.
			log.Println("QUYNGUYEN: SendDeleteObjectsMessage: ", len(batch))
			redisStore.Client.Del(context.Background(), batch...)

			if len(batch) == 0 {
				log.Println("Batch is empty: ")
				return
			}

		}(batch)
	}
	wg.Wait()
	return nil
}

func isValidKey(key string, fromDate, toDate int) bool {
	keyParts := strings.Split(key, "_")

	if len(keyParts) < 3 {
		return false
	}
	key = keyParts[2]
	keyDate, _ := strconv.Atoi(key)

	if keyDate > 0 && keyDate >= fromDate || keyDate < toDate {
		return true
	}
	return false
}
func DeleteEntryByCollectionAndTime(collection string, fromTime uint64, toTime uint64) error {
	locationPrefixes := GetLocationPrefixesMatchCollection(collection)
	fromDateNum, _ := strconv.Atoi(strings.ReplaceAll(time.Unix(int64(fromTime), 0).Format("06-01-02-15-04-05"), "-", ""))
	toDateNum, _ := strconv.Atoi(strings.ReplaceAll(time.Unix(int64(toTime), 0).Format("06-01-02-15-04-05"), "-", ""))

	for _, locationPrefix := range locationPrefixes {
		go locationPrefixHandler(locationPrefix, fromDateNum, toDateNum)
	}

	return nil
}

func locationPrefixHandler(locationPrefix string, fromTime int, toTime int) error {
	keysCh := make(chan string)
	strings.Compare("", "")
	pattern := locationPrefix + `*_` + findCommonPrefix(fromTime, toTime) + `*`
	go redisStore.ListEntriesByPattern(pattern, &keysCh)
	KeyDeleteHandler(keysCh, fromTime, toTime)
	return nil
}

func findCommonPrefix(num1, num2 int) string {
	// Chuyển số thành chuỗi
	str1 := strconv.Itoa(num1)
	str2 := strconv.Itoa(num2)

	// Tìm độ dài nhỏ nhất của hai chuỗi
	minLen := len(str1)
	if len(str2) < minLen {
		minLen = len(str2)
	}

	// So sánh từng ký tự
	for i := 0; i < minLen; i++ {
		if str1[i] != str2[i] {
			return str1[:i] // Trả về phần chung từ đầu đến vị trí khác nhau
		}
	}

	return str1[:minLen]
}

func GetLocationPrefixesMatchCollection(collection string) []string {
	var locationPrefixes []string
	for _, location := range locationPrefixesMap {
		if location.Collection == collection {
			locationPrefixes = append(locationPrefixes, location.LocationPrefix)
		}
	}
	return locationPrefixes
}

func LoadFilerConf() (err error) {

	fc := filer.NewFilerConf()

	conf := &filer_pb.FilerConf{Locations: locationPrefixesMap}
	for _, location := range conf.Locations {
		err = fc.SetLocationConf(location)
		if err != nil {
			// this is not recoverable
			return nil
		}
	}
	return

	// assert.Equal(t, "abc", fc.MatchStorageRule("/buckets/abc/jasdf").Collection)

}

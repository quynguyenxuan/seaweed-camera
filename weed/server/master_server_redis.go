package weed_server

import (
	"os"
	"strconv"

	"github.com/seaweedfs/seaweedfs/weed/filer/redis3"
)

var redisStore *redis3.Redis3Store

func InitializeRedis() {
	redisAddress := os.Getenv("REDIS_ADDRESS")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDatabase, _ := strconv.Atoi(os.Getenv("REDIS_DATABASE"))
	redisStore, _ = redis3.New(redisAddress, redisPassword, redisDatabase)
}

func DeleteColllection() {

}

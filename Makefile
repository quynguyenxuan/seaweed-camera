.PHONY: test admin-generate admin-build admin-clean admin-dev admin-run admin-test admin-fmt admin-help

BINARY = weed
ADMIN_DIR = weed/admin

SOURCE_DIR = .
debug ?= 0
AIR = ~/go/bin/air
CLEANUP_CMD = echo "🧹 Cleaning up..." && kill  $(lsof -t -i :2345) && kill  $(lsof -t -i :9333)  || true
dev:
	docker-compose -f docker-compose.yaml up -d
	watchexec -r -w ./tmp/weed -- docker-compose -f seaweedfs-dev-compose.yml restart
server-dev:
	$(AIR)
# 	dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient  exec $(AIR)
# 	@kill  $(lsof -t -i :2345)
	 
all: install

install: admin-generate
	cd weed; go install

warp_install:
	go install github.com/minio/warp@v0.7.6

full_install: admin-generate
	cd weed; go install -tags "elastic gocdk sqlite ydb tarantool tikv rclone"

server: install
	weed -v 0 server -s3 -filer -filer.maxMB=64 -volume.max=0 -master.volumeSizeLimitMB=100 -volume.preStopSeconds=1 -s3.port=8000 -s3.allowDeleteBucketNotEmpty=true -s3.config=./docker/compose/s3.json -metricsPort=9324

server1: 
	go run ./weed/main.go -v 0 server -s3 -filer -filer.maxMB=64 -volume.max=0 -master.volumeSizeLimitMB=100 -volume.preStopSeconds=1 -s3.port=8000 -s3.allowEmptyFolder=false -s3.allowDeleteBucketNotEmpty=true -s3.config=./docker/compose/s3.json -metricsPort=9324

benchmark: install warp_install
	pkill weed || true
	pkill warp || true
	weed server -debug=$(debug) -s3 -filer -volume.max=0 -master.volumeSizeLimitMB=100 -volume.preStopSeconds=1 -s3.port=8000 -s3.allowDeleteBucketNotEmpty=false -s3.config=./docker/compose/s3.json &
	warp client &
	while ! nc -z localhost 8000 ; do sleep 1 ; done
	warp mixed --host=127.0.0.1:8000 --access-key=some_access_key1 --secret-key=some_secret_key1 --autoterm
	pkill warp
	pkill weed

# curl -o profile "http://127.0.0.1:6060/debug/pprof/profile?debug=1"
benchmark_with_pprof: debug = 1
benchmark_with_pprof: benchmark

test: admin-generate
	cd weed; go test -tags "elastic gocdk sqlite ydb tarantool tikv rclone" -v ./...

# Admin component targets
admin-generate:
	@echo "Generating admin component templates..."
	@cd $(ADMIN_DIR) && $(MAKE) generate

admin-build: admin-generate
	@echo "Building admin component..."
	@cd $(ADMIN_DIR) && $(MAKE) build

admin-clean:
	@echo "Cleaning admin component..."
	@cd $(ADMIN_DIR) && $(MAKE) clean

admin-dev:
	@echo "Starting admin development server..."
	@cd $(ADMIN_DIR) && $(MAKE) dev

admin-run:
	@echo "Running admin server..."
	@cd $(ADMIN_DIR) && $(MAKE) run

admin-test:
	@echo "Testing admin component..."
	@cd $(ADMIN_DIR) && $(MAKE) test

admin-fmt:
	@echo "Formatting admin component..."
	@cd $(ADMIN_DIR) && $(MAKE) fmt

admin-help:
	@echo "Admin component help..."
	@cd $(ADMIN_DIR) && $(MAKE) help
server-docker:
	docker run --rm \
	-v "./:/app" \
	-w /app \
	-p 9333:9333 \
	-p 19333:19333 \
	--name seaweedfs \
	golang:1.24 \
	go run /app/weed/weed.go \
	-v 0 server -s3 -filer.maxMB=64 -volume.max=0 \
	-master.volumeSizeLimitMB=100 \
	-volume.preStopSeconds=1 \
	-master.defaultReplication=000 \
	-volume.dir=/mnt/dc1/volumes/volume_12/v2 

server-volume:
	docker run --rm \
	-v "./:/app" \
	-w /app \
	--name seaweedfs-volume \
	golang:1.24 \
	go run /app/weed/weed.go \
	-v 0 volume \
	-max=0 \
	-preStopSeconds=1 \
	-dir=/mnt/dc1/volumes/volume_12/v3 \
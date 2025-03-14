buckets=("camera2009" "camera2010" "camera2011" "camera2012")
packages=("007" "015" "030" "003")

retries 3
timeout connect 5s
timeout server 30s

do_put() {
    local i=$(date +'%y%m%d%H%M%S')
    

    local random_package=$(printf "%s\n" "${packages[@]}" | shuf -n 1)
    local random_bucket=$(printf "%s\n" "${buckets[@]}" | shuf -n 1)

    echo "Putting $i $random_bucket $random_package"
    warp put --duration=30m  \
    --host=127.0.0.1:80 \
    --access-key=5_test-public_accid \
    --secret-key=WGdv7-UdOLiDFbzOa6C2MG3TKy8P \
    --obj.size=512K \
    --bucket=${random_bucket} \
    --concurrent=200 \
    --noclear \
    --prefix=${random_package}/${random_package}_kdfjksdf_${i}_234234234
}

do_run () {
    echo "Putting $i"
    do_put $i
    local NOW=$(date +%s)
    sleep 3
    do_put $i
    sleep 3
    echo "Deleting $i"
    curl "http://localhost:9333/col/delete?collection=007&fromTime=1732953740&toTime=${NOW}&pretty=y"
    sleep 3
}

for i in $(seq 1 200);
do

    do_put $i
    
done

# wget http://103.5.211.42:80/camera2009/007/007_kdfjksdf_250228155600_234234234/%28JD%299F4h/10100.%28rmE2VABwlU74Kzm.rnd
# time curl "http://localhost:9333/col/delete?collection=007&fromTime=1732953740&toTime=$(date +%s)&pretty=y"
# curl "http://localhost:9333/col/delete?collection=007&fromTime=1732953740&toTime=$(date +%s)&pretty=y"

#  warp put --duration=1s  \
#  --host=127.0.0.1:80 \
#  --access-key=8_tests3_accid \
#  --secret-key=-aJ20yurXb2RhF9pYwNG9shc-RKb \
#  --obj.size=512K \
#  --bucket=camera2013 \
#  --concurrent=1 \
#  --noclear \
#  --prefix=camera2013

#  curl -F file=@filer.conf "http://localhost:8800/etc/seaweedfs/"
#  curl -F file=@s3.json "http://localhost:8800/etc/iam/"

#  curl -F file=@filer.conf "http://localhost:8888/buc-releas11/seaweedfs/"

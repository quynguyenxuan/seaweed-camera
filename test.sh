 

warp put --duration=1h      --host=localhost:8334     --obj.size=1M     --obj.randsize=false     --obj.nametemplate="bbb_%d%s_$(date +'%y%m%d%H%M%S')_2343234234.m3u8"     --bucket=aaa     --concurrent=1000     --noprefix     --noclear     --prefix=aaa

go run weed.go -v=1  volume -dir=/mnt/dc1/volumes/volume_12/v2 -preStopSeconds=3 -max=1

do_put() {
    warp put --duration=90s  \
    --host=127.0.0.1:8333 \
    --obj.size=512K \
    --bucket=camera2020 \
    --concurrent=1 \
    --noprefix \
    --noclear \
    --prefix=007

    #   --access-key=8_tests3_accid \
    # --secret-key=-aJ20yurXb2RhF9pYwNG9shc-RKb \
}

curl -X DELETE "http://localhost:8888/buckets/camera2051?collection=camera2051&fromTime=1732953740&toTime=1764317882"
curl -X DELETE "http://localhost:8888/buckets/camera2010/007?collection=camera2010_007&fromTime=1732953740&toTime=1744103784"

curl -X DELETE "http://103.5.211.42:8800/buckets/camera2010/007?collection=camera2019_001&fromTime=1742112108&toTime=1742119308"

curl -F file=@filer.conf "http://localhost:8888/buckets/camera2010/007/007_kdfjksdf_$(date +'%y%m%d%H%M%S')_2343234234"
curl -X DELETE "http://localhost:8888/buckets/camera2010/007?collection=camera2010_007&fromTime=1732953740&toTime=$(date +%s)"
export NOW=$(date +%s)
curl -X DELETE "http://localhost:8888/buckets/camera2010/007?collection=camera2010_007&fromTime=1732953740&toTime=${NOW}"
docker container logs seaweedfs385-filer-1 -f    

Collection list
camera2009/007
camera2009/015
camera2009/030
camera2009/003

for i in $(seq 1 100);
do

    echo "Deleting $i"
    do_put $i
    local NOW=$(date +%s)
    sleep 3
    do_put $i
    sleep 3

    curl "http://localhost:9333/col/delete?collection=007&fromTime=1732953740&toTime=${NOW}&pretty=y"
    sleep 3
    echo "Putting $i"
done

time AWS_ACCESS_KEY_ID=5_test-public_accid \
AWS_SECRET_ACCESS_KEY=WGdv7-UdOLiDFbzOa6C2MG3TKy8P \
 aws s3api put-object \
--bucket camera2009 \
--key "007/007_kdfjksdf_250228155600_234234234/%28JD%299F4h/10100.%sdfssdf.rnd" \
--endpoint-url http://103.5.211.42:80 \
--body video.m4a

http://localhost:8888/buckets/camera2020/007/007_kdfjksdf_250228155600_234234234/h5DQKP(Q/11.iKIYrGM8D)9OaIbr.rnd
aws s3api get-object --bucket camera2020 --key "007/007_kdfjksdf_250228155600_234234234/h5DQKP(Q/11.iKIYrGM8D)9OaIbr.rnd" example.txt --endpoint-url=http://localhost:8333 --profile=local
aws s3api list-objects-v2 --bucket camera2020 --prefix 007/007_ --endpoint-url=http://localhost:8333 --no-sign-request
aws s3api list-objects-v2 --bucket camera2020 --prefix "007" --endpoint-url=http://localhost:8333 --profile=local  
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

#  curl -F file=@filer.conf "http://localhost:8888/etc/seaweedfs/"
#  curl -F file=@filer.conf "http://localhost:8888/buc-releas11/seaweedfs/"

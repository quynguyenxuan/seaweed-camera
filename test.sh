 

do_put() {
    warp put --duration=1s  \
    --host=127.0.0.1:8333 \
    --obj.size=512K \
    --bucket=camera2009 \
    --concurrent=1 \
    --noclear \
    --prefix=007/007_kdfjksdf_250228155600_234234234
}


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

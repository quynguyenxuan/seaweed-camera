#!/bin/bash

. ../.env

echo "Các thao tác tải lên nhiều phần"
export AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
# AWS_S3_ENDPOINT_URL="http://localhost:8333"
# S3_BUCKET="test-bucket"
FILE_SIZE=25000000
echo $AWS_ACCESS_KEY_ID/$AWS_SECRET_ACCESS_KEY
set +x
PUT_BUCKET_RES=$(aws s3api create-bucket --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã tạo bucket: $S3_BUCKET"
echo "$PUT_BUCKET_RES"
-master.defaultReplicatio

# NewMultipartUpload
echo "Tạo tải lên nhiều phần mới..."
UPLOAD_ID=$(aws s3api create-multipart-upload --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --key $S3_OBJECT | jq -r '.UploadId')
echo "✅ Đã tạo UploadId: $UPLOAD_ID"
dd if=/dev/urandom of="$KEY_NAME" bs=1 count=$FILE_SIZE 2>/dev/null
dd if=$S3_OBJECT of=part1.txt bs=1 count=10485760 skip=0 2>/dev/null
dd if=$S3_OBJECT of=part2.txt bs=1 count=10485760 skip=10485760 2>/dev/null
# PutObjectPart
echo "Tải lên phần 1..."
ETAG1=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api upload-part --bucket $S3_BUCKET --key $S3_OBJECT --part-number 1 --upload-id $UPLOAD_ID --body ./part1.txt | jq -r '.ETag')
echo "✅ ETag Phần 1: $ETAG1"

echo "Tải lên phần 2..."
ETAG2=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api upload-part --bucket $S3_BUCKET --key $S3_OBJECT --part-number 2 --upload-id $UPLOAD_ID --body ./part2.txt | jq -r '.ETag')
echo "✅ ETag Phần 2: $ETAG2"

# ListObjectParts
echo "Liệt kê các phần..."
LIST_PARTS_RES=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api list-parts --bucket $S3_BUCKET --key $S3_OBJECT --upload-id $UPLOAD_ID --output json)
echo "✅ Danh sách các phần:"
echo "$LIST_PARTS_RES"

# CompleteMultipartUpload
echo "Hoàn thành tải lên nhiều phần..."
COMPLETE_RES=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api complete-multipart-upload --bucket $S3_BUCKET --key $S3_OBJECT --upload-id $UPLOAD_ID --multipart-upload "{\"Parts\": [{\"ETag\": $ETAG1, \"PartNumber\": 1}, {\"ETag\": $ETAG2, \"PartNumber\": 2}]}" --output json)
echo "✅ Đã hoàn thành tải lên nhiều phần:"
echo "$COMPLETE_RES"

# ListMultipartUploads
echo "Liệt kê các tải lên nhiều phần (nên trống)..."
LIST_UPLOADS_RES=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api list-multipart-uploads --bucket $S3_BUCKET --output json)
echo "✅ Danh sách tải lên nhiều phần:"
echo "$LIST_UPLOADS_RES"

# AbortMultipartUpload (ví dụ, không thực thi trong luồng này)
echo "Ví dụ về hủy tải lên nhiều phần (không thực thi)..."
UPLOAD_ID_TO_ABORT=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api create-multipart-upload --bucket $S3_BUCKET --key file_to_abort.txt | jq -r '.UploadId')
ABORT_RES=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api abort-multipart-upload --bucket $S3_BUCKET --key file_to_abort.txt --upload-id $UPLOAD_ID_TO_ABORT --output json)
echo "✅ Đã hủy UploadId: $UPLOAD_ID_TO_ABORT"
echo "$ABORT_RES"


 -volume.inflightDownloadDataTimeout=30s
        inflight download data wait timeout of volume servers (default 1m0s)
  -volume.inflightUploadDataTimeout=30s
  -volume.max=0
   -filer.defaultReplicaPlacement=000
#!/bin/bash

bash source ../.env

echo "Các thao tác tải lên nhiều phần"
export AWS_ACCESS_KEY_ID="some_access_key1"
export AWS_SECRET_ACCESS_KEY="some_secret_key1"

PUT_BUCKET_RES=$(aws s3api create-bucket --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã tạo bucket: $S3_BUCKET"
echo "$PUT_BUCKET_RES"

# NewMultipartUpload
echo "Tạo tải lên nhiều phần mới..."
UPLOAD_ID=$(aws s3api create-multipart-upload --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --key testfile.txt | jq -r '.UploadId')
echo "✅ Đã tạo UploadId: $UPLOAD_ID"

# PutObjectPart
echo "Tải lên phần 1..."
ETAG1=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api upload-part --bucket $S3_BUCKET --key testfile.txt --part-number 1 --upload-id $UPLOAD_ID --body ./data.txt | jq -r '.ETag')
echo "✅ ETag Phần 1: $ETAG1"

echo "Tải lên phần 2..."
ETAG2=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api upload-part --bucket $S3_BUCKET --key testfile.txt --part-number 2 --upload-id $UPLOAD_ID --body ./data.txt | jq -r '.ETag')
echo "✅ ETag Phần 2: $ETAG2"

# ListObjectParts
echo "Liệt kê các phần..."
LIST_PARTS_RES=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api list-parts --bucket $S3_BUCKET --key testfile.txt --upload-id $UPLOAD_ID --output json)
echo "✅ Danh sách các phần:"
echo "$LIST_PARTS_RES"

# CompleteMultipartUpload
echo "Hoàn thành tải lên nhiều phần..."
COMPLETE_RES=$(aws --endpoint-url $AWS_S3_ENDPOINT_URL s3api complete-multipart-upload --bucket $S3_BUCKET --key testfile.txt --upload-id $UPLOAD_ID --multipart-upload "{\"Parts\": [{\"ETag\": $ETAG1, \"PartNumber\": 1}, {\"ETag\": $ETAG2, \"PartNumber\": 2}]}" --output json)
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

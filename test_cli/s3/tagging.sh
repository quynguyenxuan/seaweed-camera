#!/bin/bash

bash source ../.env
export AWS_ACCESS_KEY_ID="some_access_key1"
export AWS_SECRET_ACCESS_KEY="some_secret_key1"
echo "Các thao tác gắn thẻ đối tượng"

# Tạo bucket để test
echo "Tạo bucket để test..."
BUCKET_RES=$(aws s3api create-bucket --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)

# Tạo đối tượng để test
echo "Tạo đối tượng để test..."
echo "Đây là nội dung của đối tượng thử nghiệm cho tagging." > testfile.txt
OBJECT_RES=$(aws s3api put-object --bucket $S3_BUCKET --key testfile.txt --body testfile.txt --endpoint-url $AWS_S3_ENDPOINT_URL --output json)

# PutObjectTagging
echo "Gắn thẻ cho đối tượng..."
TAGGING_RES=$(aws s3api put-object-tagging --bucket $S3_BUCKET --key testfile.txt --tagging '{"TagSet": [{"Key": "Environment", "Value": "Test"}, {"Key": "Project", "Value": "SeaweedFS"}]}' --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã gắn thẻ cho đối tượng: testfile.txt"
echo "$TAGGING_RES"

# GetObjectTagging
echo "Lấy thẻ của đối tượng..."
GET_TAGGING_RES=$(aws s3api get-object-tagging --bucket $S3_BUCKET --key testfile.txt --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã lấy thẻ của đối tượng: testfile.txt"
echo "$GET_TAGGING_RES"

# DeleteObjectTagging
echo "Xóa thẻ của đối tượng..."
DELETE_TAGGING_RES=$(aws s3api delete-object-tagging --bucket $S3_BUCKET --key testfile.txt --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã xóa thẻ của đối tượng: testfile.txt"
echo "$DELETE_TAGGING_RES"

# Kiểm tra lại GetObjectTagging sau khi xóa
echo "Kiểm tra lại thẻ của đối tượng sau khi xóa..."
GET_TAGGING_AFTER_DELETE_RES=$(aws s3api get-object-tagging --bucket $S3_BUCKET --key testfile.txt --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Kiểm tra lại thẻ của đối tượng sau khi xóa:"
echo "$GET_TAGGING_AFTER_DELETE_RES"

# Dọn dẹp
echo "Dọn dẹp..."
aws s3api delete-object --bucket $S3_BUCKET --key testfile.txt --endpoint-url $AWS_S3_ENDPOINT_URL --output json
# aws s3api delete-bucket --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json
rm testfile.txt

echo "✅ Đã hoàn thành các thao tác gắn thẻ đối tượng"
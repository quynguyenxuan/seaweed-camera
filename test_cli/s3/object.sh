. ../.env
// Object operations
# * PutObject
# * GetObject
# * HeadObject
# * CopyObject
# * DeleteObject
# * ListObjectsV2
# * ListObjectsV1
# * DeleteMultipleObjects
# * PostPolicy
export AWS_ACCESS_KEY_ID="some_access_key1"
export AWS_SECRET_ACCESS_KEY="some_secret_key1"
echo $AWS_ACCESS_KEY_ID/$AWS_SECRET_ACCESS_KEY
PUT_BUCKET_RES=$(aws s3api create-bucket --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã tạo bucket: $S3_BUCKET"
echo "$PUT_BUCKET_RES"

# PutObject
echo "--- Đặt đối tượng ---"
OBJECT_KEY="data.txt"
echo "Đây là nội dung của đối tượng thử nghiệm." > $OBJECT_KEY
PUT_OBJECT_RES=$(aws s3api put-object --bucket $S3_BUCKET --key $OBJECT_KEY --body $OBJECT_KEY --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã đặt đối tượng: $OBJECT_KEY"
echo "$PUT_OBJECT_RES"
rm $OBJECT_KEY

# GetObject
echo "--- Lấy đối tượng ---"
GET_OBJECT_RES=$(aws s3api get-object --bucket $S3_BUCKET --key $OBJECT_KEY --endpoint-url $AWS_S3_ENDPOINT_URL $OBJECT_KEY.downloaded)
echo "✅ Đã lấy đối tượng: $OBJECT_KEY"
cat $OBJECT_KEY.downloaded
rm $OBJECT_KEY.downloaded

# HeadObject
echo "--- Head đối tượng ---"
HEAD_OBJECT_RES=$(aws s3api head-object --bucket $S3_BUCKET --key $OBJECT_KEY --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã head đối tượng: $OBJECT_KEY"
echo "$HEAD_OBJECT_RES"

# CopyObject
echo "--- Sao chép đối tượng ---"
COPY_OBJECT_KEY="test-object-copy.txt"
COPY_OBJECT_RES=$(aws s3api copy-object --bucket $S3_BUCKET --key $COPY_OBJECT_KEY --copy-source "$S3_BUCKET/$OBJECT_KEY" --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã sao chép đối tượng: $OBJECT_KEY sang $COPY_OBJECT_KEY"
echo "$COPY_OBJECT_RES"

# ListObjectsV2
echo "--- Liệt kê đối tượng (V2) ---"
LIST_OBJECTS_V2_RES=$(aws s3api list-objects-v2 --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã liệt kê đối tượng (V2) trong bucket: $S3_BUCKET"
echo "$LIST_OBJECTS_V2_RES"

# ListObjectsV1
echo "--- Liệt kê đối tượng (V1) ---"
LIST_OBJECTS_V1_RES=$(aws s3api list-objects --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã liệt kê đối tượng (V1) trong bucket: $S3_BUCKET"
echo "$LIST_OBJECTS_V1_RES"

# DeleteObject
echo "--- Xóa đối tượng ---"
DELETE_OBJECT_RES=$(aws s3api delete-object --bucket $S3_BUCKET --key $OBJECT_KEY --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã xóa đối tượng: $OBJECT_KEY"
echo "$DELETE_OBJECT_RES"

# DeleteMultipleObjects
echo "--- Xóa nhiều đối tượng ---"
DELETE_MULTIPLE_OBJECTS_RES=$(aws s3api delete-objects --bucket $S3_BUCKET --delete "Objects=[{Key=$COPY_OBJECT_KEY}]" --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã xóa nhiều đối tượng trong bucket: $S3_BUCKET"
echo "$DELETE_MULTIPLE_OBJECTS_RES"

# PostPolicy (This is more complex and usually involves generating a signed policy. For a simple CLI test, we'll just show the command structure.)
echo "--- PostPolicy (chỉ cấu trúc lệnh) ---"
echo "Để sử dụng PostPolicy, bạn cần tạo một chính sách và ký nó. Ví dụ:"
echo "aws s3api presign-post --bucket $S3_BUCKET --key some_file.txt --expires-in 3600 --endpoint-url $AWS_S3_ENDPOINT_URL"

# Clean up the bucket
echo "--- Dọn dẹp bucket ---"
aws s3api delete-bucket --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL
echo "✅ Đã xóa bucket: $S3_BUCKET"
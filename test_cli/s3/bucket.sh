

# // Bucket operations
# * PutBucket
# * DeleteBucket
# * HeadBucket
# * ListBuckets
# * PutBucketLifecycleConfiguration (partially, only for TTL)
# * GetBucketLifecycleConfiguration (partially, only for TTL)
# * DeleteBucketLifecycleConfiguration (partially, only for TTL)
# * GetBucketCors
# * PutBucketCors
# * DeleteBucketCors
bash source ../.env
export AWS_ACCESS_KEY_ID="some_access_key1"
export AWS_SECRET_ACCESS_KEY="some_secret_key1"
# PutBucket
# PutBucket
echo $AWS_ACCESS_KEY_ID/$AWS_SECRET_ACCESS_KEY
PUT_BUCKET_RES=$(aws s3api create-bucket --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã tạo bucket: $S3_BUCKET"
echo "$PUT_BUCKET_RES"



# HeadBucket
HEAD_BUCKET_RES=$(aws s3api head-bucket --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã kiểm tra bucket: $S3_BUCKET"
echo "$HEAD_BUCKET_RES"

# ListBuckets
LIST_BUCKETS_RES=$(aws s3api list-buckets --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Danh sách bucket:"
echo "$LIST_BUCKETS_RES"

# PutBucketLifecycleConfiguration (partially, only for TTL)
PUT_BUCKET_LC_RES=$(aws s3api put-bucket-lifecycle-configuration --bucket $S3_BUCKET --lifecycle-configuration file://lifecycle.json --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã cấu hình lifecycle cho bucket: $S3_BUCKET"
echo "$PUT_BUCKET_LC_RES"

# GetBucketLifecycleConfiguration (partially, only for TTL)
GET_BUCKET_LC_RES=$(aws s3api get-bucket-lifecycle-configuration --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Lấy cấu hình lifecycle cho bucket: $S3_BUCKET"
echo "$GET_BUCKET_LC_RES"

# DeleteBucketLifecycleConfiguration (partially, only for TTL)
DELETE_BUCKET_LC_RES=$(aws s3api delete-bucket-lifecycle --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã xóa cấu hình lifecycle cho bucket: $S3_BUCKET"
echo "$DELETE_BUCKET_LC_RES"

# PutBucketCors
PUT_BUCKET_CORS_RES=$(aws s3api put-bucket-cors --bucket $S3_BUCKET --cors-configuration file://cors.json --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã cấu hình CORS cho bucket: $S3_BUCKET"
echo "$PUT_BUCKET_CORS_RES"

# GetBucketCors
GET_BUCKET_CORS_RES=$(aws s3api get-bucket-cors --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Lấy cấu hình CORS cho bucket: $S3_BUCKET"
echo "$GET_BUCKET_CORS_RES"

# DeleteBucketCors
DELETE_BUCKET_CORS_RES=$(aws s3api delete-bucket-cors --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã xóa cấu hình CORS cho bucket: $S3_BUCKET"
echo "$DELETE_BUCKET_CORS_RES"


# DeleteBucket
DELETE_BUCKET_RES=$(aws s3api delete-bucket --bucket $S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL --output json)
echo "✅ Đã xóa bucket: $S3_BUCKET"
echo "$DELETE_BUCKET_RES"
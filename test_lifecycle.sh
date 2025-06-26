
# export AWS_S3_ADDRESSING_STYLE="virtual"
export AWS_ACCESS_KEY_ID="8_tests3_accid"
export AWS_SECRET_ACCESS_KEY="-aJ20yurXb2RhF9pYwNG9shc-RKb"
export AWS_S3_ENDPOINT_URL="http://localhost"
export AWS_ENDPOINT_URL="http://localhost"
export AWS_S3_ADDRESSING_STYLE="path"
export AWS_DEFAULT_REGION="us-east-1"
export AWS_REGION="us-east-1"



# aws s3api put-bucket-lifecycle-configuration \
#     --bucket camera2024 \
#     --lifecycle-configuration file://lifecycle-policy.json

curl -X "POST"  "http://localhost:8888/buckets/camera2027/?ttl=1d"

curl -H "Accept: application/json"  "http://localhost:8888/buckets?pretty=y"

# curl -X "PUT" "http://localhost:9333/dir/assign?ttl=20m&collection=camera2030"

# aws s3api get-bucket-lifecycle-configuration \
#     --bucket camera2024

# export AWS_ACCESS_KEY_ID=AKIA40K8EP9D27CI77PY
# export AWS_SECRET_ACCESS_KEY=A+pfV58aUUyXoIcy+1k6pBkn5dpJJTLsPnYsbOr3
# export AWS_SESSION_TOKEN="123hsjfhsdfj"

# export AWS_ACCESS_KEY_ID=$1
# export AWS_SECRET_ACCESS_KEY=$2
# export AWS_SESSION_TOKEN=$3
# export AWS_DEFAULT_REGION="us-east-1"
export AWS_S3_ENDPOINT_URL=http://localhost:8333
export AWS_STS_ENDPOINT_URL=http://localhost:8111

export AWS_S3_ADDRESSING_STYLE="path"
export AWS_ACCESS_KEY_ID="Q6WBAGZ4F9P2OEPLFKWZ"
# "SX7N95SB15XMHVKJCCWNU"
export AWS_SECRET_ACCESS_KEY="ucGD69wxHreul9by41jCjkS4l9M7agh1K89RUPdr"
# "tLko8YikRA9Y7CZ1L8IyuWfVZdvw6sSMFtB0NZv4k0"

# export AWS_SESSION_TOKEN=$(bun run test_gen_token.ts)
# STRING="Hello, this is a test string!"
# PUT_URL=$(python3 test-create-presigned.py --bucket camera2022 --key test2.txt --method put_object --access_key="${AWS_ACCESS_KEY_ID}" --secret_key="${AWS_SECRET_ACCESS_KEY}" --session_token="$AWS_SESSION_TOKEN" --expires=3600)
# echo $PUT_URL
# curl -X PUT --data "$STRING" "$PUT_URL"

# GET_URL=$(python3 test-create-presigned.py --bucket camera2022 --key test2.txt --method get_object --access_key=$AWS_ACCESS_KEY_ID --secret_key=$AWS_SECRET_ACCESS_KEY  --session_token="$AWS_SESSION_TOKEN" --expires=3600)
# echo $GET_URL
# curl -X GET "${GET_URL}"

# aws s3 ls --endpoint $AWS_S3_ENDPOINT_URL
# echo '{
#   "Rules": [
#     {
#       "ID": "DeleteAfter30Days",
#       "Status": "Enabled",
#       "Filter": {
#         "Prefix": ""
#       },
#       "Expiration": {
#         "Days": 30
#       },
#       "AbortIncompleteMultipartUpload": {
#         "DaysAfterInitiation": 1
#       }
#     }
#   ]
# }' > lifecycle.json


# aws s3api put-bucket-lifecycle-configuration --bucket camera2024 --lifecycle-configuration file://lifecycle.json --endpoint $AWS_S3_ENDPOINT_URL
# set +x
# GET_URL=$(aws s3 presign  --endpoint-url http://localhost:8333  's3://camera2024/test.txt' --expires-in 3600 --region=us-east-1)
#
# curl -X POST -T test.txt "${URL}"
# curl -X PUT --data "hSDFHSDJFHJ" "$URL"
# aws s3 ls --endpoint-url https://s3-viettel.sunteco.cloud
# aws --endpoint http://127.0.0.1:8111 iam create-access-key --region us-east-1  --user-name Bob
# export AWS_ACCESS_KEY_ID=
# export AWS_SECRET_ACCESS_KEY=
# aws s3 mb s3://my-unique-bucket --endpoint-url http://localhost:8333 --no-verify-ssl --region us-east-1

# s5cmd  --endpoint-url http://localhost:8333 --no-verify-ssl mb s3://my-unique-bucket
# aws s3 mb s3://my-bucket-12345 --endpoint-url http://localhost:3000 --no-verify-ssl --region us-east-1
# aws iam create-user --user-name my-user22 \
#   --endpoint-url http://localhost:8111 \
#   --no-verify-ssl \
#   --region us-east-1
# aws iam list-access-keys --user-name quynguyen --endpoint-url http://localhost:80 --no-verify-ssl --region us-east-1
# aws iam list-users --endpoint-url http://localhost:80 --region us-east-1
# aws --endpoint http://127.0.0.1:8111 iam create-access-key --user-name Bob
# aws s3api create-bucket \
# --bucket camera2029 \
# --endpoint-url http://127.0.0.1:80
# aws --endpoint http://127.0.0.1:8111 iam list-access-keys --region us-east-1 --user-name admin
# aws --endpoint http://127.0.0.1:8100 --region us-east-1 s3api list-buckets
# aws --endpoint http://127.0.0.1:8100 --region us-east-1 iam list-user-policies --user-name admin
# aws iam get-user-policy --endpoint http://127.0.0.1:8100 --region us-east-1 --user-name admin --policy-name all
# aws sts assume-role \
#   --endpoint-url https://sts-viettel.sunteco.cloud \
#   --region us-east-1 \
#   --role-arn arn:custom:iam::123456789012:role/MyRole \
#   --duration-seconds 900 \
#   --role-session-name my-session \
#   --output json
#
# aws sts assume-role  \
# --endpoint-url https://sts-viettel.sunteco.cloud \
# --role-arn arn:custom:iam::123456789012:role/MyRole \
# --role-session-name my-session \
# --region us-east-1


# export AWS_ACCESS_KEY_ID="AKIAULT61F3PX2YDCNLS"
# export AWS_SECRET_ACCESS_KEY="nbgg9ehIx2mDV2ZCaPkCGExEDwhC9uMv3OIxT4OO"
# export AWS_SESSION_TOKEN="eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NLZXkiOiJBS0lBVUxUNjFGM1BYMllEQ05MUyIsImV4cCI6MTc0OTQzOTM1MiwiaWF0IjoxNzQ5NDM4NDUyfQ.EeCLGmmrCX38xMTUIwDp607fHj4Hqzdpdb7joHGfkvZnFkr_Itu-l67cTNSCrXUzyAa-5vzJCMoHQERTyptjXw"

aws iam create-access-key  --endpoint-url $AWS_STS_ENDPOINT_URL --user-name Bob2 --region us-east-1
# aws iam list-access-keys --endpoint-url $AWS_STS_ENDPOINT_URL --user-name Bob --region us-east-1
# # exit 1
S3_ROLE=$(aws sts assume-role  --endpoint-url $AWS_STS_ENDPOINT_URL --role-arn arn:custom:iam::123456789012:role/MyRole --role-session-name my-session --region us-east-1)
echo "STS response ${S3_ROLE}"
# export AWS_ACCESS_KEY_ID=$(echo "$S3_ROLE" | jq -r '.Credentials.AccessKeyId')
# export AWS_SECRET_ACCESS_KEY=$(echo "$S3_ROLE" | jq -r '.Credentials.SecretAccessKey')
# export AWS_SESSION_TOKEN=$(echo "$S3_ROLE" | jq -r '.Credentials.SessionToken')

echo AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID
echo AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
echo AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN
sleep 5
# echo "Put object test.txt"
# aws s3 cp test_sts.sh  s3://camera2024/backup/test.txt  --region us-east-1 --endpoint-url $AWS_S3_ENDPOINT_URL --checksum-algorithm SHA256
# echo "Get object test.txt"
# aws s3 cp s3://cameravttnew-day3-1/backup/test.txt ./test.txt  --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud
# echo "Get presigned URL"
# aws s3 presign s3://cameravttnew-day3-1/backup/test.txt   --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud
echo "List objects"
# aws s3 ls s3://cameravttnew-day3-1/backup  --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud  --recursive --human-readable --summarize --no-paginate   --output text
#

set -x
# aws s3 ls s3://camera2024/backup  --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
# aws s3 ls s3://camera2024/backu  --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
# aws s3 ls s3://camera2024/backup/  --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
# aws s3 ls s3://camera2024/backup/t  --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text


# aws s3api put-object --bucket cameravttnew-day3-1 --key backup/test.txt --body test.txt --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud

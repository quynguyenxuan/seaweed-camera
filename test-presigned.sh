export AWS_ACCESS_KEY_ID=""
export AWS_SECRET_ACCESS_KEY=""
# export AWS_SESSION_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NLZXkiOiI4X3Rlc3RzM19hY2NpZCIsImV4cCI6MTc1MTAwNzE1OCwiaWF0IjoxNzQ4MzMyOTQ0fQ.qOVdPAEIxmjvYqKenr-sBuab3NAA0L81wZbmS7lKveI"
export AWS_DEFAULT_REGION="us-east-1"
export AWS_S3_ADDRESSING_STYLE="path"
export AWS_SESSION_TOKEN=$(bun run test_gen_token.ts)
# set -x
STRING="Hello, this is a test dfd sdfsd string!"

PUT_URL=$(python3 test-create-presigned.py --bucket camera2024 --key test.txt --method put_object --access_key="${AWS_ACCESS_KEY_ID}" --secret_key="${AWS_SECRET_ACCESS_KEY}" --session_token="${AWS_SESSION_TOKEN}" --expires=3600)
# sleep 3
echo $PUT_URL
curl -X PUT --data "$STRING" "$PUT_URL"

GET_URL=$(python3 test-create-presigned.py --bucket camera2024 --key test.txt --method get_object --access_key=$AWS_ACCESS_KEY_ID --secret_key=$AWS_SECRET_ACCESS_KEY --session_token="$AWS_SESSION_TOKEN" --expires=3600)
# sleep 6
echo $GET_URL
curl -X GET "${GET_URL}"

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
#   --endpoint-url http://localhost:3000 \
#   --no-verify-ssl \
#   --region us-east-1 \
#   --role-arn arn:custom:iam::123456789012:role/MyRole \
#   --duration-seconds 900 \
#   --role-session-name my-session11
# # aws sts get-session-token \
# #   --endpoint-url http://localhost:3000 \
# #   --no-verify-ssl \
# #   --region us-east-1


# aws sts get-federation-token \
#   --endpoint-url http://localhost:3000 \
#   --name my-session \
#   --no-verify-ssl \
#   --region us-east-1

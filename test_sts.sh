
# export AWS_S3_ADDRESSING_STYLE="virtual"
export AWS_ACCESS_KEY_ID="8_tests3_accid"
export AWS_SECRET_ACCESS_KEY="-aJ20yurXb2RhF9pYwNG9shc-RKb"
export AWS_S3_ENDPOINT_URL="http://localhost"
export AWS_ENDPOINT_URL="http://localhost"
export AWS_S3_ADDRESSING_STYLE="path"


# S3_ROLE=$(aws sts assume-role  --endpoint-url https://sts-viettel.sunteco.cloud --role-arn arn:custom:iam::123456789012:role/MyRole --role-session-name my-session --region us-east-1)
# echo "STS response ${S3_ROLE}"
# export AWS_ACCESS_KEY_ID=$(echo "$S3_ROLE" | jq -r '.Credentials.AccessKeyId')
# export AWS_SECRET_ACCESS_KEY=$(echo "$S3_ROLE" | jq -r '.Credentials.SecretAccessKey')
# export AWS_SESSION_TOKEN=$(echo "$S3_ROLE" | jq -r '.Credentials.SessionToken')

# echo AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID
# echo AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
# echo AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN
# sleep 10
# echo "Put object test.txt"
# aws s3 cp test.txt  s3://cameravttnew-day3-1/backup/test.txt  --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud --checksum-algorithm SHA256
# echo "Get object test.txt"
# aws s3 cp s3://cameravttnew-day3-1/backup/test.txt ./test.txt  --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud
# echo "Get presigned URL"
# aws s3 presign s3://cameravttnew-day3-1/backup/test.txt   --region us-east-1 --endpoint-url https://s3-viettel.sunteco.cloud

# --no-cli-pager --recursive
# aws s3 ls s3://camera2024/backup/a  --region us-east-1 --endpoint-url http://localhost:80 --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
aws s3 ls s3://camera2024/backup/n --region us-east-1  --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
aws s3 ls s3://camera2024/backup/ --region us-east-1  --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
aws s3 ls s3://camera2024/backup --region us-east-1  --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
aws s3 ls s3://camera2024/backu --region us-east-1  --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
aws s3 ls s3://camera2024/ --region us-east-1  --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
aws s3 ls s3://camera2024 --region us-east-1  --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text

# aws s3 ls s3://camera2025  --region us-east-1 --endpoint-url http://localhost:80 --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text
# aws s3 ls s3://camera2024/backup/fi  --region us-east-1 --endpoint-url http://localhost:80 --bucket-region us-east-1 --recursive  --human-readable --summarize  --output text

# aws --region us-east-1 --endpoint-url http://localhost:80  s3api list-objects-v2 --prefix backup/ --bucket camera2024 --output text

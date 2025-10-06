#!/bin/bash
. ../.env
#!/bin/bash
# quick-sts.sh - Script ngắn gọn tạo Role, AssumeRole và lưu profile

# Cấu hình
REGION=$AWS_DEFAULT_REGION
DURATION=3600
  # Thay bằng username thực tế

bash ../iam/policy.sh

# 4. Assume Role và lấy credentials
CREDENTIALS=$(aws sts assume-role --endpoint-url $AWS_STS_ENDPOINT_URL  --role-arn $ARN --role-session-name TempSession --duration-seconds $DURATION)
echo $CREDENTIALS
ACCESS_KEYS=$(aws --endpoint $AWS_IAM_ENDPOINT_URL iam list-access-keys)
echo $A
# aws s3api put-object --bucket $S3_BUCKET --key S3_OBJECT --body data.txt --region $AWS_DEFAULT_REGION --endpoint-url $AWS_S3_ENDPOINT_URL

# # Thêm vào credentials file
# cat >> ~/.aws/credentials << EOF

# [$PROFILE_NAME]
# aws_access_key_id = $ACCESS_KEY
# aws_secret_access_key = $SECRET_KEY
# aws_session_token = $SESSION_TOKEN
# EOF

# echo "✅ Đã tạo profile '$PROFILE_NAME' trong ~/.aws/credentials"
# echo "💡 Sử dụng: aws s3 ls --profile $PROFILE_NAME"

# Dọn dẹpCCESS_KEYS
# 5. Lưu vào profile
export AWS_ACCESS_KEY_ID=$(echo $CREDENTIALS | jq -r '.Credentials.AccessKeyId')
export AWS_SECRET_ACCESS_KEY=$(echo $CREDENTIALS | jq -r '.Credentials.SecretAccessKey')
export AWS_SESSION_TOKEN=$(echo $CREDENTIALS | jq -r '.Credentials.SessionToken')
echo AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID
echo AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
echo AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN
cat > data.txt << EOF
$S3_DATA
EOF
aws s3 mb s3://$S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL
echo "✅ Đã tạo bucket $S3_BUCKET"
aws s3 cp data.txt://$S3_BUCKET/data.txt --endpoint-url $AWS_S3_ENDPOINT_URL
echo "✅ Đã upload object tới bucket $S3_BUCKET"

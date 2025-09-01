
bash source ../.env
#!/bin/bash
# quick-sts.sh - Script ngắn gọn tạo Role, AssumeRole và lưu profile

# Cấu hình
PROFILE_NAME=$S3_ACCOUNT
REGION=$AWS_DEFAULT_REGION
DURATION=3600

# Cấu hình
POLICY_NAME="TempS3Policy-$(date +%H%M%S)"
USER_NAME=admin  # Thay bằng username thực tế
PROFILE_NAME="temp-policy-access"
echo Access Key $AWS_ACCESS_KEY_ID/$AWS_SECRET_ACCESS_KEY
# curl -F file=@identity.json "http://localhost:8888/etc/iam/"
# echo 's3.configure -access_key some_access_key1 -secret_key some_secret_key1 -user iam -actions Admin -apply' | weed shell 
# echo 's3.configure' | weed shell
export AWS_ACCESS_KEY_ID=AFZCSDS3EE24X4S66134
export AWS_SECRET_ACCESS_KEY=ywjyumWexIWbmQlDmeU/54CKajmmdhDCTHiCf3vi
ACCESS_KEYSs=$(aws --endpoint $AWS_IAM_ENDPOINT_URL iam list-access-keys )
echo $ACCESS_KEYSs
# go run ./weed shell
# s3.configure -apply -user admin -access_key some_access_key1 -secret_key some_secret_key1 -actions Admin
# "AWS4-HMAC-SHA256 Credential=some_secret_key1/20250828/us-east-1/iam/aws4_request, SignedHeaders=content-type;host;x-amz-date, Signature=3dd39dad02f2c851ae35c26b81d5b40683b513a3491f98820eb7523059499882"
#!/bin/bash

# Test case for AssumeRole with credentials

set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. "$SCRIPT_DIR/../.env"

cleanup() {
    echo "Cleaning up..."
    aws s3 rb s3://$S3_BUCKET --force --endpoint-url $AWS_S3_ENDPOINT_URL || echo "Bucket cleanup failed, it might have been already removed."
    rm -f data.txt
}

trap cleanup EXIT

# 1. Setup policy
bash ../iam/policy.sh

# 2. Assume Role and get credentials
echo "Assuming role..."
CREDENTIALS=$(aws sts assume-role --endpoint-url $AWS_STS_ENDPOINT_URL --role-arn $ARN --role-session-name TempSessionTest --duration-seconds $DURATION)

if [ -z "$CREDENTIALS" ]; then
    echo "Assume role failed!" >&2
    exit 1
fi

# 3. Export temporary credentials
export AWS_ACCESS_KEY_ID=$(echo $CREDENTIALS | jq -r '.Credentials.AccessKeyId')
export AWS_SECRET_ACCESS_KEY=$(echo $CREDENTIALS | jq -r '.Credentials.SecretAccessKey')
export AWS_SESSION_TOKEN=$(echo $CREDENTIALS | jq -r '.Credentials.SessionToken')

if [ "$AWS_ACCESS_KEY_ID" == "null" ]; then
    echo "Failed to parse credentials!" >&2
    exit 1
fi

echo "Successfully assumed role and got temporary credentials."

# 4. Create a test file
cat > data.txt << EOF
$S3_DATA
EOF

# 5. Use temporary credentials to interact with S3
echo "Creating S3 bucket with assumed role..."
aws s3 mb s3://$S3_BUCKET --endpoint-url $AWS_S3_ENDPOINT_URL
echo "Bucket $S3_BUCKET created."

echo "Uploading object to bucket..."
aws s3 cp data.txt s3://$S3_BUCKET/data.txt --endpoint-url $AWS_S3_ENDPOINT_URL
echo "Object uploaded."

# 6. Verify object exists
echo "Verifying object..."
aws s3 ls s3://$S3_BUCKET/data.txt --endpoint-url $AWS_S3_ENDPOINT_URL

echo "✅ Test case for AssumeRole with credential PASSED."

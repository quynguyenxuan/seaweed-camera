export AWS_ACCESS_KEY_ID="8_tests3_accid"
export AWS_SECRET_ACCESS_KEY="-aJ20yurXb2RhF9pYwNG9shc-RKb"
# export AWS_SESSION_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NLZXkiOiI4X3Rlc3RzM19hY2NpZCIsImV4cCI6MTc1MTAwNzE1OCwiaWF0IjoxNzQ4MzMyOTQ0fQ.qOVdPAEIxmjvYqKenr-sBuab3NAA0L81wZbmS7lKveI"
export AWS_DEFAULT_REGION="us-east-1"
export AWS_S3_ADDRESSING_STYLE="path"
export AWS_SESSION_TOKEN=$(bun run test_gen_token.ts)
# set -x
STRING="Hello, this is a test dfd sdfsd string!"

PUT_URL=$(python3 test-create-presigned.py --bucket camera2024 --key test.txt --method put_object --access_key="${AWS_ACCESS_KEY_ID}" --secret_key="${AWS_SECRET_ACCESS_KEY}"  --expires=3600)
echo $PUT_URL
curl -X PUT --data "$STRING" "$PUT_URL"

GET_URL=$(python3 test-create-presigned.py --bucket camera2024 --key test.txt --method get_object --access_key=$AWS_ACCESS_KEY_ID --secret_key=$AWS_SECRET_ACCESS_KEY  --expires=3600)
echo $GET_URL
curl -X GET "${GET_URL}"

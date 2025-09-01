# * CreateAccessKey
# * ListAccessKeys
# * DeleteAccessKey

bash source ../.env
USER_RES=$(aws iam create-user --user-name $USER_NAME  --endpoint-url $AWS_IAM_ENDPOINT_URL --output json)
echo "✅ Đã tạo user: $USER_NAME"
echo $USER_RES
CREATE_ACCESS_KEY_RES=$(aws --endpoint $AWS_IAM_ENDPOINT_URL iam create-access-key --region us-east-1  --user-name $USER_NAME --output json)
echo "✅ Đã tạo access key cho user: $USER_NAME"

echo $CREATE_ACCESS_KEY_RES
LIST_ACCESS_KEY_RES=$( aws --endpoint $AWS_IAM_ENDPOINT_URL iam list-access-keys --region us-east-1  --user-name $USER_NAME --output json)
echo "✅ Danh sách access keys cho user: $USER_NAME"

echo $LIST_ACCESS_KEY_RES
KEY_ID=$(echo $CREATE_ACCESS_KEY_RES | jq -r '.AccessKey.AccessKeyId')
# DELETE_ACCESS_KEY_RES=$( aws --endpoint $AWS_IAM_ENDPOINT_URL iam delete-access-key --access-key-id $KEY_ID --region us-east-1  --user-name $USER_NAME)
# echo $DELETE_ACCESS_KEY_RES
# echo "✅ Xóa thành công access key vừa tạo cho user: $USER_NAME"

# * CreateUser
# * ListUsers
# * GetUser
# * UpdateUser
# * DeleteUser

bash source ../.env
USER_RES=$(aws iam create-user --user-name $USER_NAME  --endpoint-url $AWS_IAM_ENDPOINT_URL)
echo "✅ Đã tạo user: $USER_NAME"
echo $USER_RES

LIST_USERS_RES=$(aws --endpoint $AWS_IAM_ENDPOINT_URL iam list-users --region us-east-1  )
echo "✅ Đã lấy thành công danh sách user"

echo $LIST_USERS_RES

GET_USER_RES=$( aws --endpoint $AWS_IAM_ENDPOINT_URL iam get-user --region us-east-1  --user-name $USER_NAME)
echo "✅ Thông tin cho user: $GET_USER_RES"

DELETE_USER_RES=$( aws --endpoint $AWS_IAM_ENDPOINT_URL iam delete-user --region us-east-1  --user-name $USER_NAME)
echo "✅ Delete user thành công: $DELETE_USER_RES"

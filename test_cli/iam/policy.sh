# * CreatePolicy
# * PutUserPolicy
# * GetUserPolicy
# * DeleteUserPolicy
bash ../.env
# 1. Tạo Policy document
# chý ý format của action khớp trong source code
	# StatementActionAdmin    = "*"
	# StatementActionWrite    = "Put*"
	# StatementActionWriteAcp = "PutBucketAcl"
	# StatementActionRead     = "Get*"
	# StatementActionReadAcp  = "GetBucketAcl"
	# StatementActionList     = "List*"
	# StatementActionTagging  = "Tagging*"
	# StatementActionDelete   = "DeleteBucket*"
cat > policy.json << EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "s3:Get*",
      "s3:List*",
      "s3:Put*"
      ],
    "Resource": [
        "arn:aws:s3:::*"
    ]
  }]
}
EOF

USER_CREATED=$(aws iam create-user --user-name $USER_NAME  --endpoint-url $AWS_IAM_ENDPOINT_URL)
echo "✅ Đã tạo user: $USER_CREATED"
# 2. Tạo Policy (lấy ARN để sử dụng nếu cần)
USER_POLICY=$(aws iam create-policy --policy-name $POLICY_NAME --policy-document file://policy.json --endpoint-url $AWS_IAM_ENDPOINT_URL)
echo "✅ Đã tạo policy: $USER_POLICY"

ARN=$(echo $USER_POLICY | jq -r '.Policy.Arn')
# 3. Gán policy trực tiếp cho user (PutUserPolicy - inline policy)
PUT_USER_POLICY=$(aws iam put-user-policy --user-name $USER_NAME --policy-name $POLICY_NAME --policy-document file://policy.json  --endpoint-url $AWS_IAM_ENDPOINT_URL)
echo "✅ Đã gán policy cho user: $USER_NAME"


GET_USER_POLICY=$(aws iam get-user-policy --user-name $USER_NAME --policy-name $POLICY_NAME --endpoint-url $AWS_IAM_ENDPOINT_URL)
echo "💡 Thông tin  Policy đã được gán cho user $USER_NAME"
echo $GET_USER_POLICY
# DELETE_USER_POLICY=$(aws iam delete-user-policy --user-name $USER_NAME --policy-name $POLICY_NAME --endpoint-url $AWS_IAM_ENDPOINT_URL)
# echo "💡 Đã xóa policy thành công
# echo $DELETE_USER_POLICY

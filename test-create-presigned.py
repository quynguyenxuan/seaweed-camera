import argparse
import boto3
from botocore.exceptions import ClientError
from botocore.client import Config

def generate_presigned_url(s3_client, client_method, method_parameters, expires_in):
    """
    Generate a presigned Amazon S3 URL that can be used to perform an action.

    :param s3_client: A Boto3 Amazon S3 client.
    :param client_method: The name of the client method that the URL performs.
    :param method_parameters: The parameters of the specified client method.
    :param expires_in: The number of seconds the presigned URL is valid for.
    :return: The presigned URL.
    """
    try:
        url = s3_client.generate_presigned_url(
            ClientMethod=client_method,
            Params=method_parameters,
            ExpiresIn=expires_in
        )
    except ClientError:
        print(f"Couldn't get a presigned URL for client method '{client_method}'.")
        raise
    return url

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--bucket", help="The name of the bucket.")
    parser.add_argument(
        "--key", help="The key (path and filename) in the S3 bucket.",
    )
    parser.add_argument(
        "--method", help="The method in the S3 bucket.",
    )
    parser.add_argument(
        "--access_key", help="AccessKey.",
    )
    parser.add_argument(
        "--secret_key", help="SecretKey.",
    )
    parser.add_argument(
        "--session_token", help="SessionToken.",
    )
    parser.add_argument(
        "--expires",type=int, help="SessionToken.",
    )
    args = parser.parse_args()


    # By default, this will use credentials from ~/.aws/credentials
    # s3_client = boto3.client("s3")
    s3_client = boto3.client(
            "s3",
            endpoint_url="http://localhost:8333",
            aws_access_key_id=args.access_key,  # Thay bằng Access Key ID thực
            aws_secret_access_key=args.secret_key,  # Thay bằng Secret Access Key thực
            aws_session_token=args.session_token,
            config=Config(signature_version="s3v4", s3={'addressing_style': 'path'})
        )

    # The presigned URL is specified to expire in 1000 seconds
    url = generate_presigned_url(
        s3_client,
        args.method,
        {"Bucket": args.bucket, "Key": args.key},
        args.expires
    )
    print(f"{url}")

if __name__ == "__main__":
    main()

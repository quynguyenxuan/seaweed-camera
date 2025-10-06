package copying_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func createLargeFile(size int64) *bytes.Reader {
	data := make([]byte, size)
	return bytes.NewReader(data)
}

func multipartUpload(t *testing.T, endpointURL, accessKey, secretKey, bucketName, fileKey string, numParts int, partSize int64) {
	// Tạo config cho S3 client
	// cfg, err := config.LoadDefaultConfig(context.TODO(),
	// 	config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	// 	config.WithRegion("us-east-1"), // Region có thể thay đổi
	// )
	// if err != nil {
	// 	log.Fatalf("Unable to load SDK config: %v", err)
	// }

	// // Nếu có endpoint URL (cho S3-compatible service), ghi đè endpoint
	// if endpointURL != "" {
	// 	cfg.BaseEndpoint = aws.String(endpointURL)
	// }

	// client := s3.NewFromConfig(cfg)
	client := getS3Client(t)
	// Khởi tạo multipart upload
	resp, err := client.CreateMultipartUpload(context.TODO(), &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		log.Fatalf("Unable to create multipart upload: %v", err)
	}
	uploadID := resp.UploadId
	fmt.Printf("Created multipart upload with ID: %s\n", *uploadID)

	// Tạo file giả lập
	totalSize := partSize * int64(numParts)
	fileReader := createLargeFile(totalSize)

	var completedParts []types.CompletedPart

	// Upload từng phần
	for i := 0; i < numParts; i++ {
		fmt.Printf("Uploading part %d/%d\n", i+1, numParts)

		partNum := int32(i + 1)
		data := make([]byte, partSize)
		_, err := io.ReadFull(fileReader, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Fatalf("Error reading part %d: %v", i+1, err)
		}

		result, err := client.UploadPart(context.TODO(), &s3.UploadPartInput{
			Bucket:     aws.String(bucketName),
			Key:        aws.String(fileKey),
			PartNumber: &partNum,
			UploadId:   uploadID,
			Body:       bytes.NewReader(data),
		})
		if err != nil {
			log.Printf("Error uploading part %d: %v", i+1, err)
			// Hủy upload nếu lỗi
			client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
				Bucket:   aws.String(bucketName),
				Key:      aws.String(fileKey),
				UploadId: uploadID,
			})
			os.Exit(1)
		}

		completedParts = append(completedParts, types.CompletedPart{
			ETag:       result.ETag,
			PartNumber: &partNum,
		})
	}

	// Hoàn tất multipart upload
	_, err = client.CompleteMultipartUpload(context.TODO(), &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucketName),
		Key:      aws.String(fileKey),
		UploadId: uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		log.Printf("Error completing multipart upload: %v", err)
		client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucketName),
			Key:      aws.String(fileKey),
			UploadId: uploadID,
		})
		os.Exit(1)
	}

	fmt.Println("Multipart upload completed successfully!")
}

func TestMultiPartUpload(t *testing.T) {
	endpointURL := "http://127.0.0.1:8333"            //os.Args[1] // ví dụ: https://s3.us-west-2.amazonaws.com
	accessKey := "RB8OOZ094Z1NBJP2GG57"               //os.Args[2]
	secretKey := "l8G589CJfVaFS36tpsOCFvz3YD+EAv/h34" //os.Args[3]
	bucketName := "sdfhsjd"                           //os.Args[4]
	fileKey := "large-file.bin"                       // có thể truyền thêm nếu cần
	numParts := 50
	partSize := int64(5 * 1024 * 1024) // 5MB

	multipartUpload(t, endpointURL, accessKey, secretKey, bucketName, fileKey, numParts, partSize)
}

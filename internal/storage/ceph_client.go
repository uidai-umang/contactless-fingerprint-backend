package storage

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps the S3-compatible client used to talk to CEPH RGW
var Client *s3.Client

// Bucket is the target bucket name read from environment
var Bucket string

// Connect initializes the S3 client pointed at CEPH's RGW endpoint.
// CEPH RGW exposes an S3-compatible API, so aws-sdk-go-v2 works unmodified.
func Connect() error {
	endpoint := os.Getenv("CEPH_ENDPOINT")
	accessKey := os.Getenv("CEPH_ACCESS_KEY")
	secretKey := os.Getenv("CEPH_SECRET_KEY")
	region := os.Getenv("CEPH_REGION")
	Bucket = os.Getenv("CEPH_BUCKET")

	if endpoint == "" || accessKey == "" || secretKey == "" || Bucket == "" {
		return fmt.Errorf("missing required CEPH environment variables")
	}

	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to load CEPH config: %w", err)
	}

	Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(endpoint)
	})

	return nil
}

// UploadObject uploads raw bytes to the given object key inside the configured bucket
func UploadObject(ctx context.Context, objectKey string, data []byte) error {
	_, err := Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(Bucket),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(data),
	})
	return err
}

// TestConnection verifies connectivity by attempting to head the bucket.
// Used at startup to fail fast with a clear error if CEPH is unreachable.
func TestConnection(ctx context.Context) error {
	_, err := Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(Bucket),
	})
	return err
}

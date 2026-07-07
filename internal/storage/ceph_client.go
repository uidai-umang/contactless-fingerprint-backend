package storage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

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

// SaveObjectLocally is a temporary demo fallback while CEPH connectivity is
// unavailable. It stores the same generated object key under uploads/ on the
// backend filesystem and must be removed once CEPH uploads are restored.
func SaveObjectLocally(ctx context.Context, objectKey string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	localPath, err := localUploadPath(objectKey)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create local upload directory: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.WriteFile(localPath, data, 0600); err != nil {
		return fmt.Errorf("failed to save image locally: %w", err)
	}

	return nil
}

func localUploadPath(objectKey string) (string, error) {
	cleaned := path.Clean(strings.TrimSpace(objectKey))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("invalid object key")
	}

	parts := strings.Split(cleaned, "/")
	if len(parts) > 0 && parts[0] == "sitaa-clf" {
		parts = parts[1:]
	}
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid object key path")
	}

	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid object key path segment")
		}
	}

	return filepath.Join(append([]string{"uploads"}, parts...)...), nil
}

// TestConnection verifies connectivity by attempting to head the bucket.
// Used at startup to fail fast with a clear error if CEPH is unreachable.
func TestConnection(ctx context.Context) error {
	_, err := Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(Bucket),
	})
	return err
}

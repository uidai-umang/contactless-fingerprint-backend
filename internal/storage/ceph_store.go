package storage

import "context"

// CephStore implements ImageStore using the CEPH S3-compatible client.
type CephStore struct{}

func NewCephStore() *CephStore {
	return &CephStore{}
}

func (c *CephStore) Save(ctx context.Context, objectKey string, data []byte) error {
	return UploadObject(ctx, objectKey, data)
}

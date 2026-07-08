package storage

import "context"

// LocalStore implements ImageStore using the local filesystem.
// TEMPORARY — used only while CEPH network access is unavailable.
type LocalStore struct{}

func NewLocalStore() *LocalStore {
	return &LocalStore{}
}

func (l *LocalStore) Save(ctx context.Context, objectKey string, data []byte) error {
	return SaveObjectLocally(ctx, objectKey, data)
}

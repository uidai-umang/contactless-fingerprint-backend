package storage

import "context"

// ImageStore is the contract CaptureService depends on.
// It has no idea whether images end up in CEPH, local disk,
// or anywhere else — that decision lives entirely behind this interface.
type ImageStore interface {
	Save(ctx context.Context, objectKey string, data []byte) error
}

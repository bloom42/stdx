package storage

import (
	"context"
	"io"
)

type Storage interface {
	BasePath() string
	CopyObject(ctx context.Context, from, to string) error
	DeleteObject(ctx context.Context, key string) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	GetObjectSize(ctx context.Context, key string) (int64, error)
	// GetPresignedUploadUrl(ctx context.Context, key string, size uint64) (string, error)
	PutObject(ctx context.Context, key, contentType string, size int64, object io.Reader) error
	DeleteObjectsWithPrefix(ctx context.Context, prefix string) (err error)
}

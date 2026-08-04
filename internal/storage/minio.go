package storage

import (
	"context"
	"io"
)

type MinIOClient interface {
	EnsureBucket(ctx context.Context, bucketName string) error
	PutObject(ctx context.Context, bucket, objectKey string, reader io.Reader, size int64) error
	GetObject(ctx context.Context, bucket, objectKey string) (reader io.ReadCloser, size int64, err error)
	DeleteObject(ctx context.Context, bucket, objectKey string) error
	DeleteObjects(ctx context.Context, bucket string, objectKeys []string) error
}

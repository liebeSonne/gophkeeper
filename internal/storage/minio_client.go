package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

type minioClient struct {
	client *minio.Client
	logger intlogger.Logger
}

func NewMinIOClient(
	endpoint, accessKey, secretKey string, secure bool,
	logger intlogger.Logger,
) (MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}

	return &minioClient{
		client: client,
		logger: logger,
	}, nil
}

func (c *minioClient) EnsureBucket(ctx context.Context, bucketName string) error {
	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}

	if !exists {
		err = c.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *minioClient) PutObject(ctx context.Context, bucket, objectKey string, reader io.Reader, size int64) error {
	_, err := c.client.PutObject(ctx, bucket, objectKey, reader, size, minio.PutObjectOptions{})
	return err
}

func (c *minioClient) GetObject(ctx context.Context, bucket, objectKey string) (reader io.ReadCloser, size int64, err error) {
	obj, err := c.client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}

	info, err := c.client.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		errClose := obj.Close()
		if errClose != nil {
			c.logger.Warn("error on object close", "err", errClose)
		}
		return nil, 0, err
	}

	return obj, info.Size, nil
}

func (c *minioClient) DeleteObject(ctx context.Context, bucket, objectKey string) error {
	return c.client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{})
}

func (c *minioClient) DeleteObjects(ctx context.Context, bucket string, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	objCh := make(chan minio.ObjectInfo, len(objectKeys))
	for _, key := range objectKeys {
		objCh <- minio.ObjectInfo{Key: key}
	}
	close(objCh)

	errCh := c.client.RemoveObjects(ctx, bucket, objCh, minio.RemoveObjectsOptions{})
	for err := range errCh {
		return err.Err
	}

	return nil
}

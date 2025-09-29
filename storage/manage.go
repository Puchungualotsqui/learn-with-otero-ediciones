package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/kurin/blazer/b2"
)

type B2Storage struct {
	Client  *b2.Client
	Bucket  *b2.Bucket
	BaseUrl string
}

func Init(ctx context.Context, accountId, appKey, bucketName, baseUrl string) (*B2Storage, error) {
	client, err := b2.NewClient(ctx, accountId, appKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create b2 client: %w", err)
	}

	bucket, err := client.Bucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &B2Storage{Client: client, Bucket: bucket, BaseUrl: baseUrl}, nil
}

func (s *B2Storage) UploadFile(ctx context.Context, key string, r io.Reader) (string, error) {
	obj := s.Bucket.Object(key)
	w := obj.NewWriter(ctx)

	if _, err := io.Copy(w, r); err != nil {
		return "", fmt.Errorf("failed to write object: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.BaseUrl, key), nil
}

func (s *B2Storage) DownloadFile(ctx context.Context, key string, w io.Writer) error {
	obj := s.Bucket.Object(key)
	r := obj.NewReader(ctx)
	defer r.Close()

	_, err := io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("failed to read the object: %w", err)
	}

	return nil
}

func (s *B2Storage) DeleteFile(ctx context.Context, key string) error {
	obj := s.Bucket.Object(key)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kurin/blazer/b2"
)

type B2Storage struct {
	PublicClient  *b2.Client
	PrivateClient *b2.Client
	PublicBucket  *b2.Bucket
	PrivateBucket *b2.Bucket
	BaseUrl       string
}

type FileInfo struct {
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
}

func Init() (*B2Storage, error) {
	err := godotenv.Load(".venv") // use ".env" if you renamed it
	if err != nil {
		log.Fatal("Error loading .venv file")
	}

	keyIdPrivate := os.Getenv("B2_KEY_ID_PRIVATE")
	appKeyPrivate := os.Getenv("B2_APP_KEY_PRIVATE")
	bucketPrivateName := os.Getenv("B2_BUCKET_PRIVATE")
	keyIdPublic := os.Getenv("B2_KEY_ID_PUBLIC")
	appKeyPublic := os.Getenv("B2_APP_KEY_PUBLIC")
	bucketPublicName := os.Getenv("B2_BUCKET_PUBLIC")

	baseUrl := os.Getenv("B2_BASE_URL")

	if keyIdPrivate == "" || appKeyPrivate == "" || bucketPublicName == "" || keyIdPublic == "" || appKeyPublic == "" || bucketPrivateName == "" || baseUrl == "" {
		log.Fatal("missing B2 env vars")
	}

	publicClient, err := b2.NewClient(context.Background(), keyIdPublic, appKeyPublic)
	if err != nil {
		return nil, fmt.Errorf("failed to create b2 client: %w", err)
	}

	publicBucket, err := publicClient.Bucket(context.Background(), bucketPublicName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	privateClient, err := b2.NewClient(context.Background(), keyIdPrivate, appKeyPrivate)
	if err != nil {
		return nil, fmt.Errorf("failed to create b2 client: %w", err)
	}

	privateBucket, err := privateClient.Bucket(context.Background(), bucketPrivateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &B2Storage{PublicClient: publicClient, PrivateClient: privateClient, PublicBucket: publicBucket, PrivateBucket: privateBucket, BaseUrl: baseUrl}, nil
}

func (s *B2Storage) UploadFile(ctx context.Context, key string, r io.Reader) (string, error) {
	obj := s.PublicBucket.Object(key)
	w := obj.NewWriter(ctx)

	if _, err := io.Copy(w, r); err != nil {
		return "", fmt.Errorf("failed to write object: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	return fmt.Sprintf("https://%s/file/%s/%s", s.BaseUrl, s.PublicBucket.Name(), key), nil
}

func (s *B2Storage) DownloadFile(ctx context.Context, key string, w io.Writer) error {
	obj := s.PublicBucket.Object(key)
	r := obj.NewReader(ctx)
	defer r.Close()

	_, err := io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("failed to read the object: %w", err)
	}

	return nil
}

func (s *B2Storage) DeleteFile(ctx context.Context, path string) error {
	// Convert friendly URL → key if needed
	friendlyPrefix := fmt.Sprintf("https://%s/file/%s/", s.BaseUrl, s.PrivateBucket.Name())
	key := strings.ReplaceAll(path, friendlyPrefix, "")

	obj := s.PublicBucket.Object(key)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object %q: %w", key, err)
	}
	return nil
}

func (s *B2Storage) ListFiles(prefix string) ([]FileInfo, error) {
	var results []FileInfo

	iter := s.PrivateBucket.List(context.Background(), b2.ListPrefix(prefix))
	for iter.Next() {
		obj := iter.Object()
		name := obj.Name()

		// Skip folders
		if strings.HasSuffix(name, "/") {
			continue
		}

		// Build public URL (same format as UploadFile)
		url := fmt.Sprintf("https://%s/file/%s/%s", s.BaseUrl, s.PrivateBucket.Name(), name)

		results = append(results, FileInfo{
			FileName:    name,
			DownloadURL: url,
		})
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list files with prefix %q: %w", prefix, err)
	}

	return results, nil
}

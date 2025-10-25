package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/kurin/blazer/b2"
)

type FileInfo struct {
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
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

func (s *B2Storage) UploadPrivateFile(ctx context.Context, key string, r io.Reader) (string, error) {
	obj := s.PrivateBucket.Object(key)
	w := obj.NewWriter(ctx)

	if _, err := io.Copy(w, r); err != nil {
		return "", fmt.Errorf("failed to write object: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	return fmt.Sprintf("https://%s/file/%s/%s", s.BaseUrl, s.PrivateBucket.Name(), key), nil
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

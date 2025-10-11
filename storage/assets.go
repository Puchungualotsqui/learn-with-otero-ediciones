package storage

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"
)

func (s *B2Storage) GetTemporaryLink(ctx context.Context, key string, duration time.Duration) (string, error) {
	if s.PrivateBucket == nil {
		return "", fmt.Errorf("bucket not initialized")
	}

	token, err := s.PrivateBucket.AuthToken(ctx, key, duration)
	if err != nil {
		return "", fmt.Errorf("failed to get auth token: %w", err)
	}

	safeKey := path.Clean(key)
	safeKey = strings.TrimPrefix(safeKey, "/")
	safeKey = url.PathEscape(safeKey)

	// Build temporary URL
	tempURL := fmt.Sprintf("https://%s/file/%s/%s?Authorization=%s",
		s.BaseUrl,
		s.PrivateBucket.Name(),
		safeKey,
		token,
	)

	return tempURL, nil
}

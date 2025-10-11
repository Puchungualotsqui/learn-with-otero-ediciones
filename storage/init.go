package storage

import (
	"context"
	"fmt"
	"log"
	"os"

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

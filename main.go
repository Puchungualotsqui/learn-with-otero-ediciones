package main

import (
	"context"
	"fmt"
	"frontend/database"
	"frontend/internal/router"
	"frontend/storage"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".venv") // use ".env" if you renamed it
	if err != nil {
		log.Fatal("Error loading .venv file")
	}

	keyId := os.Getenv("B2_KEY_ID")
	appKey := os.Getenv("B2_APP_KEY")
	bucketName := os.Getenv("B2_BUCKET")
	baseUrl := os.Getenv("B2_BASE_URL")

	if keyId == "" || appKey == "" || bucketName == "" {
		log.Fatal("missing B2 env vars")
	}

	ctx := context.Background()
	storage, err := storage.Init(ctx, keyId, appKey, bucketName, baseUrl)
	if err != nil {
		log.Fatalf("Error initializing storage: %v", err)
	}
	fmt.Println("B2 Storage ready:", storage.BaseUrl)

	store, err := database.Init("data/school.db")
	if err != nil {
		log.Fatal("failed to init database:", err)
	}
	defer store.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		router.Router(store, storage, w, r)
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("🚀 Server running at http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

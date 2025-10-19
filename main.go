package main

import (
	"fmt"
	"frontend/database"
	"frontend/internal/router"
	"frontend/storage"
	"log"
	"net/http"
)

func main() {
	storage, err := storage.Init()
	if err != nil {
		log.Fatalf("Error initializing storage: %v", err)
	}
	fmt.Println("B2 Storage ready:", storage.BaseUrl)

	store, err := database.Init("data/school.db")
	if err != nil {
		log.Fatal("failed to init database:", err)
	}
	defer store.Close()

	database.RefreshAssets(store, storage)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		router.Router(store, storage, w, r)
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("🚀 Server running at http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

package main

import (
	"frontend/database"
	"frontend/internal"
	"log"
	"net/http"
)

func main() {
	store, err := database.Init("data/school.db")
	if err != nil {
		log.Fatal("failed to init database:", err)
	}
	defer store.Close()

	http.HandleFunc("/", internal.Router)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("🚀 Server running at http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

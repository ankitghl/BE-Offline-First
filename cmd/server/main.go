package main

import (
	"Offline-First/internal/db"
	"log"
	"net/http"
	"os"
)

func main() {
	connectPostgres()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Println("api listening on :8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func connectPostgres() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set!")
	}

	conn, err := db.NewPostgres(dsn)
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	log.Println("database connected")

}

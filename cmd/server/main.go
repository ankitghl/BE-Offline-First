package main

import (
	"Offline-First/internal/db"
	"log"
	"os"
)

func main() {
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

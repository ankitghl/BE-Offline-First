package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"Offline-First/internal/db"
	httpapi "Offline-First/internal/http"
	"Offline-First/internal/http/handler"
	"Offline-First/internal/http/middleware"
	"Offline-First/internal/repository/postgres"
)

func main() {
	// 1️⃣ Read env
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set!")
	}

	// 2️⃣ Connect DB (keep it alive)
	dbConn := connectPostgresWithRetry(dsn)
	defer dbConn.Close()

	log.Println("database connected")

	// 3️⃣ Create repository
	itemRepo := postgres.NewItemRepository(dbConn)

	// 4️⃣ Create handlers
	itemHandler := handler.NewItemHandler(itemRepo)

	// 5️⃣ Create router
	router := httpapi.NewRouter(itemHandler)

	// 🔐 wrap router with auth
	securedRouter := middleware.Auth(router)

	// 6️⃣ Add health endpoint
	routerWithHealth := addHealth(securedRouter)

	// 7️⃣ Start server
	log.Println("api listening on :8081")
	if err := http.ListenAndServe(":8081", routerWithHealth); err != nil {
		log.Fatal(err)
	}
}

func connectPostgresWithRetry(dsn string) *sql.DB {
	var dbConn *sql.DB
	var err error

	for i := 1; i <= 10; i++ {
		dbConn, err = db.NewPostgres(dsn)
		if err == nil {
			log.Println("database connected")
			return dbConn
		}

		log.Printf("waiting for database (%d/10): %v", i, err)
		time.Sleep(2 * time.Second)
	}

	log.Fatal("could not connect to database after retries")
	return nil
}

func addHealth(next http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mux.Handle("/", next)
	return mux
}

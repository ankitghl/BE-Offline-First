package http

import (
	"Offline-First/internal/http/handler"
	"Offline-First/internal/http/middleware"
	"net/http"
)

func NewRouter(itemHandler *handler.ItemHandler) http.Handler {
	mux := http.NewServeMux()

	// /items (create, list)
	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			middleware.MutationMiddleware(http.HandlerFunc(itemHandler.Create)).ServeHTTP(w, r)
		case http.MethodGet:
			itemHandler.List(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// /items/{id} (update, delete)
	mux.HandleFunc("/items/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			middleware.MutationMiddleware(http.HandlerFunc(itemHandler.Update)).ServeHTTP(w, r)
		case http.MethodDelete:
			middleware.MutationMiddleware(http.HandlerFunc(itemHandler.Delete)).ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)

		}
	})

	// /changes (sync API)
	mux.HandleFunc("/changes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		itemHandler.GetChanges(w, r)
	})

	return mux
}

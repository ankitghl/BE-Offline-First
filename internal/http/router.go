package http

import (
	"Offline-First/internal/http/handler"
	"net/http"
)

func NewRouter(itemHandler *handler.ItemHandler) http.Handler {
	mux := http.NewServeMux()

	// /items (create, list)
	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			itemHandler.Create(w, r)
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
			itemHandler.Update(w, r)
		case http.MethodDelete:
			itemHandler.Delete(w, r)
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

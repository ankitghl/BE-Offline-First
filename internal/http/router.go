package http

import (
	"Offline-First/internal/http/handler"
	"net/http"
)

func NewRouter(itemHandler *handler.ItemHandler) http.Handler {
	mux := http.NewServeMux()

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
	return mux
}

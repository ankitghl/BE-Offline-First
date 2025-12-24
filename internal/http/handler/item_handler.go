package handler

import (
	domain "Offline-First/internal/domain/model"
	"Offline-First/internal/repository"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type ItemHandler struct {
	repo repository.ItemRepository
}

func NewItemHandler(repo repository.ItemRepository) *ItemHandler {
	return &ItemHandler{repo: repo}
}

type CreateItemRequest struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

type ItemResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Version   int    `json:"version"`
	Deleted   bool   `json:"deleted"`
	UpdatedAt string `json:"updated_at"`
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	item := &domain.Item{
		ID:      req.ID,
		UserID:  req.UserID,
		Type:    req.Type,
		Title:   req.Title,
		Content: req.Content,
		Version: 1,
		Deleted: false,
	}

	if err := h.repo.Create(r.Context(), item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("user_id")
	if userId == "" {
		http.Error(w, "user_id required!", http.StatusBadRequest)
		return
	}

	items, err := h.repo.ListByOwner(r.Context(), userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]ItemResponse, 0, len(items))

	for _, item := range items {
		resp = append(resp, ItemResponse{
			ID:        item.ID,
			UserID:    item.UserID,
			Type:      item.Type,
			Title:     item.Title,
			Content:   item.Content,
			Version:   item.Version,
			Deleted:   item.Deleted,
			UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/items/")
	if id == "" {
		http.Error(w, "id required!", http.StatusBadRequest)
	}

	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	item := &domain.Item{
		ID:      id,
		Title:   req.Title,
		Content: req.Content,
		Type:    req.Type,
	}

	updated, err := h.repo.Update(r.Context(), item)
	if err == sql.ErrNoRows {
		http.Error(w, "No items found!", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/items/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	deletedItem, err := h.repo.SoftDelete(r.Context(), id)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deletedItem)
}

package handler

import (
	domain "Offline-First/internal/domain/model"
	"Offline-First/internal/http/middleware"
	"Offline-First/internal/repository"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
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
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Version int    `json:"version"`
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

type ChangeResponse struct {
	LatestVersion int            `json:"latest_version"`
	Items         []ItemResponse `json:"items"`
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorised!", http.StatusUnauthorized)
		return

	}

	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json!", http.StatusBadRequest)
		return
	}

	item := &domain.Item{
		ID:      req.ID,
		UserID:  userID,
		Type:    req.Type,
		Title:   req.Title,
		Content: req.Content,
		Version: 1,
		Deleted: false,
	}

	created, err := h.repo.Create(r.Context(), item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ItemResponse{
		ID:        created.ID,
		UserID:    created.UserID,
		Type:      created.Type,
		Title:     created.Title,
		Content:   created.Content,
		Version:   created.Version,
		Deleted:   created.Deleted,
		UpdatedAt: created.UpdatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := h.repo.ListByUser(r.Context(), userID)
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
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/items/")
	if id == "" {
		http.Error(w, "id required!", http.StatusBadRequest)
		return
	}

	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	item := &domain.Item{
		ID:      id,
		UserID:  userID,
		Type:    req.Type,
		Title:   req.Title,
		Content: req.Content,
		Version: req.Version,
	}

	updated, err := h.repo.Update(r.Context(), item)
	if err != nil {
		switch e := err.(type) {
		case *domain.ConflictError:
			conflictedItem := ItemResponse{
				ID:        e.ServerItem.ID,
				UserID:    e.ServerItem.UserID,
				Type:      e.ServerItem.Type,
				Title:     e.ServerItem.Title,
				Content:   e.ServerItem.Content,
				Version:   e.ServerItem.Version,
				Deleted:   e.ServerItem.Deleted,
				UpdatedAt: e.ServerItem.UpdatedAt.Format(time.RFC3339),
			}

			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":       "version_conflict",
				"server_item": conflictedItem,
			})
			return

		case error:
			if err == domain.ErrNotFound {
				http.Error(w, "not found!", http.StatusNotFound)
				return
			}
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := ItemResponse{
		ID:        updated.ID,
		UserID:    updated.UserID,
		Type:      updated.Type,
		Title:     updated.Title,
		Content:   updated.Content,
		Version:   updated.Version,
		Deleted:   updated.Deleted,
		UpdatedAt: updated.UpdatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/items/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	versionStr := r.URL.Query().Get("version")
	if versionStr == "" {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		http.Error(w, "invalid version", http.StatusBadRequest)
		return
	}

	deletedItem, err := h.repo.SoftDelete(r.Context(), id, userID, version)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ItemResponse{
		ID:        deletedItem.ID,
		UserID:    deletedItem.UserID,
		Type:      deletedItem.Type,
		Title:     deletedItem.Title,
		Content:   deletedItem.Content,
		Version:   deletedItem.Version,
		Deleted:   deletedItem.Deleted,
		UpdatedAt: deletedItem.UpdatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ItemHandler) GetChanges(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sinceStr := r.URL.Query().Get("since_version")
	if sinceStr == "" {
		http.Error(w, "since_version is required!", http.StatusBadRequest)
		return
	}

	sinceVersion, err := strconv.Atoi(sinceStr)
	if err != nil {
		http.Error(w, "invalid since_version!", http.StatusBadRequest)
		return
	}

	items, latestVersion, err := h.repo.GetChanges(r.Context(), userID, sinceVersion)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	itemResponse := make([]ItemResponse, 0, len(items))

	for _, item := range items {
		itemResponse = append(itemResponse, ItemResponse{
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

	resp := ChangeResponse{
		LatestVersion: latestVersion,
		Items:         itemResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

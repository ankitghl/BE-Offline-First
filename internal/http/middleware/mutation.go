package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type mutationIDKeyType struct{}

var mutationIDKey = mutationIDKeyType{}

func MutationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutationIDStr := r.Header.Get("X-MUTATION-ID")
		if mutationIDStr == "" {
			http.Error(w, "X-MUTATION-ID header is required", http.StatusBadRequest)
			return
		}

		mutationID, err := uuid.Parse(mutationIDStr)
		if err != nil {
			http.Error(w, "invalid X-MUTATION-ID", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), mutationIDKey, mutationID.String())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MutationIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(mutationIDKey).(string)
	return id, ok
}

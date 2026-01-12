package middleware

import (
	"context"
	"log"
)

func LogWithContext(ctx context.Context, msg string, kv ...any) {
	fields := []any{}

	if userID, ok := UserIDFromContext(ctx); ok {
		fields = append(fields, "user_id", userID)
	}

	if mutationID, ok := MutationIDFromContext(ctx); ok {
		fields = append(fields, "mutation_id", mutationID)
	}

	fields = append(fields, kv...)

	log.Println(append([]any{msg}, fields...)...)
}

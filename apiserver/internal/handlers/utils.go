package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type contextKey string

const contextSubjectKey contextKey = "sub"

func userIDFromContext(ctx context.Context) (int, error) {
	value := ctx.Value(contextSubjectKey)
	switch subject := value.(type) {
	case int:
		if subject < 1 {
			return 0, errors.New("invalid subject")
		}
		return subject, nil
	case int64:
		if subject < 1 {
			return 0, errors.New("invalid subject")
		}
		return int(subject), nil
	case float64:
		if subject < 1 {
			return 0, errors.New("invalid subject")
		}
		return int(subject), nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(subject))
		if err != nil || parsed < 1 {
			return 0, errors.New("invalid subject")
		}
		return parsed, nil
	default:
		return 0, errors.New("missing subject")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

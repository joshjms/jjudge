package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/store"
)

// ManagerHandler handles manager-specific views.
type ManagerHandler struct {
	problemService *services.ProblemService
	contestService *services.ContestService
	userService    *services.UserService
}

func NewManagerHandler(
	problemService *services.ProblemService,
	contestService *services.ContestService,
	userService *services.UserService,
) *ManagerHandler {
	return &ManagerHandler{
		problemService: problemService,
		contestService: contestService,
		userService:    userService,
	}
}

// ManagerRouter registers manager-specific routes. All require manager or admin.
func ManagerRouter(
	r chi.Router,
	problemService *services.ProblemService,
	contestService *services.ContestService,
	userService *services.UserService,
	authMiddleware func(http.Handler) http.Handler,
) {
	h := NewManagerHandler(problemService, contestService, userService)

	requireAdminOrManager := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := userIDFromContext(r.Context())
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			user, err := userService.GetByID(r.Context(), userID)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeError(w, http.StatusUnauthorized, "unauthorized")
					return
				}
				writeError(w, http.StatusInternalServerError, "failed to load user")
				return
			}
			role := strings.ToLower(user.Role)
			if role != adminRole && role != managerRole {
				writeError(w, http.StatusForbidden, "manager or admin access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	if authMiddleware != nil {
		r.With(authMiddleware, requireAdminOrManager).Get("/contests", h.ListMyContests)
		r.With(authMiddleware, requireAdminOrManager).Get("/problems", h.ListMyProblems)
	} else {
		r.With(requireAdminOrManager).Get("/contests", h.ListMyContests)
		r.With(requireAdminOrManager).Get("/problems", h.ListMyProblems)
	}
}

func (h *ManagerHandler) ListMyContests(w http.ResponseWriter, r *http.Request) {
	_, limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	contests, total, err := h.contestService.ListContestsByOwner(r.Context(), userID, offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list contests")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": contests,
		"total": total,
	})
}

func (h *ManagerHandler) ListMyProblems(w http.ResponseWriter, r *http.Request) {
	_, limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	problems, total, err := h.problemService.ListByCreator(r.Context(), userID, offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list problems")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": problems,
		"total": total,
	})
}

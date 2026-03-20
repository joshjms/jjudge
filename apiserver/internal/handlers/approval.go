package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/store"
)

// ApprovalHandler handles admin approval workflows for problems and contests.
type ApprovalHandler struct {
	problemService *services.ProblemService
	contestService *services.ContestService
	userService    *services.UserService
}

func NewApprovalHandler(
	problemService *services.ProblemService,
	contestService *services.ContestService,
	userService *services.UserService,
) *ApprovalHandler {
	return &ApprovalHandler{
		problemService: problemService,
		contestService: contestService,
		userService:    userService,
	}
}

// ApprovalRouter registers approval routes. All routes are admin-only.
func ApprovalRouter(
	r chi.Router,
	problemService *services.ProblemService,
	contestService *services.ContestService,
	userService *services.UserService,
	authMiddleware func(http.Handler) http.Handler,
) {
	h := NewApprovalHandler(problemService, contestService, userService)

	requireAdmin := func(next http.Handler) http.Handler {
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
			if !strings.EqualFold(user.Role, adminRole) {
				writeError(w, http.StatusForbidden, "admin access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	if authMiddleware != nil {
		r.With(authMiddleware, requireAdmin).Get("/problems", h.ListPendingProblems)
		r.With(authMiddleware, requireAdmin).Post("/problems/{problemID}/approve", h.ApproveProblem)
		r.With(authMiddleware, requireAdmin).Post("/problems/{problemID}/reject", h.RejectProblem)
		r.With(authMiddleware, requireAdmin).Get("/contests", h.ListPendingContests)
		r.With(authMiddleware, requireAdmin).Post("/contests/{contestID}/approve", h.ApproveContest)
		r.With(authMiddleware, requireAdmin).Post("/contests/{contestID}/reject", h.RejectContest)
	} else {
		r.With(requireAdmin).Get("/problems", h.ListPendingProblems)
		r.With(requireAdmin).Post("/problems/{problemID}/approve", h.ApproveProblem)
		r.With(requireAdmin).Post("/problems/{problemID}/reject", h.RejectProblem)
		r.With(requireAdmin).Get("/contests", h.ListPendingContests)
		r.With(requireAdmin).Post("/contests/{contestID}/approve", h.ApproveContest)
		r.With(requireAdmin).Post("/contests/{contestID}/reject", h.RejectContest)
	}
}

func (h *ApprovalHandler) ListPendingProblems(w http.ResponseWriter, r *http.Request) {
	_, limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	problems, total, err := h.problemService.ListPending(r.Context(), offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pending problems")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": problems,
		"total": total,
	})
}

func (h *ApprovalHandler) ApproveProblem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "problemID"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	if err := h.problemService.Approve(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "problem not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to approve problem")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *ApprovalHandler) RejectProblem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "problemID"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	if err := h.problemService.Reject(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "problem not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to reject problem")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (h *ApprovalHandler) ListPendingContests(w http.ResponseWriter, r *http.Request) {
	_, limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	contests, total, err := h.contestService.ListPendingContests(r.Context(), offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pending contests")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": contests,
		"total": total,
	})
}

func (h *ApprovalHandler) ApproveContest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "contestID"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid contest id")
		return
	}

	if err := h.contestService.ApproveContest(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to approve contest")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *ApprovalHandler) RejectContest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "contestID"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid contest id")
		return
	}

	if err := h.contestService.RejectContest(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to reject contest")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

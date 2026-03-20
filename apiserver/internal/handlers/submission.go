package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/store"
)

const maxSubmissionBytes = 1 << 20

// SubmissionHandler provides HTTP handlers for submissions.
type SubmissionHandler struct {
	submissionService *services.SubmissionService
	problemService    *services.ProblemService
	userService       *services.UserService
}

// NewSubmissionHandler constructs a handler with the provided services.
func NewSubmissionHandler(submissionService *services.SubmissionService, problemService *services.ProblemService, userService *services.UserService) *SubmissionHandler {
	return &SubmissionHandler{
		submissionService: submissionService,
		problemService:    problemService,
		userService:       userService,
	}
}

// SubmissionRouter registers submission routes on the given router.
func SubmissionRouter(
	r chi.Router,
	submissionService *services.SubmissionService,
	problemService *services.ProblemService,
	userService *services.UserService,
	authMiddleware func(http.Handler) http.Handler,
) {
	handler := NewSubmissionHandler(submissionService, problemService, userService)

	if authMiddleware != nil {
		r.With(authMiddleware).Post("/", handler.CreateSubmission)
	} else {
		r.Post("/", handler.CreateSubmission)
	}
}

// SubmissionCreateRequest represents the payload for creating a submission.
type SubmissionCreateRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

// SubmissionCreateResponse is the create submission response payload.
type SubmissionCreateResponse struct {
	Submission  types.Submission `json:"submission"`
	ArtifactKey string           `json:"artifact_key"`
}

// CreateSubmission accepts source code for a problem and enqueues a judge job.
func (h *SubmissionHandler) CreateSubmission(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	problemID, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	problem, err := h.problemService.GetWithTestcases(r.Context(), problemID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "problem not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch problem")
		return
	}

	// Private problems: only accessible to admin or the problem creator.
	isAdmin := false
	if u, uErr := h.userService.GetByID(r.Context(), userID); uErr == nil {
		isAdmin = strings.EqualFold(u.Role, adminRole)
	}
	if problem.Visibility == "private" {
		if !isAdmin && problem.CreatorID != userID {
			writeError(w, http.StatusNotFound, "problem not found")
			return
		}
	}

	// Unapproved problems do not accept submissions.
	if problem.ApprovalStatus != "approved" {
		writeError(w, http.StatusForbidden, "problem is not yet approved for submissions")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSubmissionBytes)
	var req SubmissionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	req.Code = strings.TrimSpace(req.Code)
	req.Language = strings.TrimSpace(req.Language)
	if req.Code == "" || req.Language == "" {
		writeError(w, http.StatusBadRequest, "missing required fields")
		return
	}

	submission := types.Submission{
		ProblemID: problemID,
		UserID:    userID,
		Code:      req.Code,
		Language:  req.Language,
		Verdict:   types.VerdictPending,
	}

	created, artifactKey, err := h.submissionService.CreateAndEnqueue(r.Context(), submission, problem)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to submit")
		return
	}

	writeJSON(w, http.StatusCreated, SubmissionCreateResponse{
		Submission:  created,
		ArtifactKey: artifactKey,
	})
}

// GetSubmission returns a single submission by ID.
func (h *SubmissionHandler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submissionID")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid submission id")
		return
	}

	submission, err := h.submissionService.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "submission not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch submission")
		return
	}

	writeJSON(w, http.StatusOK, submission)
}

// ListSubmissions returns submissions filtered by optional problem_id and user_id query params.
func (h *SubmissionHandler) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	var problemID, userID int

	if v := r.URL.Query().Get("problem_id"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid problem_id")
			return
		}
		problemID = parsed
	}

	if v := r.URL.Query().Get("user_id"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		userID = parsed
	}

	submissions, err := h.submissionService.List(r.Context(), problemID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list submissions")
		return
	}

	if submissions == nil {
		submissions = []types.Submission{}
	}

	writeJSON(w, http.StatusOK, submissions)
}

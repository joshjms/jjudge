package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/store"
)

// ContestHandler provides HTTP handlers for contests.
type ContestHandler struct {
	contestService *services.ContestService
	problemService *services.ProblemService
	userService    *services.UserService
}

// NewContestHandler constructs a handler with the provided services.
func NewContestHandler(
	contestService *services.ContestService,
	problemService *services.ProblemService,
	userService *services.UserService,
) *ContestHandler {
	return &ContestHandler{
		contestService: contestService,
		problemService: problemService,
		userService:    userService,
	}
}

// ContestRouter registers all contest routes on the given router.
func ContestRouter(
	r chi.Router,
	contestService *services.ContestService,
	problemService *services.ProblemService,
	userService *services.UserService,
	authMiddleware func(http.Handler) http.Handler,
	optionalAuthMiddleware func(http.Handler) http.Handler,
) {
	h := NewContestHandler(contestService, problemService, userService)

	if optionalAuthMiddleware != nil {
		r.With(optionalAuthMiddleware).Get("/", h.ListContests)
	} else {
		r.Get("/", h.ListContests)
	}
	if authMiddleware != nil {
		r.With(authMiddleware, h.requireAdminOrManager).Post("/", h.CreateContest)
	} else {
		r.With(h.requireAdminOrManager).Post("/", h.CreateContest)
	}

	r.Route("/{contestID}", func(r chi.Router) {
		if optionalAuthMiddleware != nil {
			r.With(optionalAuthMiddleware).Get("/", h.GetContest)
		} else {
			r.Get("/", h.GetContest)
		}
		if authMiddleware != nil {
			r.With(authMiddleware, h.requireAdminOrContestOwner).Put("/", h.UpdateContest)
			r.With(authMiddleware, h.requireAdmin).Delete("/", h.DeleteContest)
		} else {
			r.With(h.requireAdminOrContestOwner).Put("/", h.UpdateContest)
			r.With(h.requireAdmin).Delete("/", h.DeleteContest)
		}
		// Problems sub-resource
		r.Get("/problems", h.ListContestProblems)
		if authMiddleware != nil {
			r.With(authMiddleware, h.requireAdminOrContestOwner).Post("/problems", h.AddContestProblem)
			r.With(authMiddleware, h.requireAdminOrContestOwner).Put("/problems/reorder", h.ReorderContestProblems)
			r.With(authMiddleware, h.requireAdminOrContestOwner).Put("/problems/{problemID}", h.UpdateContestProblem)
			r.With(authMiddleware, h.requireAdminOrContestOwner).Delete("/problems/{problemID}", h.RemoveContestProblem)
			r.With(authMiddleware, h.requireAdmin).Post("/problems/{problemID}/rejudge", h.RejudgeContestProblem)
		} else {
			r.With(h.requireAdminOrContestOwner).Post("/problems", h.AddContestProblem)
			r.With(h.requireAdminOrContestOwner).Put("/problems/reorder", h.ReorderContestProblems)
			r.With(h.requireAdminOrContestOwner).Put("/problems/{problemID}", h.UpdateContestProblem)
			r.With(h.requireAdminOrContestOwner).Delete("/problems/{problemID}", h.RemoveContestProblem)
			r.With(h.requireAdmin).Post("/problems/{problemID}/rejudge", h.RejudgeContestProblem)
		}

		// Registration
		if authMiddleware != nil {
			r.With(authMiddleware).Get("/register", h.CheckRegistration)
			r.With(authMiddleware).Post("/register", h.RegisterForContest)
			r.With(authMiddleware).Delete("/register", h.UnregisterFromContest)
		} else {
			r.Get("/register", h.CheckRegistration)
			r.Post("/register", h.RegisterForContest)
			r.Delete("/register", h.UnregisterFromContest)
		}
		if authMiddleware != nil {
			r.With(authMiddleware, h.requireAdmin).Get("/registrations", h.ListRegistrations)
		} else {
			r.With(h.requireAdmin).Get("/registrations", h.ListRegistrations)
		}

		// Submissions
		r.Get("/submissions", h.ListContestSubmissions)
		r.Get("/submissions/{submissionID}", h.GetContestSubmission)
		if authMiddleware != nil {
			r.With(authMiddleware).Post("/problems/{problemID}/submissions", h.CreateContestSubmission)
		} else {
			r.Post("/problems/{problemID}/submissions", h.CreateContestSubmission)
		}

		// Leaderboard
		r.Get("/leaderboard", h.GetLeaderboard)
	})
}

// ---------- Request/Response Types ----------

// ContestCreateRequest is the payload for creating or updating a contest.
type ContestCreateRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time"`
	ScoringType types.ScoringType `json:"scoring_type"`
	Visibility  string          `json:"visibility"`
}

// ContestListResponse is the paginated list response.
type ContestListResponse struct {
	Items []types.Contest `json:"items"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
	Total int             `json:"total"`
}

// ContestProblemAddRequest adds or updates a problem within a contest.
type ContestProblemAddRequest struct {
	ProblemID int `json:"problem_id"`
	Ordinal   int `json:"ordinal"`
	MaxPoints int `json:"max_points"`
}

// ContestProblemReorderRequest is a map of problem_id → ordinal.
type ContestProblemReorderRequest struct {
	Ordinals map[string]int `json:"ordinals"`
}

// ContestSubmissionCreateRequest is the payload for submitting code in a contest.
type ContestSubmissionCreateRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

// ContestSubmissionCreateResponse wraps the created contest submission.
type ContestSubmissionCreateResponse struct {
	Submission  types.ContestSubmission `json:"submission"`
	ArtifactKey string                  `json:"artifact_key"`
}

// LeaderboardResponse wraps the computed standings.
type LeaderboardResponse struct {
	Entries []types.ContestLeaderboardEntry `json:"entries"`
}

// ---------- Handlers ----------

func (h *ContestHandler) ListContests(w http.ResponseWriter, r *http.Request) {
	page, limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	publicOnly := !h.isCallerAdmin(r)
	contests, total, err := h.contestService.ListContests(r.Context(), offset, limit, publicOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list contests")
		return
	}

	if contests == nil {
		contests = []types.Contest{}
	}

	writeJSON(w, http.StatusOK, ContestListResponse{
		Items: contests,
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

func (h *ContestHandler) GetContest(w http.ResponseWriter, r *http.Request) {
	id, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	contest, err := h.contestService.GetContestWithProblems(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch contest")
		return
	}

	// Non-approved contests are only visible to admin or owner
	if contest.ApprovalStatus != "approved" && !h.isCallerAdmin(r) {
		callerID, _ := userIDFromContext(r.Context())
		if contest.OwnerID != callerID {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
	}

	writeJSON(w, http.StatusOK, contest)
}

func (h *ContestHandler) CreateContest(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ContestCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		writeError(w, http.StatusBadRequest, "start_time and end_time are required")
		return
	}
	if !req.EndTime.After(req.StartTime) {
		writeError(w, http.StatusBadRequest, "end_time must be after start_time")
		return
	}
	if req.ScoringType == "" {
		req.ScoringType = types.ScoringICPC
	}
	if req.Visibility == "" {
		req.Visibility = "public"
	}

	approvalStatus := "approved"
	if !h.isCallerAdmin(r) {
		approvalStatus = "pending"
	}

	contest := types.Contest{
		Title:          req.Title,
		Description:    req.Description,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		ScoringType:    req.ScoringType,
		Visibility:     req.Visibility,
		OwnerID:        userID,
		ApprovalStatus: approvalStatus,
	}

	created, err := h.contestService.CreateContest(r.Context(), contest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create contest")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *ContestHandler) UpdateContest(w http.ResponseWriter, r *http.Request) {
	id, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	existing, err := h.contestService.GetContest(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch contest")
		return
	}

	var req ContestCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if t := strings.TrimSpace(req.Title); t != "" {
		existing.Title = t
	}
	existing.Description = req.Description
	if !req.StartTime.IsZero() {
		existing.StartTime = req.StartTime
	}
	if !req.EndTime.IsZero() {
		existing.EndTime = req.EndTime
	}
	if req.ScoringType != "" {
		existing.ScoringType = req.ScoringType
	}
	if req.Visibility != "" {
		existing.Visibility = req.Visibility
	}
	// Only admin can change approval_status
	if !h.isCallerAdmin(r) {
		// keep existing approval_status (no self-approval)
	}

	updated, err := h.contestService.UpdateContest(r.Context(), existing)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update contest")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *ContestHandler) DeleteContest(w http.ResponseWriter, r *http.Request) {
	id, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.contestService.DeleteContest(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete contest")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------- Contest Problems ----------

func (h *ContestHandler) ListContestProblems(w http.ResponseWriter, r *http.Request) {
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	problems, err := h.contestService.ListContestProblems(r.Context(), contestID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list contest problems")
		return
	}

	if problems == nil {
		problems = []types.ContestProblem{}
	}
	writeJSON(w, http.StatusOK, problems)
}

func (h *ContestHandler) AddContestProblem(w http.ResponseWriter, r *http.Request) {
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req ContestProblemAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ProblemID < 1 {
		writeError(w, http.StatusBadRequest, "problem_id is required")
		return
	}
	if req.MaxPoints == 0 {
		req.MaxPoints = 100
	}

	cp := types.ContestProblem{
		ContestID: contestID,
		ProblemID: req.ProblemID,
		Ordinal:   req.Ordinal,
		MaxPoints: req.MaxPoints,
	}

	added, err := h.contestService.AddContestProblem(r.Context(), cp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add contest problem")
		return
	}
	writeJSON(w, http.StatusCreated, added)
}

func (h *ContestHandler) UpdateContestProblem(w http.ResponseWriter, r *http.Request) {
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	problemID, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req ContestProblemAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	cp := types.ContestProblem{
		ContestID: contestID,
		ProblemID: problemID,
		Ordinal:   req.Ordinal,
		MaxPoints: req.MaxPoints,
	}
	if cp.MaxPoints == 0 {
		cp.MaxPoints = 100
	}

	updated, err := h.contestService.AddContestProblem(r.Context(), cp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update contest problem")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *ContestHandler) RemoveContestProblem(w http.ResponseWriter, r *http.Request) {
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	problemID, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.contestService.RemoveContestProblem(r.Context(), contestID, problemID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest problem not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to remove contest problem")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ContestHandler) ReorderContestProblems(w http.ResponseWriter, r *http.Request) {
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req ContestProblemReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Convert string keys to int
	ordinals := make(map[int]int, len(req.Ordinals))
	for k, v := range req.Ordinals {
		pid, err := strconv.Atoi(k)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid problem_id in ordinals")
			return
		}
		ordinals[pid] = v
	}

	if err := h.contestService.ReorderContestProblems(r.Context(), contestID, ordinals); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reorder problems")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Registrations ----------

func (h *ContestHandler) RegisterForContest(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.contestService.GetContest(r.Context(), contestID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch contest")
		return
	}

	if err := h.contestService.Register(r.Context(), contestID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

func (h *ContestHandler) UnregisterFromContest(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.contestService.Unregister(r.Context(), contestID, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			w.WriteHeader(http.StatusNoContent) // idempotent: not registered is fine
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to unregister")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ContestHandler) CheckRegistration(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	registered, err := h.contestService.IsRegistered(r.Context(), contestID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check registration")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"registered": registered})
}

func (h *ContestHandler) ListRegistrations(w http.ResponseWriter, r *http.Request) {
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	regs, err := h.contestService.ListRegistrations(r.Context(), contestID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list registrations")
		return
	}

	if regs == nil {
		regs = []types.ContestRegistration{}
	}
	writeJSON(w, http.StatusOK, regs)
}

// ---------- Contest Submissions ----------

func (h *ContestHandler) CreateContestSubmission(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	problemID, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	contest, err := h.contestService.GetContest(r.Context(), contestID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch contest")
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

	r.Body = http.MaxBytesReader(w, r.Body, maxSubmissionBytes)
	var req ContestSubmissionCreateRequest
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

	cs := types.ContestSubmission{
		ContestID: contestID,
		ProblemID: problemID,
		UserID:    userID,
		Code:      req.Code,
		Language:  req.Language,
		Verdict:   types.VerdictPending,
	}

	created, artifactKey, err := h.contestService.CreateAndEnqueueContestSubmission(r.Context(), cs, problem, contest)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotRegistered):
			writeError(w, http.StatusForbidden, "you must register for the contest before submitting")
		case errors.Is(err, services.ErrContestNotActive):
			writeError(w, http.StatusBadRequest, "contest is not currently active")
		default:
			writeError(w, http.StatusInternalServerError, "failed to submit")
		}
		return
	}

	writeJSON(w, http.StatusCreated, ContestSubmissionCreateResponse{
		Submission:  created,
		ArtifactKey: artifactKey,
	})
}

func (h *ContestHandler) GetContestSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submissionID")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid submission id")
		return
	}

	submission, err := h.contestService.GetContestSubmission(r.Context(), id)
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

func (h *ContestHandler) ListContestSubmissions(w http.ResponseWriter, r *http.Request) {
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var problemID, userID int
	if v := r.URL.Query().Get("problem_id"); v != "" {
		problemID, err = strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid problem_id")
			return
		}
	}
	if v := r.URL.Query().Get("user_id"); v != "" {
		userID, err = strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
	}

	submissions, err := h.contestService.ListContestSubmissions(r.Context(), contestID, problemID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list submissions")
		return
	}

	if submissions == nil {
		submissions = []types.ContestSubmission{}
	}
	writeJSON(w, http.StatusOK, submissions)
}

// ---------- Leaderboard ----------

func (h *ContestHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	contest, err := h.contestService.GetContest(r.Context(), contestID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch contest")
		return
	}

	entries, err := h.contestService.GetLeaderboard(r.Context(), contest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute leaderboard")
		return
	}

	if entries == nil {
		entries = []types.ContestLeaderboardEntry{}
	}
	writeJSON(w, http.StatusOK, LeaderboardResponse{Entries: entries})
}

// ---------- Rejudge ----------

func (h *ContestHandler) RejudgeContestProblem(w http.ResponseWriter, r *http.Request) {
	contestID, err := parseContestID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	problemID, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	contest, err := h.contestService.GetContest(r.Context(), contestID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contest not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch contest")
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

	if err := h.contestService.RejudgeContestProblem(r.Context(), contestID, problemID, problem, contest); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to rejudge")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejudge enqueued"})
}

// ---------- Middleware ----------

func (h *ContestHandler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromContext(r.Context())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		user, err := h.userService.GetByID(r.Context(), userID)
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

// requireAdminOrManager allows admin or manager roles.
func (h *ContestHandler) requireAdminOrManager(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromContext(r.Context())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		user, err := h.userService.GetByID(r.Context(), userID)
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
			writeError(w, http.StatusForbidden, "admin or manager access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAdminOrContestOwner allows admin, or manager/owner of the contest.
func (h *ContestHandler) requireAdminOrContestOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromContext(r.Context())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		user, err := h.userService.GetByID(r.Context(), userID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to load user")
			return
		}

		role := strings.ToLower(user.Role)
		if role == adminRole {
			next.ServeHTTP(w, r)
			return
		}

		if role != managerRole {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}

		// Manager: must be owner of the contest
		id, err := parseContestID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		contest, err := h.contestService.GetContest(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "contest not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to load contest")
			return
		}

		if contest.OwnerID != userID {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isCallerAdmin returns true if the authenticated caller is an admin.
func (h *ContestHandler) isCallerAdmin(r *http.Request) bool {
	userID, err := userIDFromContext(r.Context())
	if err != nil {
		return false
	}
	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		return false
	}
	return strings.EqualFold(user.Role, adminRole)
}

// ---------- Helpers ----------

func parseContestID(r *http.Request) (int, error) {
	raw := chi.URLParam(r, "contestID")
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		return 0, errors.New("invalid contest id")
	}
	return id, nil
}

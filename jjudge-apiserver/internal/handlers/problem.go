package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/store"
	"github.com/jjudge-oj/apiserver/types"
)

const (
	defaultPage         = 1
	defaultLimit        = 20
	maxLimit            = 100
	maxMultipartMemory  = 128 << 20
	maxBundleBytes      = 256 << 20
	adminRole           = "admin"
	formFieldBundle     = "bundle"
	formFieldGroups     = "testcase_groups"
	formFieldTitle      = "title"
	formFieldDesc       = "description"
	formFieldDifficulty = "difficulty"
	formFieldTimeLimit  = "time_limit"
	formFieldMemLimit   = "memory_limit"
	formFieldTags       = "tags"
)

// BundleFile represents an uploaded testcase bundle.
type BundleFile struct {
	Filename string
	Data     []byte
}

// ProblemHandler provides HTTP handlers for problems.
type ProblemHandler struct {
	problemService *services.ProblemService
	userService    *services.UserService
}

// NewProblemHandler constructs a handler with the provided store.
func NewProblemHandler(problemService *services.ProblemService, userService *services.UserService) *ProblemHandler {
	return &ProblemHandler{
		problemService: problemService,
		userService:    userService,
	}
}

// ProblemRouter registers problem routes on the given router.
func ProblemRouter(
	r chi.Router,
	problemService *services.ProblemService,
	userService *services.UserService,
	authMiddleware func(http.Handler) http.Handler,
) {
	handler := NewProblemHandler(problemService, userService)

	r.Get("/", handler.ListProblems)
	if authMiddleware != nil {
		r.With(authMiddleware, handler.requireAdmin).Post("/", handler.CreateProblem)
	} else {
		r.With(handler.requireAdmin).Post("/", handler.CreateProblem)
	}
	r.Route("/{problemID}", func(r chi.Router) {
		r.Get("/", handler.GetProblem)
		if authMiddleware != nil {
			r.With(authMiddleware, handler.requireAdmin).Put("/", handler.UpdateProblem)
			r.With(authMiddleware, handler.requireAdmin).Delete("/", handler.DeleteProblem)
		} else {
			r.With(handler.requireAdmin).Put("/", handler.UpdateProblem)
			r.With(handler.requireAdmin).Delete("/", handler.DeleteProblem)
		}
	})
}

func (h *ProblemHandler) ListProblems(w http.ResponseWriter, r *http.Request) {
	page, limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	items, total, err := h.problemService.List(r.Context(), offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list problems")
		return
	}

	resp := ProblemListResponse{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ProblemHandler) GetProblem(w http.ResponseWriter, r *http.Request) {
	id, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	problem, err := h.problemService.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "problem not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch problem")
		return
	}

	writeJSON(w, http.StatusOK, problem)
}

func (h *ProblemHandler) CreateProblem(w http.ResponseWriter, r *http.Request) {
	req, err := parseProblemForm(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tcBundle, err := h.problemService.GetTestcaseBundleFromArchive(req.Bundle.Filename, req.Bundle.Data, req.TestcaseGroups)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	problem := types.Problem{
		Title:          req.Title,
		Description:    req.Description,
		Difficulty:     req.Difficulty,
		TimeLimit:      req.TimeLimit,
		MemoryLimit:    req.MemoryLimit,
		Tags:           req.Tags,
		TestcaseBundle: tcBundle,
	}

	created, err := h.problemService.Create(r.Context(), problem)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create problem")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *ProblemHandler) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	id, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req, err := parseProblemForm(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Update testcase bundle if provided.
	if req.Bundle.Data != nil {
		tcBundle, err := h.problemService.GetTestcaseBundleFromArchive(req.Bundle.Filename, req.Bundle.Data, req.TestcaseGroups)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.problemService.UpdateTestcaseBundle(r.Context(), id, tcBundle); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update testcase bundle")
			return
		}
	}

	updated, err := h.problemService.Update(r.Context(), types.Problem{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		Difficulty:  req.Difficulty,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		Tags:        req.Tags,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "problem not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update problem")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *ProblemHandler) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	id, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.problemService.Delete(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "problem not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete problem")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ProblemUpsertRequest represents the parsed multipart form payload.
type ProblemUpsertRequest struct {
	Title          string
	Description    string
	Difficulty     int
	TimeLimit      int64
	MemoryLimit    int64
	Tags           []string
	TestcaseGroups []types.TestcaseGroup
	Bundle         BundleFile
}

// ProblemListResponse is the paginated list response payload.
type ProblemListResponse struct {
	Items []types.Problem `json:"items"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
	Total int             `json:"total"`
}

// ErrorResponse is a simple error payload.
type ErrorResponse struct {
	Error string `json:"error"`
}

func parsePagination(r *http.Request) (page, limit, offset int, err error) {
	page = defaultPage
	limit = defaultLimit

	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 {
			return 0, 0, 0, errors.New("invalid page")
		}
	}

	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if rawLimit == "" {
		rawLimit = strings.TrimSpace(r.URL.Query().Get("per_page"))
	}
	if rawLimit != "" {
		limit, err = strconv.Atoi(rawLimit)
		if err != nil || limit < 1 {
			return 0, 0, 0, errors.New("invalid limit")
		}
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	offset = (page - 1) * limit
	return page, limit, offset, nil
}

func parseProblemID(r *http.Request) (int, error) {
	raw := chi.URLParam(r, "problemID")
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		return 0, errors.New("invalid problem id")
	}
	return id, nil
}

func parseProblemForm(r *http.Request) (ProblemUpsertRequest, error) {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		return ProblemUpsertRequest{}, errors.New("invalid multipart form")
	}

	title := strings.TrimSpace(r.FormValue(formFieldTitle))
	if title == "" {
		return ProblemUpsertRequest{}, errors.New("title is required")
	}

	description := strings.TrimSpace(r.FormValue(formFieldDesc))
	if description == "" {
		return ProblemUpsertRequest{}, errors.New("description is required")
	}

	difficulty, err := parseOptionalInt(r.FormValue(formFieldDifficulty))
	if err != nil {
		return ProblemUpsertRequest{}, errors.New("invalid difficulty")
	}

	timeLimit, err := parseOptionalInt64(r.FormValue(formFieldTimeLimit))
	if err != nil {
		return ProblemUpsertRequest{}, errors.New("invalid time limit")
	}

	memoryLimit, err := parseOptionalInt64(r.FormValue(formFieldMemLimit))
	if err != nil {
		return ProblemUpsertRequest{}, errors.New("invalid memory limit")
	}

	tags := parseTags(r.FormValue(formFieldTags))

	var tcGroups []types.TestcaseGroup
	if rawGroups := strings.TrimSpace(r.FormValue(formFieldGroups)); rawGroups != "" {
		if err := json.Unmarshal([]byte(rawGroups), &tcGroups); err != nil {
			return ProblemUpsertRequest{}, errors.New("invalid testcase groups")
		}
	}

	bundle, err := parseBundleFile(r.MultipartForm)
	if err != nil {
		return ProblemUpsertRequest{}, err
	}

	return ProblemUpsertRequest{
		Title:          title,
		Description:    description,
		Difficulty:     difficulty,
		TimeLimit:      timeLimit,
		MemoryLimit:    memoryLimit,
		Tags:           tags,
		TestcaseGroups: tcGroups,
		Bundle:         bundle,
	}, nil
}

func parseOptionalInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func parseOptionalInt64(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func parseTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func parseBundleFile(form *multipart.Form) (BundleFile, error) {
	if form == nil {
		return BundleFile{}, errors.New("missing form data")
	}

	files := form.File[formFieldBundle]
	if len(files) == 0 {
		return BundleFile{}, errors.New("bundle file is required")
	}
	if len(files) > 1 {
		return BundleFile{}, errors.New("only one bundle file is allowed")
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		return BundleFile{}, fmt.Errorf("failed to read bundle file: %w", err)
	}

	data, err := readFileLimited(file, maxBundleBytes)
	_ = file.Close()
	if err != nil {
		return BundleFile{}, err
	}

	return BundleFile{
		Filename: fileHeader.Filename,
		Data:     data,
	}, nil
}

func readFileLimited(reader io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(reader, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, errors.New("failed to read upload")
	}
	if int64(len(data)) > limit {
		return nil, errors.New("uploaded file too large")
	}
	return data, nil
}

func (h *ProblemHandler) requireAdmin(next http.Handler) http.Handler {
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

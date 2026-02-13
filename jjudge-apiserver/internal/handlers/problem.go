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
	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/store"
)

const (
	defaultPage        = 1
	defaultLimit       = 20
	maxLimit           = 100
	maxMultipartMemory = 128 << 20
	maxBundleBytes     = 256 << 20
	adminRole          = "admin"
	formFieldMetadata  = "metadata"
)

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
	req, err := parseProblemForm(r, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	problem := types.Problem{
		Title:       req.Metadata.Title,
		Description: req.Metadata.Description,
		Difficulty:  req.Metadata.Difficulty,
		TimeLimit:   req.Metadata.TimeLimit,
		MemoryLimit: req.Metadata.MemoryLimit,
		Tags:        req.Metadata.Tags,
	}

	created, err := h.problemService.Create(r.Context(), problem)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create problem")
		return
	}

	tcBundle, err := h.problemService.GetTestcaseBundleFromFiles(r.Context(), created.ID, req.TestcaseFiles, req.Metadata.TestcaseBundle.TestcaseGroups)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.problemService.UpdateTestcaseBundle(r.Context(), created.ID, tcBundle); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update testcase bundle")
		return
	}
	created.TestcaseBundle = tcBundle

	writeJSON(w, http.StatusCreated, created)
}

func (h *ProblemHandler) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	id, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req, err := parseProblemForm(r, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(req.TestcaseFiles) > 0 {
		tcBundle, err := h.problemService.GetTestcaseBundleFromFiles(r.Context(), id, req.TestcaseFiles, req.Metadata.TestcaseBundle.TestcaseGroups)
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
		Title:       req.Metadata.Title,
		Description: req.Metadata.Description,
		Difficulty:  req.Metadata.Difficulty,
		TimeLimit:   req.Metadata.TimeLimit,
		MemoryLimit: req.Metadata.MemoryLimit,
		Tags:        req.Metadata.Tags,
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
	Metadata      types.Problem
	TestcaseFiles map[string][]byte
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

func parseProblemForm(r *http.Request, requireTestcases bool) (ProblemUpsertRequest, error) {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		return ProblemUpsertRequest{}, errors.New("invalid multipart form")
	}

	metadata, err := parseMetadata(r)
	if err != nil {
		return ProblemUpsertRequest{}, err
	}

	testcaseFiles, err := parseTestcaseFiles(r.MultipartForm, metadata.TestcaseBundle.TestcaseGroups, requireTestcases)
	if err != nil {
		return ProblemUpsertRequest{}, err
	}

	return ProblemUpsertRequest{
		Metadata:      metadata,
		TestcaseFiles: testcaseFiles,
	}, nil
}

func parseMetadata(r *http.Request) (types.Problem, error) {
	raw := strings.TrimSpace(r.FormValue(formFieldMetadata))
	if raw == "" {
		return types.Problem{}, errors.New("metadata is required")
	}

	var metadata types.Problem
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return types.Problem{}, errors.New("invalid metadata")
	}

	metadata.Title = strings.TrimSpace(metadata.Title)
	if metadata.Title == "" {
		return types.Problem{}, errors.New("title is required")
	}
	metadata.Description = strings.TrimSpace(metadata.Description)
	if metadata.Description == "" {
		return types.Problem{}, errors.New("description is required")
	}

	return metadata, nil
}

func parseTestcaseFiles(form *multipart.Form, groups []types.TestcaseGroup, requireTestcases bool) (map[string][]byte, error) {
	if form == nil {
		return nil, errors.New("missing form data")
	}

	keys := testcaseKeysFromGroups(groups)
	if len(keys) == 0 {
		if !requireTestcases && len(form.File) == 0 {
			return map[string][]byte{}, nil
		}
		return nil, errors.New("testcase files are required")
	}
	if !requireTestcases && len(form.File) == 0 {
		return map[string][]byte{}, nil
	}

	filesByKey := make(map[string][]byte, len(keys))
	var totalBytes int64
	for _, key := range keys {
		files := form.File[key]
		if len(files) == 0 {
			return nil, fmt.Errorf("missing testcase file for key: %s", key)
		}
		if len(files) > 1 {
			return nil, fmt.Errorf("multiple testcase files provided for key: %s", key)
		}

		fileHeader := files[0]
		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to read testcase file: %w", err)
		}

		data, err := readFileLimited(file, maxBundleBytes)
		_ = file.Close()
		if err != nil {
			return nil, err
		}

		totalBytes += int64(len(data))
		if totalBytes > maxBundleBytes {
			return nil, errors.New("uploaded files too large")
		}

		filesByKey[key] = data
	}
	return filesByKey, nil
}

func testcaseKeysFromGroups(groups []types.TestcaseGroup) []string {
	keys := make([]string, 0)
	seen := make(map[string]struct{})
	for _, group := range groups {
		for _, testcase := range group.Testcases {
			if testcase.InKey != "" {
				if _, exists := seen[testcase.InKey]; !exists {
					seen[testcase.InKey] = struct{}{}
					keys = append(keys, testcase.InKey)
				}
			}
			if testcase.OutKey != "" {
				if _, exists := seen[testcase.OutKey]; !exists {
					seen[testcase.OutKey] = struct{}{}
					keys = append(keys, testcase.OutKey)
				}
			}
		}
	}
	return keys
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

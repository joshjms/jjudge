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
	defaultPage           = 1
	defaultLimit          = 20
	maxLimit              = 100
	maxMultipartMemory    = 128 << 20
	maxBundleBytes        = 256 << 20
	adminRole             = "admin"
	managerRole           = "manager"
	formFieldMetadata     = "metadata"
	formFieldTestcasesZip = "testcases_zip"
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
	optionalAuthMiddleware func(http.Handler) http.Handler,
) {
	handler := NewProblemHandler(problemService, userService)

	if optionalAuthMiddleware != nil {
		r.With(optionalAuthMiddleware).Get("/", handler.ListProblems)
	} else {
		r.Get("/", handler.ListProblems)
	}
	if authMiddleware != nil {
		r.With(authMiddleware, handler.requireAdminOrManager).Post("/", handler.CreateProblem)
		r.With(authMiddleware, handler.requireAdminOrManager).Post("/zip", handler.CreateProblemFromZip)
	} else {
		r.With(handler.requireAdminOrManager).Post("/", handler.CreateProblem)
		r.With(handler.requireAdminOrManager).Post("/zip", handler.CreateProblemFromZip)
	}
	r.Route("/{problemID}", func(r chi.Router) {
		if optionalAuthMiddleware != nil {
			r.With(optionalAuthMiddleware).Get("/", handler.GetProblem)
		} else {
			r.Get("/", handler.GetProblem)
		}
		if authMiddleware != nil {
			r.With(authMiddleware, handler.requireAdminOrProblemCreator).Put("/", handler.UpdateProblem)
			r.With(authMiddleware, handler.requireAdminOrProblemCreator).Put("/zip", handler.UpdateProblemFromZip)
			r.With(authMiddleware, handler.requireAdmin).Delete("/", handler.DeleteProblem)
		} else {
			r.With(handler.requireAdminOrProblemCreator).Put("/", handler.UpdateProblem)
			r.With(handler.requireAdminOrProblemCreator).Put("/zip", handler.UpdateProblemFromZip)
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

	isAdmin := h.isCallerAdmin(r)
	callerID := 0
	if !isAdmin {
		callerID, _ = userIDFromContext(r.Context())
	}
	items, total, err := h.problemService.List(r.Context(), offset, limit, callerID, isAdmin)
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

	isAdmin := h.isCallerAdmin(r)
	callerID, _ := userIDFromContext(r.Context())
	isCreator := callerID > 0 && problem.CreatorID == callerID

	if problem.Visibility == "private" && !isAdmin && !isCreator {
		writeError(w, http.StatusNotFound, "problem not found")
		return
	}

	// Non-approved problems are only visible to admin or creator
	if problem.ApprovalStatus != "approved" && !isAdmin && !isCreator {
		writeError(w, http.StatusNotFound, "problem not found")
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

	userID, _ := userIDFromContext(r.Context())
	approvalStatus := "approved"
	if !h.isCallerAdmin(r) {
		approvalStatus = "pending"
	}

	problem := types.Problem{
		Title:          req.Metadata.Title,
		Description:    req.Metadata.Description,
		Difficulty:     req.Metadata.Difficulty,
		TimeLimit:      req.Metadata.TimeLimit,
		MemoryLimit:    req.Metadata.MemoryLimit,
		Tags:           req.Metadata.Tags,
		Visibility:     req.Metadata.Visibility,
		CreatorID:      userID,
		ApprovalStatus: approvalStatus,
	}

	created, err := h.problemService.Create(r.Context(), problem)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create problem")
		return
	}

	// Process and upload testcase files
	updatedGroups, err := h.problemService.ProcessTestcaseFiles(r.Context(), created.ID, req.TestcaseFiles, req.Metadata.TestcaseGroups)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Save testcase groups to database
	if err := h.problemService.SaveTestcaseGroups(r.Context(), created.ID, updatedGroups); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save testcase groups")
		return
	}

	created.TestcaseGroups = updatedGroups
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
		// Process and upload testcase files
		updatedGroups, err := h.problemService.ProcessTestcaseFiles(r.Context(), id, req.TestcaseFiles, req.Metadata.TestcaseGroups)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Save testcase groups to database
		if err := h.problemService.SaveTestcaseGroups(r.Context(), id, updatedGroups); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save testcase groups")
			return
		}
	}

	// Load existing problem to get its current approval_status and creator_id
	existing, err := h.problemService.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "problem not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch problem")
		return
	}

	approvalStatus := existing.ApprovalStatus
	// Only admin can change approval_status
	if h.isCallerAdmin(r) && req.Metadata.ApprovalStatus != "" {
		approvalStatus = req.Metadata.ApprovalStatus
	}

	updated, err := h.problemService.Update(r.Context(), types.Problem{
		ID:             id,
		Title:          req.Metadata.Title,
		Description:    req.Metadata.Description,
		Difficulty:     req.Metadata.Difficulty,
		TimeLimit:      req.Metadata.TimeLimit,
		MemoryLimit:    req.Metadata.MemoryLimit,
		Tags:           req.Metadata.Tags,
		Visibility:     req.Metadata.Visibility,
		CreatorID:      existing.CreatorID,
		ApprovalStatus: approvalStatus,
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

	testcaseFiles, err := parseTestcaseFiles(r.MultipartForm, metadata.TestcaseGroups, requireTestcases)
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

// ZipProblemUpsertRequest is the parsed payload for zip-based problem creation/update.
type ZipProblemUpsertRequest struct {
	Metadata types.Problem
	ZipData  []byte
}

func parseZipProblemForm(r *http.Request) (ZipProblemUpsertRequest, error) {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		return ZipProblemUpsertRequest{}, errors.New("invalid multipart form")
	}

	metadata, err := parseMetadata(r)
	if err != nil {
		return ZipProblemUpsertRequest{}, err
	}

	files := r.MultipartForm.File[formFieldTestcasesZip]
	if len(files) == 0 {
		return ZipProblemUpsertRequest{}, fmt.Errorf("%s file is required", formFieldTestcasesZip)
	}
	if len(files) > 1 {
		return ZipProblemUpsertRequest{}, fmt.Errorf("only one %s file is allowed", formFieldTestcasesZip)
	}

	f, err := files[0].Open()
	if err != nil {
		return ZipProblemUpsertRequest{}, errors.New("failed to read zip file")
	}
	defer f.Close()

	zipData, err := readFileLimited(f, maxBundleBytes)
	if err != nil {
		return ZipProblemUpsertRequest{}, err
	}

	return ZipProblemUpsertRequest{Metadata: metadata, ZipData: zipData}, nil
}

func (h *ProblemHandler) CreateProblemFromZip(w http.ResponseWriter, r *http.Request) {
	req, err := parseZipProblemForm(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := userIDFromContext(r.Context())
	approvalStatus := "approved"
	if !h.isCallerAdmin(r) {
		approvalStatus = "pending"
	}

	problem := types.Problem{
		Title:          req.Metadata.Title,
		Description:    req.Metadata.Description,
		Difficulty:     req.Metadata.Difficulty,
		TimeLimit:      req.Metadata.TimeLimit,
		MemoryLimit:    req.Metadata.MemoryLimit,
		Tags:           req.Metadata.Tags,
		Visibility:     req.Metadata.Visibility,
		CreatorID:      userID,
		ApprovalStatus: approvalStatus,
	}

	created, err := h.problemService.Create(r.Context(), problem)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create problem")
		return
	}

	updatedGroups, err := h.problemService.ProcessTestcasesFromZip(r.Context(), created.ID, req.ZipData, req.Metadata.TestcaseGroups)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.problemService.SaveTestcaseGroups(r.Context(), created.ID, updatedGroups); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save testcase groups")
		return
	}

	created.TestcaseGroups = updatedGroups
	writeJSON(w, http.StatusCreated, created)
}

func (h *ProblemHandler) UpdateProblemFromZip(w http.ResponseWriter, r *http.Request) {
	id, err := parseProblemID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req, err := parseZipProblemForm(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updatedGroups, err := h.problemService.ProcessTestcasesFromZip(r.Context(), id, req.ZipData, req.Metadata.TestcaseGroups)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.problemService.SaveTestcaseGroups(r.Context(), id, updatedGroups); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save testcase groups")
		return
	}

	existing, err := h.problemService.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "problem not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch problem")
		return
	}

	approvalStatus := existing.ApprovalStatus
	if h.isCallerAdmin(r) && req.Metadata.ApprovalStatus != "" {
		approvalStatus = req.Metadata.ApprovalStatus
	}

	updated, err := h.problemService.Update(r.Context(), types.Problem{
		ID:             id,
		Title:          req.Metadata.Title,
		Description:    req.Metadata.Description,
		Difficulty:     req.Metadata.Difficulty,
		TimeLimit:      req.Metadata.TimeLimit,
		MemoryLimit:    req.Metadata.MemoryLimit,
		Tags:           req.Metadata.Tags,
		Visibility:     req.Metadata.Visibility,
		CreatorID:      existing.CreatorID,
		ApprovalStatus: approvalStatus,
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

// isCallerAdmin returns true if the request context contains a valid user with admin role.
func (h *ProblemHandler) isCallerAdmin(r *http.Request) bool {
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

// callerRole returns the role of the authenticated user, or empty string if not authenticated.
func (h *ProblemHandler) callerRole(r *http.Request) string {
	userID, err := userIDFromContext(r.Context())
	if err != nil {
		return ""
	}
	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		return ""
	}
	return strings.ToLower(user.Role)
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

// requireAdminOrManager allows admin or manager roles.
func (h *ProblemHandler) requireAdminOrManager(next http.Handler) http.Handler {
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

// requireAdminOrProblemCreator allows admin, or manager who is the creator of the problem.
func (h *ProblemHandler) requireAdminOrProblemCreator(next http.Handler) http.Handler {
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

		// Manager: must be creator of the problem
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
			writeError(w, http.StatusInternalServerError, "failed to load problem")
			return
		}

		if problem.CreatorID != userID {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		next.ServeHTTP(w, r)
	})
}

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/storage"
	"github.com/jjudge-oj/apiserver/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const defaultTokenTTL = 24 * time.Hour
const defaultUserRole = "user"

// AuthHandler provides JWT authentication endpoints.
type AuthHandler struct {
	userService *services.UserService
	storage     *storage.Storage
	secret      []byte
	tokenTTL    time.Duration
}

// NewAuthHandler constructs an AuthHandler with the provided dependencies.
func NewAuthHandler(userService *services.UserService, jwtSecret string, storageClient *storage.Storage) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		storage:     storageClient,
		secret:      []byte(jwtSecret),
		tokenTTL:    defaultTokenTTL,
	}
}

// AuthRouter registers auth routes on the given router.
func AuthRouter(r chi.Router, userService *services.UserService, jwtSecret string) {
	handler := NewAuthHandler(userService, jwtSecret, nil)

	r.Post("/register", handler.Register)
	r.Post("/login", handler.Login)
	r.With(handler.RequireAuth).Get("/me", handler.Me)
	r.With(handler.RequireAuth, handler.requireAdmin).Get("/users", handler.ListUsers)
	r.With(handler.RequireAuth, handler.requireAdmin).Patch("/users/{id}/role", handler.UpdateUserRole)
}

// UserRouter registers public and authenticated user profile routes.
func UserRouter(r chi.Router, userService *services.UserService, jwtSecret string, storageClient *storage.Storage) {
	handler := NewAuthHandler(userService, jwtSecret, storageClient)

	r.Get("/{username}", handler.GetProfile)
	r.Get("/{username}/avatar", handler.GetAvatar)
	r.With(handler.RequireAuth).Patch("/{username}/profile", handler.UpdateProfile)
	r.With(handler.RequireAuth).Post("/{username}/avatar", handler.UploadAvatar)
}

// RequireAuth enforces JWT authentication and injects the subject into context.
func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return requireAuth(h.secret)(next)
}

// RequireAuth constructs auth middleware for other routers.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return requireAuth([]byte(jwtSecret))
}

// OptionalAuth constructs middleware that populates the user ID in context if a
// valid bearer token is present, but does not reject requests without one.
func OptionalAuth(jwtSecret string) func(http.Handler) http.Handler {
	secret := []byte(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tokenString, err := bearerToken(r); err == nil {
				if subject, err := parseTokenSubject(tokenString, secret); err == nil {
					ctx := context.WithValue(r.Context(), contextSubjectKey, subject)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := bearerToken(r)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			subject, err := parseTokenSubject(tokenString, secret)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			ctx := context.WithValue(r.Context(), contextSubjectKey, subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Register creates a new user account and returns a JWT.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	req.Email = strings.TrimSpace(req.Email)
	req.Name = strings.TrimSpace(req.Name)
	if req.Username == "" || req.Email == "" || req.Name == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "missing required fields")
		return
	}

	if _, err := h.userService.GetByUsername(r.Context(), req.Username); err == nil {
		writeError(w, http.StatusConflict, "username already exists")
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to check user")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	user, err := h.userService.Create(r.Context(), types.User{
		Username:     req.Username,
		Email:        req.Email,
		Name:         req.Name,
		Role:         defaultUserRole,
		PasswordHash: string(hashed),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := issueToken(user.ID, h.secret, h.tokenTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	writeJSON(w, http.StatusCreated, AuthResponse{Token: token, User: user})
}

// Login verifies credentials and returns a JWT.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "missing credentials")
		return
	}

	user, err := h.userService.GetByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := issueToken(user.ID, h.secret, h.tokenTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{Token: token, User: user})
}

// Me returns the current authenticated user.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, user)
}

// ListUsers returns a paginated list of all users. Admin only.
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	users, total, err := h.userService.List(r.Context(), offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	if users == nil {
		users = []types.User{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": users,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// UpdateUserRole changes the role of a user. Admin only.
func (h *AuthHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	switch body.Role {
	case "admin", "manager", "user":
	default:
		writeError(w, http.StatusBadRequest, "role must be admin, manager, or user")
		return
	}

	user, err := h.userService.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	user.Role = body.Role
	updated, err := h.userService.Update(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// GetProfile returns the public profile for a user by username.
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	user, err := h.userService.GetByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// UpdateProfile allows an authenticated user to update their own bio and socials.
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	callerID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.userService.GetByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	if user.ID != callerID {
		writeError(w, http.StatusForbidden, "cannot edit another user's profile")
		return
	}

	var body struct {
		Bio        *string `json:"bio"`
		GitHub     *string `json:"github"`
		Codeforces *string `json:"codeforces"`
		AtCoder    *string `json:"atcoder"`
		Website    *string `json:"website"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user.Bio = body.Bio
	user.GitHub = body.GitHub
	user.Codeforces = body.Codeforces
	user.AtCoder = body.AtCoder
	user.Website = body.Website

	updated, err := h.userService.Update(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *AuthHandler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromContext(r.Context())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		user, err := h.userService.GetByID(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !strings.EqualFold(user.Role, adminRole) {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  types.User `json:"user"`
}

func issueToken(userID int, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.Itoa(userID),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func parseTokenSubject(tokenString string, secret []byte) (string, error) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return secret, nil
	})
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("invalid token")
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return "", errors.New("missing subject")
	}
	return claims.Subject, nil
}

// avatarContentTypes maps allowed MIME types to file extensions.
var avatarContentTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

const maxAvatarSize = 2 << 20 // 2 MB

// UploadAvatar handles POST /{username}/avatar.
// It accepts a multipart file upload ("avatar" field) and stores the image in object storage.
func (h *AuthHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	callerID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.userService.GetByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	if user.ID != callerID {
		writeError(w, http.StatusForbidden, "cannot edit another user's avatar")
		return
	}

	if h.storage == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured")
		return
	}

	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing avatar field")
		return
	}
	defer file.Close()

	// Detect content type from the first 512 bytes.
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])

	ext, ok := avatarContentTypes[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported image type; use jpeg, png, gif, or webp")
		return
	}

	// Seek back so we upload the full file.
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read file")
			return
		}
	}

	key := fmt.Sprintf("avatars/%d.%s", user.ID, ext)
	if err := h.storage.Put(r.Context(), key, file, header.Size, contentType); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store avatar")
		return
	}

	user.AvatarURL = &key
	updated, err := h.userService.Update(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// GetAvatar handles GET /{username}/avatar.
// It streams the user's avatar image from object storage.
func (h *AuthHandler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	user, err := h.userService.GetByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	if user.AvatarURL == nil || *user.AvatarURL == "" {
		writeError(w, http.StatusNotFound, "no avatar")
		return
	}

	if h.storage == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured")
		return
	}

	rc, err := h.storage.Get(r.Context(), *user.AvatarURL)
	if err != nil {
		writeError(w, http.StatusNotFound, "avatar not found")
		return
	}
	defer rc.Close()

	// Derive content type from key extension.
	key := *user.AvatarURL
	ct := "image/jpeg"
	if len(key) > 4 {
		switch key[len(key)-3:] {
		case "png":
			ct = "image/png"
		case "gif":
			ct = "image/gif"
		case "ebp":
			ct = "image/webp"
		}
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	io.Copy(w, rc) //nolint:errcheck
}

func bearerToken(r *http.Request) (string, error) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return "", errors.New("missing authorization")
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("invalid authorization")
	}
	return token, nil
}

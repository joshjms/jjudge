package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/store"
	"github.com/jjudge-oj/apiserver/types"
	"golang.org/x/crypto/bcrypt"
)

const defaultTokenTTL = 24 * time.Hour
const defaultUserRole = "user"

// AuthHandler provides JWT authentication endpoints.
type AuthHandler struct {
	userService *services.UserService
	secret      []byte
	tokenTTL    time.Duration
}

// NewAuthHandler constructs an AuthHandler with the provided dependencies.
func NewAuthHandler(userService *services.UserService, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		secret:      []byte(jwtSecret),
		tokenTTL:    defaultTokenTTL,
	}
}

// AuthRouter registers auth routes on the given router.
func AuthRouter(r chi.Router, userService *services.UserService, jwtSecret string) {
	handler := NewAuthHandler(userService, jwtSecret)

	r.Post("/register", handler.Register)
	r.Post("/login", handler.Login)
	r.With(handler.RequireAuth).Get("/me", handler.Me)
}

// RequireAuth enforces JWT authentication and injects the subject into context.
func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return requireAuth(h.secret)(next)
}

// RequireAuth constructs auth middleware for other routers.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return requireAuth([]byte(jwtSecret))
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

	req.Username = strings.TrimSpace(req.Username)
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

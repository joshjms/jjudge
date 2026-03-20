package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/store"
)

// BlogHandler provides blog CRUD endpoints.
type BlogHandler struct {
	blog *services.BlogService
	user *services.UserService
}

func NewBlogHandler(blog *services.BlogService, user *services.UserService) *BlogHandler {
	return &BlogHandler{blog: blog, user: user}
}

// BlogRouter registers blog routes on the given router.
func BlogRouter(r chi.Router, blog *services.BlogService, user *services.UserService, auth, optAuth func(http.Handler) http.Handler) {
	h := NewBlogHandler(blog, user)

	r.With(optAuth).Get("/", h.List)
	r.With(optAuth).Get("/{slug}", h.Get)
	r.With(auth).Post("/", h.Create)
	r.With(auth).Patch("/{slug}", h.Update)
	r.With(auth).Delete("/{slug}", h.Delete)
}

// List returns published posts (public) or all posts for admin/manager.
func (h *BlogHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	publishedOnly := true
	if callerID, err := userIDFromContext(r.Context()); err == nil && callerID > 0 {
		if caller, err := h.user.GetByID(r.Context(), callerID); err == nil && isAdminOrManager(caller.Role) {
			publishedOnly = false
		}
	}

	posts, total, err := h.blog.List(r.Context(), offset, limit, publishedOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list posts")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": posts,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// Get returns a single post by slug.
// Draft posts are only visible to admin/manager or the author.
func (h *BlogHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	post, err := h.blog.Get(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load post")
		return
	}

	if !post.Published {
		callerID, err := userIDFromContext(r.Context())
		if err != nil {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		caller, err := h.user.GetByID(r.Context(), callerID)
		if err != nil || (!isAdminOrManager(caller.Role) && caller.ID != post.AuthorID) {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
	}

	writeJSON(w, http.StatusOK, post)
}

// Create creates a new blog post. Admin/manager only.
func (h *BlogHandler) Create(w http.ResponseWriter, r *http.Request) {
	callerID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	caller, err := h.user.GetByID(r.Context(), callerID)
	if err != nil || !isAdminOrManager(caller.Role) {
		writeError(w, http.StatusForbidden, "admin or manager access required")
		return
	}

	var body struct {
		Title     string   `json:"title"`
		Slug      string   `json:"slug"`
		Content   string   `json:"content"`
		Excerpt   string   `json:"excerpt"`
		Published bool     `json:"published"`
		Tags      []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	body.Title = strings.TrimSpace(body.Title)
	if body.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	slug, err := h.uniqueSlug(r, body.Slug, body.Title, 0)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	if body.Tags == nil {
		body.Tags = []string{}
	}

	post, err := h.blog.Create(r.Context(), types.BlogPost{
		Title:     body.Title,
		Slug:      slug,
		Content:   body.Content,
		Excerpt:   body.Excerpt,
		AuthorID:  callerID,
		Published: body.Published,
		Tags:      body.Tags,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create post")
		return
	}

	writeJSON(w, http.StatusCreated, post)
}

// Update modifies a blog post. Admin/manager or original author may update.
func (h *BlogHandler) Update(w http.ResponseWriter, r *http.Request) {
	urlSlug := chi.URLParam(r, "slug")

	callerID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	caller, err := h.user.GetByID(r.Context(), callerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	post, err := h.blog.Get(r.Context(), urlSlug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load post")
		return
	}

	if !isAdminOrManager(caller.Role) && caller.ID != post.AuthorID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var body struct {
		Title     *string  `json:"title"`
		Slug      *string  `json:"slug"`
		Content   *string  `json:"content"`
		Excerpt   *string  `json:"excerpt"`
		Published *bool    `json:"published"`
		Tags      []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Title != nil {
		post.Title = strings.TrimSpace(*body.Title)
	}
	if body.Content != nil {
		post.Content = *body.Content
	}
	if body.Excerpt != nil {
		post.Excerpt = *body.Excerpt
	}
	if body.Published != nil {
		post.Published = *body.Published
	}
	if body.Tags != nil {
		post.Tags = body.Tags
	}
	if body.Slug != nil {
		newSlug, err := h.uniqueSlug(r, *body.Slug, post.Title, post.ID)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		post.Slug = newSlug
	}

	updated, err := h.blog.Update(r.Context(), post)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update post")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// Delete removes a blog post. Admin/manager only.
func (h *BlogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	callerID, err := userIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	caller, err := h.user.GetByID(r.Context(), callerID)
	if err != nil || !isAdminOrManager(caller.Role) {
		writeError(w, http.StatusForbidden, "admin or manager access required")
		return
	}

	if err := h.blog.Delete(r.Context(), slug); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete post")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func isAdminOrManager(role string) bool {
	r := strings.ToLower(role)
	return r == "admin" || r == "manager"
}

// uniqueSlug generates a slug from input (or title if input is empty) and
// ensures it is unique, appending "-2", "-3" etc. if there is a collision.
// excludeID is the ID of the post being updated (0 for new posts).
func (h *BlogHandler) uniqueSlug(r *http.Request, input, title string, excludeID int) (string, error) {
	base := slugify(input)
	if base == "" {
		base = slugify(title)
	}
	slug := base
	for i := 2; i <= 100; i++ {
		exists, err := h.blog.SlugExists(r.Context(), slug, excludeID)
		if err != nil {
			return "", fmt.Errorf("failed to check slug uniqueness")
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	return "", fmt.Errorf("could not generate a unique slug")
}

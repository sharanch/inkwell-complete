package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/sharanch/inkwell/blog-service/internal/model"
	"github.com/sharanch/inkwell/blog-service/internal/service"
)

type BlogHandler struct {
	svc *service.BlogService
}

func NewBlogHandler(svc *service.BlogService) *BlogHandler {
	return &BlogHandler{svc: svc}
}

func (h *BlogHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	authorID := r.Header.Get("X-User-ID")
	var req model.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	post, err := h.svc.CreatePost(r.Context(), authorID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, post)
}

func (h *BlogHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := r.Header.Get("X-User-ID")

	post, err := h.svc.GetPost(r.Context(), id, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (h *BlogHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	authorID := r.Header.Get("X-User-ID")
	var req model.UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	post, err := h.svc.UpdatePost(r.Context(), id, authorID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (h *BlogHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	authorID := r.Header.Get("X-User-ID")
	if err := h.svc.DeletePost(r.Context(), id, authorID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *BlogHandler) ListMyPosts(w http.ResponseWriter, r *http.Request) {
	authorID := r.Header.Get("X-User-ID")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	q := &model.ListPostsQuery{AuthorID: authorID, Page: page, PageSize: 20}
	result, err := h.svc.ListPosts(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *BlogHandler) ListPublicPosts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	q := &model.ListPostsQuery{
		Visibility: model.VisibilityPublic,
		Tag:        r.URL.Query().Get("tag"),
		Page:       page,
		PageSize:   20,
	}
	result, err := h.svc.ListPosts(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *BlogHandler) ToggleLike(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")
	userID := r.Header.Get("X-User-ID")
	liked, err := h.svc.ToggleLike(r.Context(), postID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"liked": liked})
}

// GetByIDs is an internal endpoint for the feed service.
func (h *BlogHandler) GetByIDs(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil || len(ids) == 0 {
		writeError(w, http.StatusBadRequest, "ids required")
		return
	}
	posts, err := h.svc.GetByIDs(r.Context(), ids)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

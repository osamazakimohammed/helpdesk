package api

import (
	"encoding/json"
	"net/http"

	"helpdesk/internal/db"
	"helpdesk/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type KBHandler struct {
	queries db.Querier
}

func NewKBHandler(queries db.Querier) *KBHandler {
	return &KBHandler{queries: queries}
}

// ListSpaces handles GET /api/v1/kb/spaces
func (h *KBHandler) ListSpaces(w http.ResponseWriter, r *http.Request) {
	spaces, err := h.queries.ListKBSpaces(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch kb spaces"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(spaces)
}

// ListCategoriesBySpace handles GET /api/v1/kb/spaces/{space_id}/categories
func (h *KBHandler) ListCategoriesBySpace(w http.ResponseWriter, r *http.Request) {
	spaceIDStr := chi.URLParam(r, "space_id")
	spaceID, err := types.StringToUUID(spaceIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid space uuid"}`, http.StatusBadRequest)
		return
	}

	cats, err := h.queries.ListKBCategoriesBySpace(r.Context(), spaceID)
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch categories"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cats)
}

// ListArticlesByCategory handles GET /api/v1/kb/categories/{category_id}/articles
func (h *KBHandler) ListArticlesByCategory(w http.ResponseWriter, r *http.Request) {
	catIDStr := chi.URLParam(r, "category_id")
	catID, err := types.StringToUUID(catIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid category uuid"}`, http.StatusBadRequest)
		return
	}

	articles, err := h.queries.ListKBArticlesByCategory(r.Context(), db.ListKBArticlesByCategoryParams{
		CategoryID: catID,
		Column2:    "published",
	})
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"failed to fetch articles"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(articles)
}

// GetArticleBySlug handles GET /api/v1/kb/articles/{slug}
func (h *KBHandler) GetArticleBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, `{"error":"bad_request","message":"slug is required"}`, http.StatusBadRequest)
		return
	}

	article, err := h.queries.GetKBArticleBySlug(r.Context(), slug)
	if err != nil || !article.ID.Valid {
		http.Error(w, `{"error":"not_found","message":"article not found"}`, http.StatusNotFound)
		return
	}

	// Increment view count asynchronously
	go func(id pgtype.UUID) {
		_ = h.queries.IncrementArticleViewCount(r.Context(), id)
	}(article.ID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(article)
}

// SearchArticles handles GET /api/v1/kb/search?q=query
func (h *KBHandler) SearchArticles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
		return
	}

	results, err := h.queries.SearchKBArticles(r.Context(), db.SearchKBArticlesParams{
		WebsearchToTsquery: q,
		Limit:              20,
	})
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"search failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": results})
}

type KBFeedbackReq struct {
	IsHelpful bool   `json:"is_helpful"`
	Comment   string `json:"comment,omitempty"`
}

// ArticleFeedback handles POST /api/v1/kb/articles/{id}/feedback
func (h *KBHandler) ArticleFeedback(w http.ResponseWriter, r *http.Request) {
	articleIDStr := chi.URLParam(r, "id")
	articleID, err := types.StringToUUID(articleIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid article uuid"}`, http.StatusBadRequest)
		return
	}

	var req KBFeedbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid json payload"}`, http.StatusBadRequest)
		return
	}

	feedbackID := types.NewUUIDv7()
	_ = h.queries.RecordArticleFeedback(r.Context(), db.RecordArticleFeedbackParams{
		ID:        feedbackID,
		ArticleID: articleID,
		IsHelpful: req.IsHelpful,
		Comment:   pgtype.Text{String: req.Comment, Valid: req.Comment != ""},
	})

	_ = h.queries.UpdateArticleHelpfulScore(r.Context(), articleID)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success":true}`))
}

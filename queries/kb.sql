-- name: ListKBSpaces :many
SELECT * FROM kb_spaces WHERE deleted_at IS NULL ORDER BY name ASC;

-- name: ListKBCategoriesBySpace :many
SELECT 
    c.*,
    (SELECT count(*) FROM kb_articles a WHERE a.category_id = c.id AND a.status = 'published' AND a.deleted_at IS NULL) AS published_article_count
FROM kb_categories c
WHERE c.space_id = $1 AND c.deleted_at IS NULL
ORDER BY c.sort_order ASC, c.name ASC;

-- name: ListKBArticlesByCategory :many
SELECT 
    id, category_id, slug, title, status, locale, excerpt, author_id, published_at, view_count, helpful_count, unhelpful_count
FROM kb_articles
WHERE category_id = $1 
  AND ($2::text IS NULL OR status = $2)
  AND deleted_at IS NULL
ORDER BY published_at DESC NULLS LAST, title ASC;

-- name: GetKBArticleBySlug :one
SELECT 
    a.id,
    a.category_id,
    a.slug,
    a.title,
    a.status,
    a.locale,
    a.body_html,
    a.body_text,
    a.excerpt,
    a.author_id,
    a.reviewer_id,
    a.published_at,
    a.view_count,
    a.helpful_count,
    a.unhelpful_count,
    a.keywords,
    a.translation_of_id,
    a.created_at,
    a.updated_at,
    a.created_by,
    a.updated_by,
    a.deleted_at,
    c.name AS category_name,
    c.slug AS category_slug,
    s.name AS space_name,
    s.slug AS space_slug
FROM kb_articles a
JOIN kb_categories c ON c.id = a.category_id
JOIN kb_spaces s ON s.id = c.space_id
WHERE a.slug = $1 AND a.deleted_at IS NULL;

-- name: SearchKBArticles :many
SELECT 
    a.id,
    a.category_id,
    a.slug,
    a.title,
    a.excerpt,
    a.status,
    a.locale,
    ts_rank_cd(a.search_vector, websearch_to_tsquery('english', $1)) AS rank,
    c.name AS category_name,
    s.name AS space_name
FROM kb_articles a
JOIN kb_categories c ON c.id = a.category_id
JOIN kb_spaces s ON s.id = c.space_id
WHERE a.status = 'published' 
  AND a.deleted_at IS NULL
  AND a.search_vector @@ websearch_to_tsquery('english', $1)
ORDER BY rank DESC
LIMIT $2;

-- name: IncrementArticleViewCount :exec
UPDATE kb_articles SET view_count = view_count + 1 WHERE kb_articles.id = $1;

-- name: RecordArticleFeedback :exec
INSERT INTO kb_feedback (id, article_id, contact_id, is_helpful, comment)
VALUES ($1, $2, $3, $4, $5);

-- name: UpdateArticleHelpfulScore :exec
UPDATE kb_articles 
SET 
    helpful_count = (SELECT count(*) FROM kb_feedback kbf WHERE kbf.article_id = $1 AND kbf.is_helpful = TRUE),
    unhelpful_count = (SELECT count(*) FROM kb_feedback kbf WHERE kbf.article_id = $1 AND kbf.is_helpful = FALSE)
WHERE kb_articles.id = $1;

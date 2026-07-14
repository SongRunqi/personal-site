package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Article 是通过网页编辑器发布、存在 SQLite 里的文章。
// 仓库里的 Markdown 文件文章(Store)依旧有效,二者在列表接口里合并,
// slug 全局唯一。
type Article struct {
	ID          int64
	Slug        string
	Title       string
	Markdown    string
	HTML        string
	Summary     string
	Tags        []string
	Draft       bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}

func (a *Article) toPost() *Post {
	date := a.CreatedAt
	if a.PublishedAt != nil {
		date = *a.PublishedAt
	}
	return &Post{
		PostMeta: PostMeta{
			Slug: a.Slug, Title: a.Title, Date: date,
			Tags: a.Tags, Summary: a.Summary, Draft: a.Draft, Source: "db",
		},
		HTML: a.HTML,
	}
}

func renderMarkdown(source string) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ---------- 查询 ----------

const articleCols = "id, slug, title, markdown, html, summary, tags, draft, created_at, updated_at, published_at"

func scanArticle(row interface{ Scan(...any) error }) (*Article, error) {
	var a Article
	var tags string
	var published sql.NullTime
	if err := row.Scan(&a.ID, &a.Slug, &a.Title, &a.Markdown, &a.HTML, &a.Summary,
		&tags, &a.Draft, &a.CreatedAt, &a.UpdatedAt, &published); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(tags), &a.Tags)
	if published.Valid {
		t := published.Time
		a.PublishedAt = &t
	}
	return &a, nil
}

func (s *server) articleBySlug(slug string) (*Article, error) {
	return scanArticle(s.db.QueryRow("SELECT "+articleCols+" FROM articles WHERE slug = $1", slug))
}

func (s *server) allArticles() ([]*Article, error) {
	rows, err := s.db.Query("SELECT " + articleCols + " FROM articles ORDER BY COALESCE(published_at, created_at) DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Article
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---------- 文件 + DB 合并视图 ----------

func (s *server) mergedPosts() []PostMeta {
	metas := s.store.Posts()
	for i := range metas {
		metas[i].Source = "file"
	}
	articles, err := s.allArticles()
	if err != nil {
		log.Printf("读取文章失败:%v", err)
	}
	for _, a := range articles {
		if !a.Draft {
			metas = append(metas, a.toPost().PostMeta)
		}
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].Date.After(metas[j].Date) })
	return metas
}

func (s *server) mergedPost(slug string) *Post {
	if a, err := s.articleBySlug(slug); err == nil && !a.Draft {
		return a.toPost()
	}
	return s.store.Post(slug)
}

// postExists:点赞/评论前校验目标文章存在(draft 不算)。
func (s *server) postExists(slug string) bool {
	return s.mergedPost(slug) != nil
}

func (s *server) mergedFullPosts() []*Post {
	posts := s.store.FullPosts()
	articles, err := s.allArticles()
	if err == nil {
		for _, a := range articles {
			if !a.Draft {
				posts = append(posts, a.toPost())
			}
		}
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].Date.After(posts[j].Date) })
	return posts
}

// ---------- 管理接口 ----------

func (s *server) adminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/articles", s.withAdmin(s.handleAdminList))
	mux.HandleFunc("POST /api/admin/articles", s.withAdmin(s.handleAdminCreate))
	mux.HandleFunc("GET /api/admin/articles/{slug}", s.withAdmin(s.handleAdminGet))
	mux.HandleFunc("PUT /api/admin/articles/{slug}", s.withAdmin(s.handleAdminUpdate))
	mux.HandleFunc("DELETE /api/admin/articles/{slug}", s.withAdmin(s.handleAdminDelete))
	mux.HandleFunc("POST /api/admin/preview", s.withAdmin(s.handleAdminPreview))
	mux.HandleFunc("POST /api/admin/upload", s.withAdmin(s.handleAdminUpload))
}

func (s *server) withAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := s.currentUser(r)
		if u == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "请先登录"})
			return
		}
		if !u.IsAdmin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "没有权限"})
			return
		}
		h(w, r)
	}
}

type articleJSON struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Markdown    string   `json:"markdown"`
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
	Draft       bool     `json:"draft"`
	Date        string   `json:"date,omitempty"`
	UpdatedAt   string   `json:"updatedAt,omitempty"`
	PublishedAt string   `json:"publishedAt,omitempty"`
	Source      string   `json:"source,omitempty"`
}

func articleToJSON(a *Article) articleJSON {
	j := articleJSON{
		Slug: a.Slug, Title: a.Title, Markdown: a.Markdown, Summary: a.Summary,
		Tags: a.Tags, Draft: a.Draft, Source: "db",
		Date:      a.CreatedAt.Format(time.RFC3339),
		UpdatedAt: a.UpdatedAt.Format(time.RFC3339),
	}
	if a.PublishedAt != nil {
		j.PublishedAt = a.PublishedAt.Format(time.RFC3339)
	}
	return j
}

// handleAdminList:全部 DB 文章(含草稿)+ 仓库文章(只读,便于总览)。
func (s *server) handleAdminList(w http.ResponseWriter, r *http.Request) {
	articles, err := s.allArticles()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "读取文章失败"})
		return
	}
	out := make([]articleJSON, 0, len(articles))
	for _, a := range articles {
		j := articleToJSON(a)
		j.Markdown = "" // 列表不带正文
		out = append(out, j)
	}
	for _, p := range s.store.Posts() {
		out = append(out, articleJSON{
			Slug: p.Slug, Title: p.Title, Summary: p.Summary, Tags: p.Tags,
			Date: p.Date.Format(time.RFC3339), Source: "file",
		})
	}
	writeJSON(w, 200, out)
}

func (s *server) handleAdminGet(w http.ResponseWriter, r *http.Request) {
	a, err := s.articleBySlug(r.PathValue("slug"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "文章不存在"})
		return
	}
	writeJSON(w, 200, articleToJSON(a))
}

// validateArticle 统一校验 + 渲染 + 摘要,create / update 共用。
func (s *server) validateArticle(in *articleJSON, currentSlug string) (htmlBody string, tagsJSON string, errMsg string) {
	in.Slug = strings.TrimSpace(in.Slug)
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return "", "", "标题不能为空"
	}
	if !slugRe.MatchString(in.Slug) {
		return "", "", "slug 只能是小写字母、数字和连字符,例如 my-first-post"
	}
	// slug 不能与仓库文章冲突;与其他 DB 文章的冲突交给 UNIQUE 约束
	if in.Slug != currentSlug && s.store.Post(in.Slug) != nil {
		return "", "", "slug 与仓库里的文章重复"
	}
	htmlBody, err := renderMarkdown(in.Markdown)
	if err != nil {
		return "", "", "Markdown 渲染失败:" + err.Error()
	}
	in.Summary = strings.TrimSpace(in.Summary)
	if in.Summary == "" {
		in.Summary = excerpt(htmlBody, 160)
	}
	if in.Tags == nil {
		in.Tags = []string{}
	}
	b, _ := json.Marshal(in.Tags)
	return htmlBody, string(b), ""
}

func (s *server) handleAdminCreate(w http.ResponseWriter, r *http.Request) {
	var in articleJSON
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, 400, map[string]string{"error": "请求格式不对"})
		return
	}
	htmlBody, tagsJSON, msg := s.validateArticle(&in, "")
	if msg != "" {
		writeJSON(w, 422, map[string]string{"error": msg})
		return
	}
	now := time.Now().UTC()
	var published *time.Time
	if !in.Draft {
		published = &now
	}
	_, err := s.db.Exec(`
INSERT INTO articles (slug, title, markdown, html, summary, tags, draft, created_at, updated_at, published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		in.Slug, in.Title, in.Markdown, htmlBody, in.Summary, tagsJSON, in.Draft, now, now, published)
	if err != nil {
		if isUniqueViolation(err) {
			writeJSON(w, 422, map[string]string{"error": "slug 已存在"})
			return
		}
		log.Printf("建文章失败:%v", err)
		writeJSON(w, 500, map[string]string{"error": "保存失败"})
		return
	}
	a, _ := s.articleBySlug(in.Slug)
	writeJSON(w, 201, articleToJSON(a))
}

func (s *server) handleAdminUpdate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	cur, err := s.articleBySlug(slug)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "文章不存在"})
		return
	}
	var in articleJSON
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, 400, map[string]string{"error": "请求格式不对"})
		return
	}
	if in.Slug == "" {
		in.Slug = slug
	}
	htmlBody, tagsJSON, msg := s.validateArticle(&in, slug)
	if msg != "" {
		writeJSON(w, 422, map[string]string{"error": msg})
		return
	}
	now := time.Now().UTC()
	// 首次从草稿转公开时记录发布时间
	published := cur.PublishedAt
	if published == nil && !in.Draft {
		published = &now
	}
	_, err = s.db.Exec(`
UPDATE articles SET slug=$1, title=$2, markdown=$3, html=$4, summary=$5, tags=$6, draft=$7, updated_at=$8, published_at=$9
WHERE id=$10`,
		in.Slug, in.Title, in.Markdown, htmlBody, in.Summary, tagsJSON, in.Draft, now, published, cur.ID)
	if err != nil {
		if isUniqueViolation(err) {
			writeJSON(w, 422, map[string]string{"error": "slug 已存在"})
			return
		}
		log.Printf("更新文章失败:%v", err)
		writeJSON(w, 500, map[string]string{"error": "保存失败"})
		return
	}
	// slug 变了,点赞评论跟着走
	if in.Slug != slug {
		s.db.Exec("UPDATE likes SET slug=$1 WHERE slug=$2", in.Slug, slug)
		s.db.Exec("UPDATE comments SET slug=$1 WHERE slug=$2", in.Slug, slug)
	}
	a, _ := s.articleBySlug(in.Slug)
	writeJSON(w, 200, articleToJSON(a))
}

func (s *server) handleAdminDelete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	res, err := s.db.Exec("DELETE FROM articles WHERE slug = $1", slug)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "删除失败"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, 404, map[string]string{"error": "文章不存在"})
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *server) handleAdminPreview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, 400, map[string]string{"error": "请求格式不对"})
		return
	}
	htmlBody, err := renderMarkdown(in.Markdown)
	if err != nil {
		writeJSON(w, 422, map[string]string{"error": fmt.Sprintf("渲染失败:%v", err)})
		return
	}
	writeJSON(w, 200, map[string]string{"html": htmlBody})
}

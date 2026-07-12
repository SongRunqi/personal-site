package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxCommentLen = 2000

func (s *server) socialRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/posts/{slug}/likes", s.handleLikes)
	mux.HandleFunc("POST /api/posts/{slug}/like", s.handleToggleLike)
	mux.HandleFunc("GET /api/posts/{slug}/comments", s.handleComments)
	mux.HandleFunc("POST /api/posts/{slug}/comments", s.handleAddComment)
	mux.HandleFunc("DELETE /api/comments/{id}", s.handleDeleteComment)
}

// ---------- 点赞 ----------

func (s *server) handleLikes(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM likes WHERE slug = ?", slug).Scan(&count); err != nil {
		writeJSON(w, 500, map[string]string{"error": "读取失败"})
		return
	}
	liked := false
	if u := s.currentUser(r); u != nil {
		var n int
		s.db.QueryRow("SELECT COUNT(*) FROM likes WHERE slug = ? AND user_id = ?", slug, u.ID).Scan(&n)
		liked = n > 0
	}
	writeJSON(w, 200, map[string]any{"count": count, "liked": liked})
}

// handleToggleLike:登录用户点一下赞、再点取消。
func (s *server) handleToggleLike(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "登录后才能点赞"})
		return
	}
	slug := r.PathValue("slug")
	if !s.postExists(slug) {
		writeJSON(w, 404, map[string]string{"error": "文章不存在"})
		return
	}
	res, err := s.db.Exec("DELETE FROM likes WHERE slug = ? AND user_id = ?", slug, u.ID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "操作失败"})
		return
	}
	liked := false
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := s.db.Exec("INSERT INTO likes (user_id, slug) VALUES (?, ?)", u.ID, slug); err != nil {
			writeJSON(w, 500, map[string]string{"error": "操作失败"})
			return
		}
		liked = true
	}
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM likes WHERE slug = ?", slug).Scan(&count)
	writeJSON(w, 200, map[string]any{"count": count, "liked": liked})
}

// ---------- 评论 ----------

type commentJSON struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	Author    struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"author"`
	Mine bool `json:"mine"`
}

func (s *server) handleComments(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	// 注意:必须先查当前用户再开 rows——连接池只有一个连接,
	// rows 未关闭时再发查询会死锁。
	var me int64 = -1
	if u := s.currentUser(r); u != nil {
		me = u.ID
		if u.IsAdmin {
			me = -2 // 管理员可删任何评论,前端用 mine 控制删除按钮
		}
	}

	rows, err := s.db.Query(`
SELECT c.id, c.body, c.created_at, c.user_id, u.name, u.avatar_url
FROM comments c JOIN users u ON u.id = c.user_id
WHERE c.slug = ? ORDER BY c.created_at ASC, c.id ASC`, slug)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "读取失败"})
		return
	}
	defer rows.Close()
	out := []commentJSON{}
	for rows.Next() {
		var c commentJSON
		var userID int64
		if err := rows.Scan(&c.ID, &c.Body, &c.CreatedAt, &userID, &c.Author.Name, &c.Author.AvatarURL); err != nil {
			writeJSON(w, 500, map[string]string{"error": "读取失败"})
			return
		}
		c.Mine = me == -2 || userID == me
		out = append(out, c)
	}
	writeJSON(w, 200, out)
}

func (s *server) handleAddComment(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "登录后才能留言"})
		return
	}
	slug := r.PathValue("slug")
	if !s.postExists(slug) {
		writeJSON(w, 404, map[string]string{"error": "文章不存在"})
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, 400, map[string]string{"error": "请求格式不对"})
		return
	}
	body := strings.TrimSpace(in.Body)
	if body == "" {
		writeJSON(w, 422, map[string]string{"error": "留言不能为空"})
		return
	}
	if len([]rune(body)) > maxCommentLen {
		writeJSON(w, 422, map[string]string{"error": "留言太长(最多 2000 字)"})
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec("INSERT INTO comments (slug, user_id, body, created_at) VALUES (?, ?, ?, ?)",
		slug, u.ID, body, now)
	if err != nil {
		log.Printf("写评论失败:%v", err)
		writeJSON(w, 500, map[string]string{"error": "保存失败"})
		return
	}
	id, _ := res.LastInsertId()
	var c commentJSON
	c.ID = id
	c.Body = body
	c.CreatedAt = now
	c.Author.Name = u.Name
	c.Author.AvatarURL = u.AvatarURL
	c.Mine = true
	writeJSON(w, 201, c)
}

// handleDeleteComment:作者本人或管理员可删。
func (s *server) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "请先登录"})
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "无效的评论 ID"})
		return
	}
	var res interface{ RowsAffected() (int64, error) }
	if u.IsAdmin {
		res, err = s.db.Exec("DELETE FROM comments WHERE id = ?", id)
	} else {
		res, err = s.db.Exec("DELETE FROM comments WHERE id = ? AND user_id = ?", id, u.ID)
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "删除失败"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, 404, map[string]string{"error": "评论不存在或没有权限"})
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

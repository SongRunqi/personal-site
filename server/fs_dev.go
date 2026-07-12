//go:build dev

package main

import (
	"io/fs"
	"net/http"
	"os"
	"strings"
)

// 开发模式(go run -tags dev .):内容直读磁盘,每次请求重新扫描;
// 前端由 Vite(:5173)提供。
var autoReload = true

func contentFS() fs.FS {
	return os.DirFS(envOr("CONTENT_DIR", "../content"))
}

func webHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "开发模式:前端请访问 http://localhost:5173", http.StatusNotFound)
	})
}

// registerDevRoutes 提供本地假登录(仅 -tags dev 编译进来,生产二进制没有):
// /auth/dev/login?admin=1&name=测试用户
func registerDevRoutes(s *server, mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/dev/login", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "本地用户"
		}
		email := ""
		if r.URL.Query().Get("admin") == "1" {
			email = envOr("ADMIN_EMAILS", "yitiansong4@gmail.com")
			if i := len(email); i > 0 {
				email = strings.Split(email, ",")[0]
			}
		}
		u, err := s.upsertUser("dev", name, email, name, "")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err := s.createSession(w, u.ID); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})
}

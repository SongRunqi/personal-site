//go:build dev

package main

import (
	"io/fs"
	"net/http"
	"os"
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

package main

import (
	"io/fs"
	"net/http"
	"os"
)

// P1 阶段:内容直读磁盘,每次请求重新扫描;P3 会用 build tag 区分 dev/embed。
var autoReload = true

func contentFS() fs.FS {
	return os.DirFS(envOr("CONTENT_DIR", "../content"))
}

// webHandler 处理非 /api 路径。开发模式下前端由 Vite(:5173)提供,
// 这里只留一个提示;P3 换成 embed 的静态文件 + SPA fallback。
func webHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "开发模式:前端请访问 http://localhost:5173", http.StatusNotFound)
	})
}

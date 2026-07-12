//go:build !dev

package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// 生产模式:前端产物与 content/ 都打进二进制。
// web/dist 与 content 由 make build 拷贝到 server/ 下(见 Makefile)。

//go:embed all:web/dist
var distEmbed embed.FS

//go:embed all:content
var contentEmbed embed.FS

var autoReload = false

// 生产模式没有开发用路由。
func registerDevRoutes(*server, *http.ServeMux) {}

func contentFS() fs.FS {
	sub, err := fs.Sub(contentEmbed, "content")
	if err != nil {
		panic(err)
	}
	return sub
}

// webHandler 先找静态文件,找不到 fallback 到 index.html(SPA 路由)。
func webHandler() http.Handler {
	dist, err := fs.Sub(distEmbed, "web/dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServerFS(dist)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name != "" {
			if f, err := dist.Open(name); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFileFS(w, r, dist, "index.html")
	})
}

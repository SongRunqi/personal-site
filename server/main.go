package main

import (
	"log"
	"net/http"
	"os"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	addr := envOr("ADDR", ":8080")
	baseURL := envOr("SITE_BASE_URL", "http://localhost:8080")
	dataDir := envOr("DATA_DIR", "./data") // 上传图片目录
	databaseURL := envOr("DATABASE_URL",
		"postgres://site:site@localhost:15432/site?sslmode=disable") // 默认对齐 make db 起的本地 PG

	store := NewStore(contentFS())
	store.AutoReload = autoReload
	if err := store.Reload(); err != nil {
		log.Fatalf("加载内容失败:%v", err)
	}

	db, err := openDB(databaseURL)
	if err != nil {
		log.Fatalf("连接数据库失败:%v", err)
	}
	defer db.Close()

	srv := &server{store: store, db: db, baseURL: baseURL, dataDir: dataDir}
	srv.initProviders()

	mux := http.NewServeMux()
	srv.routes(mux)
	registerDevRoutes(srv, mux)
	mux.Handle("/", webHandler())

	log.Printf("listening on %s (base URL %s, data %s)", addr, baseURL, dataDir)
	log.Fatal(http.ListenAndServe(addr, mux))
}

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
	dataDir := envOr("DATA_DIR", "./data")

	store := NewStore(contentFS())
	store.AutoReload = autoReload
	if err := store.Reload(); err != nil {
		log.Fatalf("加载内容失败:%v", err)
	}

	db, err := openDB(dataDir)
	if err != nil {
		log.Fatalf("打开数据库失败:%v", err)
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

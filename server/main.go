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

	store := NewStore(contentFS())
	store.AutoReload = autoReload
	if err := store.Reload(); err != nil {
		log.Fatalf("加载内容失败:%v", err)
	}

	srv := &server{store: store, baseURL: baseURL}
	mux := http.NewServeMux()
	srv.routes(mux)
	mux.Handle("/", webHandler())

	log.Printf("listening on %s (base URL %s)", addr, baseURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}

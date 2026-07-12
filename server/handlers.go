package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type server struct {
	store     *Store
	db        *sql.DB
	baseURL   string
	dataDir   string
	providers map[string]*provider
}

func (s *server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/posts", s.handlePosts)
	mux.HandleFunc("GET /api/posts/{slug}", s.handlePost)
	mux.HandleFunc("GET /api/projects", s.handleProjects)
	mux.HandleFunc("GET /feed.xml", s.handleFeed)
	mux.Handle("GET /uploads/", s.uploadsHandler())
	s.authRoutes(mux)
	s.adminRoutes(mux)
	s.socialRoutes(mux)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("写响应失败:%v", err)
	}
}

func (s *server) handlePosts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.mergedPosts())
}

func (s *server) handlePost(w http.ResponseWriter, r *http.Request) {
	post := s.mergedPost(r.PathValue("slug"))
	if post == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "文章不存在"})
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (s *server) handleProjects(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Projects())
}

package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type server struct {
	store   *Store
	baseURL string
}

func (s *server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/posts", s.handlePosts)
	mux.HandleFunc("GET /api/posts/{slug}", s.handlePost)
	mux.HandleFunc("GET /api/projects", s.handleProjects)
	mux.HandleFunc("GET /feed.xml", s.handleFeed)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("写响应失败:%v", err)
	}
}

func (s *server) handlePosts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Posts())
}

func (s *server) handlePost(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	post := s.store.Post(slug)
	if post == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "文章不存在"})
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (s *server) handleProjects(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Projects())
}

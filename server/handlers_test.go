package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := NewStore(testFS())
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	srv := &server{store: store, baseURL: "https://example.com"}
	mux := http.NewServeMux()
	srv.routes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func getJSON(t *testing.T, url string, v any) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			t.Fatalf("解码 %s: %v", url, err)
		}
	}
	return resp
}

func TestAPIPosts(t *testing.T) {
	ts := newTestServer(t)
	var posts []PostMeta
	getJSON(t, ts.URL+"/api/posts", &posts)
	if len(posts) != 2 {
		t.Fatalf("文章数 = %d", len(posts))
	}
	if posts[0].Title != "第二篇" {
		t.Errorf("第一条应是最新的,得到 %q", posts[0].Title)
	}
}

func TestAPIPost(t *testing.T) {
	ts := newTestServer(t)
	var post Post
	resp := getJSON(t, ts.URL+"/api/posts/first", &post)
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !strings.Contains(post.HTML, "<h2") {
		t.Error("正文应为渲染后的 HTML")
	}
}

func TestAPIPostNotFound(t *testing.T) {
	ts := newTestServer(t)
	resp := getJSON(t, ts.URL+"/api/posts/nope", nil)
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	resp = getJSON(t, ts.URL+"/api/posts/draft", nil)
	if resp.StatusCode != 404 {
		t.Errorf("draft 应 404,得到 %d", resp.StatusCode)
	}
}

func TestAPIProjects(t *testing.T) {
	ts := newTestServer(t)
	var projects []Project
	getJSON(t, ts.URL+"/api/projects", &projects)
	if len(projects) != 1 || projects[0].Name != "onething" {
		t.Errorf("projects 不对:%+v", projects)
	}
}

func TestFeed(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/feed.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, `<rss version="2.0"`) {
		t.Error("不是 RSS 2.0")
	}
	if !strings.Contains(body, "<title>第二篇</title>") {
		t.Error("缺少文章条目")
	}
	if !strings.Contains(body, "https://example.com/blog/second") {
		t.Error("链接应使用 baseURL 拼绝对地址")
	}
	if strings.Contains(body, "草稿") {
		t.Error("draft 不应进 RSS")
	}
}

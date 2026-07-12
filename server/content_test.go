package main

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

const projectsYAML = `projects:
  - name: onething
    tagline: 桌面 AI 助手
    description: 开源桌面 AI 助手
    url: https://github.com/SongRunqi/one-thing
    repo: https://github.com/SongRunqi/one-thing
    stack: [Electron, Vue]
    status: active
`

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"posts/first.md": {Data: []byte(`---
title: 第一篇
date: 2026-01-02
tags: [go, web]
summary: 手写的摘要
draft: false
---

## 小标题

正文内容,带一段 **加粗**。

` + "```go\nfmt.Println(\"hi\")\n```" + `
`)},
		"posts/second.md": {Data: []byte(`---
title: 第二篇
date: 2026-03-04
tags: [随笔]
---

这一篇没有手写摘要,应该从正文里取前一段文字作为摘要。
`)},
		"posts/draft.md": {Data: []byte(`---
title: 草稿
date: 2026-05-06
draft: true
---

还没写完。
`)},
		"posts/broken.md": {Data: []byte("没有 front matter 的坏文件")},
		"projects.yaml":   {Data: []byte(projectsYAML)},
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s := NewStore(testFS())
	if err := s.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	return s
}

func TestFrontMatterFields(t *testing.T) {
	s := newTestStore(t)
	p := s.Post("first")
	if p == nil {
		t.Fatal("找不到 first")
	}
	if p.Title != "第一篇" {
		t.Errorf("title = %q", p.Title)
	}
	want := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !p.Date.Equal(want) {
		t.Errorf("date = %v, want %v", p.Date, want)
	}
	if len(p.Tags) != 2 || p.Tags[0] != "go" || p.Tags[1] != "web" {
		t.Errorf("tags = %v", p.Tags)
	}
	if p.Summary != "手写的摘要" {
		t.Errorf("summary = %q", p.Summary)
	}
}

func TestMarkdownRendering(t *testing.T) {
	s := newTestStore(t)
	p := s.Post("first")
	if p == nil {
		t.Fatal("找不到 first")
	}
	if !strings.Contains(p.HTML, "<h2") {
		t.Error("HTML 里没有 <h2 标题")
	}
	if !strings.Contains(p.HTML, "<strong>加粗</strong>") {
		t.Error("加粗没有渲染")
	}
	if !strings.Contains(p.HTML, "chroma") && !strings.Contains(p.HTML, "background-color") {
		t.Error("代码块似乎没有高亮")
	}
}

func TestSummaryFallback(t *testing.T) {
	s := newTestStore(t)
	p := s.Post("second")
	if p == nil {
		t.Fatal("找不到 second")
	}
	if !strings.HasPrefix(p.Summary, "这一篇没有手写摘要") {
		t.Errorf("summary 应取自正文,得到 %q", p.Summary)
	}
}

func TestSummaryTruncation(t *testing.T) {
	long := strings.Repeat("字", 300)
	post, err := parsePost("long.md", []byte("---\ntitle: 长文\ndate: 2026-01-01\n---\n\n"+long+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := len([]rune(post.Summary)); got != 161 { // 160 字 + 省略号
		t.Errorf("摘要长度 = %d rune,want 161", got)
	}
	if !strings.HasSuffix(post.Summary, "…") {
		t.Error("截断摘要应以 … 结尾")
	}
}

func TestDraftFilteredAndSorted(t *testing.T) {
	s := newTestStore(t)
	posts := s.Posts()
	if len(posts) != 2 {
		t.Fatalf("非 draft 文章应有 2 篇,得到 %d", len(posts))
	}
	if posts[0].Slug != "second" || posts[1].Slug != "first" {
		t.Errorf("应按 date 倒序:%s, %s", posts[0].Slug, posts[1].Slug)
	}
	if s.Post("draft") != nil {
		t.Error("draft 不应能按 slug 取到")
	}
}

func TestBrokenFileSkipped(t *testing.T) {
	// broken.md 解析失败不应让 Reload 报错,其余文章照常可用
	s := newTestStore(t)
	if s.Post("broken") != nil {
		t.Error("坏文件不应出现在索引里")
	}
	if len(s.Posts()) != 2 {
		t.Error("坏文件不应影响其他文章")
	}
}

func TestProjects(t *testing.T) {
	s := newTestStore(t)
	ps := s.Projects()
	if len(ps) != 1 {
		t.Fatalf("projects = %d 个", len(ps))
	}
	if ps[0].Name != "onething" || ps[0].Status != "active" || len(ps[0].Stack) != 2 {
		t.Errorf("项目字段解析不对:%+v", ps[0])
	}
}

func TestDateLayouts(t *testing.T) {
	for _, d := range []string{"2026-07-13", "2026-07-13 09:30", "2026-07-13T09:30:00+08:00"} {
		_, err := parsePost("x.md", []byte("---\ntitle: t\ndate: \""+d+"\"\n---\n\n正文\n"))
		if err != nil {
			t.Errorf("date %q 应能解析:%v", d, err)
		}
	}
}

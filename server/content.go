package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/anchor"
	"gopkg.in/yaml.v3"
)

// PostMeta 是文章列表项(不含正文)。
type PostMeta struct {
	Slug    string    `json:"slug"`
	Title   string    `json:"title"`
	Date    time.Time `json:"date"`
	Tags    []string  `json:"tags"`
	Summary string    `json:"summary"`
	Draft   bool      `json:"-"`
	Source  string    `json:"source,omitempty"` // file(仓库)或 db(网页发布)
}

// Post 是单篇文章,正文为渲染后的 HTML。
type Post struct {
	PostMeta
	HTML string `json:"html"`
}

// Project 对应 projects.yaml 里的一项。
type Project struct {
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Repo        string   `json:"repo"`
	Stack       []string `json:"stack"`
	Status      string   `json:"status"`
}

// Store 在内存里持有全部内容;AutoReload 为 true 时每次读取前重新扫描
// (开发模式直读磁盘,改完文件刷新页面即可见)。
type Store struct {
	fsys       fs.FS
	AutoReload bool

	mu       sync.RWMutex
	posts    []*Post // 按 date 倒序,含 draft
	projects []Project
}

func NewStore(fsys fs.FS) *Store {
	return &Store{fsys: fsys}
}

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle("solarized-light"),
			highlighting.WithFormatOptions(chromahtml.TabWidth(4)),
		),
		&anchor.Extender{Texter: anchor.Text("#")},
	),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

// Reload 扫描 posts/*.md 与 projects.yaml,重建内存索引。
// 单个文件解析失败只记日志、跳过,不影响其余内容。
func (s *Store) Reload() error {
	entries, err := fs.ReadDir(s.fsys, "posts")
	if err != nil {
		return fmt.Errorf("读取 posts 目录: %w", err)
	}

	var posts []*Post
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := fs.ReadFile(s.fsys, path.Join("posts", e.Name()))
		if err != nil {
			log.Printf("读取 %s 失败:%v", e.Name(), err)
			continue
		}
		post, err := parsePost(e.Name(), data)
		if err != nil {
			log.Printf("解析 %s 失败,已跳过:%v", e.Name(), err)
			continue
		}
		posts = append(posts, post)
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].Date.After(posts[j].Date) })

	projects, err := loadProjects(s.fsys)
	if err != nil {
		log.Printf("加载 projects.yaml 失败:%v", err)
		projects = nil
	}

	s.mu.Lock()
	s.posts = posts
	s.projects = projects
	s.mu.Unlock()
	return nil
}

func (s *Store) maybeReload() {
	if s.AutoReload {
		if err := s.Reload(); err != nil {
			log.Printf("重新扫描内容失败:%v", err)
		}
	}
}

// Posts 返回全部非 draft 文章的元数据,按 date 倒序。
func (s *Store) Posts() []PostMeta {
	s.maybeReload()
	s.mu.RLock()
	defer s.mu.RUnlock()
	metas := make([]PostMeta, 0, len(s.posts))
	for _, p := range s.posts {
		if p.Draft {
			continue
		}
		metas = append(metas, p.PostMeta)
	}
	return metas
}

// Post 按 slug 查找非 draft 文章;找不到返回 nil。
func (s *Store) Post(slug string) *Post {
	s.maybeReload()
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.posts {
		if p.Slug == slug && !p.Draft {
			return p
		}
	}
	return nil
}

// FullPosts 返回全部非 draft 文章(含 HTML 正文),供 RSS 使用。
func (s *Store) FullPosts() []*Post {
	s.maybeReload()
	s.mu.RLock()
	defer s.mu.RUnlock()
	full := make([]*Post, 0, len(s.posts))
	for _, p := range s.posts {
		if p.Draft {
			continue
		}
		full = append(full, p)
	}
	return full
}

func (s *Store) Projects() []Project {
	s.maybeReload()
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projects
}

type frontMatter struct {
	Title   string   `yaml:"title"`
	Date    string   `yaml:"date"`
	Tags    []string `yaml:"tags"`
	Draft   bool     `yaml:"draft"`
	Summary string   `yaml:"summary"`
}

var dateLayouts = []string{"2006-01-02", "2006-01-02 15:04", time.RFC3339}

func parsePost(filename string, data []byte) (*Post, error) {
	fm, body, err := splitFrontMatter(data)
	if err != nil {
		return nil, err
	}

	var meta frontMatter
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		return nil, fmt.Errorf("front matter 不是合法 YAML: %w", err)
	}
	if meta.Title == "" {
		return nil, fmt.Errorf("front matter 缺少 title")
	}

	var date time.Time
	for _, layout := range dateLayouts {
		if d, err := time.Parse(layout, meta.Date); err == nil {
			date = d
			break
		}
	}
	if date.IsZero() {
		return nil, fmt.Errorf("date %q 无法解析(支持 2006-01-02 / 2006-01-02 15:04 / RFC3339)", meta.Date)
	}

	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return nil, fmt.Errorf("markdown 渲染失败: %w", err)
	}
	htmlBody := buf.String()

	summary := strings.TrimSpace(meta.Summary)
	if summary == "" {
		summary = excerpt(htmlBody, 160)
	}

	return &Post{
		PostMeta: PostMeta{
			Slug:    strings.TrimSuffix(filename, ".md"),
			Title:   meta.Title,
			Date:    date,
			Tags:    meta.Tags,
			Summary: summary,
			Draft:   meta.Draft,
		},
		HTML: htmlBody,
	}, nil
}

func splitFrontMatter(data []byte) (fm, body []byte, err error) {
	const delim = "---"
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, delim+"\n") {
		return nil, nil, fmt.Errorf("缺少 YAML front matter(需以 --- 开头)")
	}
	rest := text[len(delim)+1:]
	idx := strings.Index(rest, "\n"+delim+"\n")
	if idx < 0 {
		return nil, nil, fmt.Errorf("front matter 未闭合(找不到结尾 ---)")
	}
	return []byte(rest[:idx]), []byte(rest[idx+len(delim)+2:]), nil
}

var (
	tagRe   = regexp.MustCompile(`<[^>]*>`)
	spaceRe = regexp.MustCompile(`\s+`)
)

// excerpt 从渲染后的 HTML 提取纯文本前 n 个字符作为摘要。
func excerpt(htmlBody string, n int) string {
	text := tagRe.ReplaceAllString(htmlBody, " ")
	text = html2text(text)
	text = strings.TrimSpace(spaceRe.ReplaceAllString(text, " "))
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n]) + "…"
}

func html2text(s string) string {
	r := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&nbsp;", " ")
	return r.Replace(s)
}

func loadProjects(fsys fs.FS) ([]Project, error) {
	data, err := fs.ReadFile(fsys, "projects.yaml")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Projects []Project `yaml:"projects"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc.Projects, nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

// doJSON 发一个可带会话 cookie 的 JSON 请求。
func doJSON(t *testing.T, method, url string, cookie *http.Cookie, body any, out any) *http.Response {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, rd)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("解码 %s %s: %v", method, url, err)
		}
	}
	return resp
}

func TestMe(t *testing.T) {
	s, ts := newTestEnv(t)
	var out struct{ User *User }
	doJSON(t, "GET", ts.URL+"/api/me", nil, nil, &out)
	if out.User != nil {
		t.Error("未登录时 user 应为 null")
	}
	c := sessionCookieFor(t, s, "张三", false)
	doJSON(t, "GET", ts.URL+"/api/me", c, nil, &out)
	if out.User == nil || out.User.Name != "张三" || out.User.IsAdmin {
		t.Errorf("me = %+v", out.User)
	}
}

func TestArticleLifecycle(t *testing.T) {
	s, ts := newTestEnv(t)
	admin := sessionCookieFor(t, s, "站长", true)
	visitor := sessionCookieFor(t, s, "路人", false)

	article := map[string]any{
		"slug": "web-post", "title": "网页发的文章",
		"markdown": "## 一节\n\n有图:![图](/uploads/2026/07/x.png)\n", "tags": []string{"web"}, "draft": true,
	}
	// 权限:未登录 401,非管理员 403
	if r := doJSON(t, "POST", ts.URL+"/api/admin/articles", nil, article, nil); r.StatusCode != 401 {
		t.Fatalf("未登录建文章 = %d", r.StatusCode)
	}
	if r := doJSON(t, "POST", ts.URL+"/api/admin/articles", visitor, article, nil); r.StatusCode != 403 {
		t.Fatalf("非管理员建文章 = %d", r.StatusCode)
	}
	if r := doJSON(t, "POST", ts.URL+"/api/admin/articles", admin, article, nil); r.StatusCode != 201 {
		t.Fatalf("建文章 = %d", r.StatusCode)
	}

	// 草稿不出现在公开列表
	var posts []PostMeta
	doJSON(t, "GET", ts.URL+"/api/posts", nil, nil, &posts)
	for _, p := range posts {
		if p.Slug == "web-post" {
			t.Fatal("草稿不应出现在 /api/posts")
		}
	}

	// 发布
	article["draft"] = false
	if r := doJSON(t, "PUT", ts.URL+"/api/admin/articles/web-post", admin, article, nil); r.StatusCode != 200 {
		t.Fatalf("发布 = %d", r.StatusCode)
	}
	doJSON(t, "GET", ts.URL+"/api/posts", nil, nil, &posts)
	found := false
	for _, p := range posts {
		if p.Slug == "web-post" && p.Source == "db" {
			found = true
		}
	}
	if !found {
		t.Fatal("发布后应出现在 /api/posts")
	}
	var post Post
	doJSON(t, "GET", ts.URL+"/api/posts/web-post", nil, nil, &post)
	if !strings.Contains(post.HTML, "<h2") || !strings.Contains(post.HTML, "<img") {
		t.Errorf("正文渲染不对:%q", post.HTML)
	}

	// slug 不能撞仓库文章
	bad := map[string]any{"slug": "first", "title": "撞车", "markdown": "x", "draft": false}
	if r := doJSON(t, "POST", ts.URL+"/api/admin/articles", admin, bad, nil); r.StatusCode != 422 {
		t.Errorf("撞仓库 slug = %d, want 422", r.StatusCode)
	}

	// 删除
	if r := doJSON(t, "DELETE", ts.URL+"/api/admin/articles/web-post", admin, nil, nil); r.StatusCode != 200 {
		t.Fatalf("删除 = %d", r.StatusCode)
	}
	if r := doJSON(t, "GET", ts.URL+"/api/posts/web-post", nil, nil, nil); r.StatusCode != 404 {
		t.Errorf("删除后 = %d, want 404", r.StatusCode)
	}
}

func TestPreview(t *testing.T) {
	s, ts := newTestEnv(t)
	admin := sessionCookieFor(t, s, "站长", true)
	var out struct{ HTML string `json:"html"` }
	r := doJSON(t, "POST", ts.URL+"/api/admin/preview", admin, map[string]string{"markdown": "**粗**"}, &out)
	if r.StatusCode != 200 || !strings.Contains(out.HTML, "<strong>粗</strong>") {
		t.Errorf("preview = %d, %q", r.StatusCode, out.HTML)
	}
}

func TestLikes(t *testing.T) {
	s, ts := newTestEnv(t)
	u1 := sessionCookieFor(t, s, "甲", false)
	u2 := sessionCookieFor(t, s, "乙", false)

	if r := doJSON(t, "POST", ts.URL+"/api/posts/first/like", nil, nil, nil); r.StatusCode != 401 {
		t.Fatalf("未登录点赞 = %d", r.StatusCode)
	}
	if r := doJSON(t, "POST", ts.URL+"/api/posts/nope/like", u1, nil, nil); r.StatusCode != 404 {
		t.Fatalf("给不存在的文章点赞 = %d", r.StatusCode)
	}

	var st struct {
		Count int  `json:"count"`
		Liked bool `json:"liked"`
	}
	doJSON(t, "POST", ts.URL+"/api/posts/first/like", u1, nil, &st)
	if st.Count != 1 || !st.Liked {
		t.Fatalf("点赞后 = %+v", st)
	}
	doJSON(t, "POST", ts.URL+"/api/posts/first/like", u2, nil, &st)
	if st.Count != 2 {
		t.Fatalf("两人点赞 = %+v", st)
	}
	// 再点取消
	doJSON(t, "POST", ts.URL+"/api/posts/first/like", u1, nil, &st)
	if st.Count != 1 || st.Liked {
		t.Fatalf("取消后 = %+v", st)
	}
	// 未登录也能看数量
	doJSON(t, "GET", ts.URL+"/api/posts/first/likes", nil, nil, &st)
	if st.Count != 1 || st.Liked {
		t.Fatalf("匿名查看 = %+v", st)
	}
}

func TestComments(t *testing.T) {
	s, ts := newTestEnv(t)
	u1 := sessionCookieFor(t, s, "甲", false)
	u2 := sessionCookieFor(t, s, "乙", false)
	admin := sessionCookieFor(t, s, "站长", true)

	if r := doJSON(t, "POST", ts.URL+"/api/posts/first/comments", nil, map[string]string{"body": "x"}, nil); r.StatusCode != 401 {
		t.Fatalf("未登录评论 = %d", r.StatusCode)
	}
	if r := doJSON(t, "POST", ts.URL+"/api/posts/first/comments", u1, map[string]string{"body": "  "}, nil); r.StatusCode != 422 {
		t.Fatalf("空评论 = %d", r.StatusCode)
	}

	var c commentJSON
	doJSON(t, "POST", ts.URL+"/api/posts/first/comments", u1, map[string]string{"body": "写得好"}, &c)
	if c.ID == 0 || c.Author.Name != "甲" || !c.Mine {
		t.Fatalf("评论 = %+v", c)
	}

	var list []commentJSON
	doJSON(t, "GET", ts.URL+"/api/posts/first/comments", u2, nil, &list)
	if len(list) != 1 || list[0].Mine {
		t.Fatalf("乙看到 = %+v", list)
	}

	// 乙删不了甲的评论;管理员可以
	url := fmt.Sprintf("%s/api/comments/%d", ts.URL, c.ID)
	if r := doJSON(t, "DELETE", url, u2, nil, nil); r.StatusCode != 404 {
		t.Fatalf("乙删甲 = %d", r.StatusCode)
	}
	if r := doJSON(t, "DELETE", url, admin, nil, nil); r.StatusCode != 200 {
		t.Fatalf("管理员删 = %d", r.StatusCode)
	}
	doJSON(t, "GET", ts.URL+"/api/posts/first/comments", nil, nil, &list)
	if len(list) != 0 {
		t.Fatalf("删除后还剩 %d 条", len(list))
	}
}

func TestUpload(t *testing.T) {
	s, ts := newTestEnv(t)
	admin := sessionCookieFor(t, s, "站长", true)

	// 最小合法 PNG 头,足够 DetectContentType 识别
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "截图 2026.png")
	fw.Write(png)
	mw.Close()

	req, _ := http.NewRequest("POST", ts.URL+"/api/admin/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(admin)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct{ URL string `json:"url"` }
	json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != 201 || !strings.HasPrefix(out.URL, "/uploads/") || !strings.HasSuffix(out.URL, ".png") {
		t.Fatalf("upload = %d %q", resp.StatusCode, out.URL)
	}
	// 传回来的 URL 能取到
	if r, err := http.Get(ts.URL + out.URL); err != nil || r.StatusCode != 200 {
		t.Fatalf("取上传图片失败:%v %d", err, r.StatusCode)
	}

	// 非图片被拒
	buf.Reset()
	mw = multipart.NewWriter(&buf)
	fw, _ = mw.CreateFormFile("file", "evil.html")
	fw.Write([]byte("<script>alert(1)</script>"))
	mw.Close()
	req, _ = http.NewRequest("POST", ts.URL+"/api/admin/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(admin)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 422 {
		t.Errorf("非图片 = %d, want 422", resp2.StatusCode)
	}
}

func TestFeedIncludesArticles(t *testing.T) {
	s, ts := newTestEnv(t)
	admin := sessionCookieFor(t, s, "站长", true)
	a := map[string]any{"slug": "from-web", "title": "网页文章进 RSS", "markdown": "正文", "draft": false}
	if r := doJSON(t, "POST", ts.URL+"/api/admin/articles", admin, a, nil); r.StatusCode != 201 {
		t.Fatalf("建文章 = %d", r.StatusCode)
	}
	resp, err := http.Get(ts.URL + "/feed.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b := new(strings.Builder)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	if !strings.Contains(b.String(), "网页文章进 RSS") {
		t.Error("RSS 里没有网页发布的文章")
	}
}

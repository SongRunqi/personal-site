package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	sessionCookie = "site_session"
	stateCookie   = "oauth_state"
	sessionTTL    = 30 * 24 * time.Hour
)

type User struct {
	ID        int64  `json:"id"`
	Provider  string `json:"provider"`
	Email     string `json:"-"` // 邮箱不对外暴露
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
	IsAdmin   bool   `json:"isAdmin"`
}

type provider struct {
	name      string
	config    *oauth2.Config
	fetchUser func(ctx context.Context, client *http.Client) (id, email, name, avatar string, err error)
}

// initProviders 装配 OAuth 提供方。本站只开 GitHub,且回调里只放行站长
// 本人(见 handleOAuthCallback);未配置时登录页按钮置灰。
func (s *server) initProviders() {
	s.providers = map[string]*provider{}

	if id, secret := envOr("GITHUB_CLIENT_ID", ""), envOr("GITHUB_CLIENT_SECRET", ""); id != "" && secret != "" {
		s.providers["github"] = &provider{
			name: "github",
			config: &oauth2.Config{
				ClientID:     id,
				ClientSecret: secret,
				Endpoint:     github.Endpoint,
				RedirectURL:  s.baseURL + "/auth/github/callback",
				Scopes:       []string{"read:user", "user:email"},
			},
			fetchUser: fetchGitHubUser,
		}
	}
}

func fetchGitHubUser(ctx context.Context, client *http.Client) (id, email, name, avatar string, err error) {
	var v struct {
		ID     int64  `json:"id"`
		Login  string `json:"login"`
		Name   string `json:"name"`
		Avatar string `json:"avatar_url"`
		Email  string `json:"email"`
	}
	if err = getJSONAs(ctx, client, "https://api.github.com/user", &v); err != nil {
		return
	}
	email = v.Email
	if email == "" {
		var emails []struct {
			Email   string `json:"email"`
			Primary bool   `json:"primary"`
		}
		if err2 := getJSONAs(ctx, client, "https://api.github.com/user/emails", &emails); err2 == nil {
			for _, e := range emails {
				if e.Primary {
					email = e.Email
					break
				}
			}
		}
	}
	name = v.Name
	if name == "" {
		name = v.Login
	}
	// GitHub 用 login 作为 provider_id 的补充展示,但唯一键用数字 ID
	return fmt.Sprintf("%d|%s", v.ID, v.Login), email, name, v.Avatar, nil
}

func getJSONAs(ctx context.Context, client *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s: %d %s", url, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (s *server) secureCookies() bool {
	return strings.HasPrefix(s.baseURL, "https://")
}

// ---------- HTTP handlers ----------

func (s *server) authRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/{provider}/login", s.handleOAuthLogin)
	mux.HandleFunc("GET /auth/{provider}/callback", s.handleOAuthCallback)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/me", s.handleMe)
	mux.HandleFunc("GET /api/auth/providers", s.handleProviders)
}

func (s *server) handleProviders(w http.ResponseWriter, r *http.Request) {
	type p struct {
		Name       string `json:"name"`
		Configured bool   `json:"configured"`
	}
	out := []p{
		{"github", s.providers["github"] != nil},
	}
	writeJSON(w, 200, out)
}

func (s *server) handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	prov := s.providers[r.PathValue("provider")]
	if prov == nil {
		http.Error(w, "该登录方式未配置", http.StatusNotFound)
		return
	}
	state := randomToken()
	returnTo := sanitizeReturnTo(r.URL.Query().Get("return_to"))
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Value: state + "|" + returnTo, Path: "/",
		MaxAge: 600, HttpOnly: true, Secure: s.secureCookies(), SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, prov.config.AuthCodeURL(state), http.StatusFound)
}

// sanitizeReturnTo 只允许站内相对路径,防开放跳转。
func sanitizeReturnTo(p string) string {
	if p == "" || !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") {
		return "/"
	}
	return p
}

func (s *server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	prov := s.providers[r.PathValue("provider")]
	if prov == nil {
		http.Error(w, "该登录方式未配置", http.StatusNotFound)
		return
	}
	c, err := r.Cookie(stateCookie)
	if err != nil {
		http.Error(w, "登录会话已过期,请重新登录", http.StatusBadRequest)
		return
	}
	state, returnTo, _ := strings.Cut(c.Value, "|")
	if state == "" || r.URL.Query().Get("state") != state {
		http.Error(w, "state 校验失败,请重新登录", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Value: "", Path: "/", MaxAge: -1})

	ctx := r.Context()
	tok, err := prov.config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("%s 换取 token 失败:%v", prov.name, err)
		http.Error(w, "登录失败,请重试", http.StatusBadGateway)
		return
	}
	pid, email, name, avatar, err := prov.fetchUser(ctx, prov.config.Client(ctx, tok))
	if err != nil {
		log.Printf("%s 获取用户信息失败:%v", prov.name, err)
		http.Error(w, "获取用户信息失败,请重试", http.StatusBadGateway)
		return
	}

	// 本站只有站长本人可以登录:身份不匹配就不建用户、不发会话。
	if !s.isAdminIdentity(prov.name, pid, email) {
		log.Printf("拒绝非站长登录:%s %s", prov.name, pid)
		http.Redirect(w, r, "/login?error=owner-only", http.StatusFound)
		return
	}

	user, err := s.upsertUser(prov.name, pid, email, name, avatar)
	if err != nil {
		log.Printf("写入用户失败:%v", err)
		http.Error(w, "登录失败,请重试", http.StatusInternalServerError)
		return
	}
	if err := s.createSession(w, user.ID); err != nil {
		log.Printf("创建会话失败:%v", err)
		http.Error(w, "登录失败,请重试", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, sanitizeReturnTo(returnTo), http.StatusFound)
}

func (s *server) upsertUser(providerName, pid, email, name, avatar string) (*User, error) {
	isAdmin := s.isAdminIdentity(providerName, pid, email)
	_, err := s.db.Exec(`
INSERT INTO users (provider, provider_id, email, name, avatar_url, is_admin)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (provider, provider_id) DO UPDATE SET
	email = excluded.email, name = excluded.name,
	avatar_url = excluded.avatar_url, is_admin = excluded.is_admin`,
		providerName, pid, email, name, avatar, isAdmin)
	if err != nil {
		return nil, err
	}
	return s.userBy("provider = $1 AND provider_id = $2", providerName, pid)
}

// isAdminIdentity:邮箱在 ADMIN_EMAILS 里,或 GitHub 登录名在 ADMIN_GITHUB_LOGINS 里。
func (s *server) isAdminIdentity(providerName, pid, email string) bool {
	for _, e := range strings.Split(envOr("ADMIN_EMAILS", "yitiansong4@gmail.com"), ",") {
		if e = strings.TrimSpace(e); e != "" && strings.EqualFold(e, email) {
			return true
		}
	}
	if providerName == "github" {
		_, login, _ := strings.Cut(pid, "|")
		for _, l := range strings.Split(envOr("ADMIN_GITHUB_LOGINS", "SongRunqi"), ",") {
			if l = strings.TrimSpace(l); l != "" && strings.EqualFold(l, login) {
				return true
			}
		}
	}
	return false
}

func (s *server) userBy(where string, args ...any) (*User, error) {
	var u User
	err := s.db.QueryRow(
		"SELECT id, provider, email, name, avatar_url, is_admin FROM users WHERE "+where, args...).
		Scan(&u.ID, &u.Provider, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *server) createSession(w http.ResponseWriter, userID int64) error {
	token := randomToken()
	expires := time.Now().Add(sessionTTL)
	if _, err := s.db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)",
		token, userID, expires); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/",
		Expires: expires, HttpOnly: true, Secure: s.secureCookies(), SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// currentUser 从会话 cookie 取当前用户;未登录返回 nil。
func (s *server) currentUser(r *http.Request) *User {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	var userID int64
	var expires time.Time
	err = s.db.QueryRow("SELECT user_id, expires_at FROM sessions WHERE token = $1", c.Value).
		Scan(&userID, &expires)
	if err != nil {
		return nil
	}
	if time.Now().After(expires) {
		s.db.Exec("DELETE FROM sessions WHERE token = $1", c.Value)
		return nil
	}
	u, err := s.userBy("id = $1", userID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("查用户失败:%v", err)
		}
		return nil
	}
	return u
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.db.Exec("DELETE FROM sessions WHERE token = $1", c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]*User{"user": s.currentUser(r)})
}

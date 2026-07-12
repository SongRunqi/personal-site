package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// openDB 打开(必要时创建)DATA_DIR 下的 SQLite 库并跑迁移。
// 动态数据(用户、文章、点赞、评论、上传图片)都放 DATA_DIR,
// 与「内容即文件」的仓库文章互不干扰。
func openDB(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录: %w", err)
	}
	dsn := filepath.Join(dataDir, "site.db") +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// modernc/sqlite 单写者:串行化连接,避免并发写报 SQLITE_BUSY
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
	id          INTEGER PRIMARY KEY,
	provider    TEXT NOT NULL,
	provider_id TEXT NOT NULL,
	email       TEXT NOT NULL DEFAULT '',
	name        TEXT NOT NULL DEFAULT '',
	avatar_url  TEXT NOT NULL DEFAULT '',
	is_admin    INTEGER NOT NULL DEFAULT 0,
	created_at  TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE (provider, provider_id)
);

CREATE TABLE IF NOT EXISTS sessions (
	token      TEXT PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS articles (
	id           INTEGER PRIMARY KEY,
	slug         TEXT NOT NULL UNIQUE,
	title        TEXT NOT NULL,
	markdown     TEXT NOT NULL,
	html         TEXT NOT NULL,
	summary      TEXT NOT NULL DEFAULT '',
	tags         TEXT NOT NULL DEFAULT '[]',
	draft        INTEGER NOT NULL DEFAULT 1,
	created_at   TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at   TEXT NOT NULL DEFAULT (datetime('now')),
	published_at TEXT
);

CREATE TABLE IF NOT EXISTS likes (
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	slug       TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (user_id, slug)
);

CREATE TABLE IF NOT EXISTS comments (
	id         INTEGER PRIMARY KEY,
	slug       TEXT NOT NULL,
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	body       TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_comments_slug ON comments(slug, created_at);
CREATE INDEX IF NOT EXISTS idx_likes_slug ON likes(slug);
`)
	return err
}

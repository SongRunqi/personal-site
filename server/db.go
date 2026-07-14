package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// openDB 连接 PostgreSQL 并跑迁移。compose 启动时 PG 可能还没就绪,
// 这里带重试(最多 30 秒)。
func openDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		err = db.Ping()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			db.Close()
			return nil, fmt.Errorf("连接 PostgreSQL 超时: %w", err)
		}
		log.Printf("等待 PostgreSQL 就绪…(%v)", err)
		time.Sleep(time.Second)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// isUniqueViolation 判断是否撞了唯一约束(如 slug 重复)。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
	id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	provider    TEXT NOT NULL,
	provider_id TEXT NOT NULL,
	email       TEXT NOT NULL DEFAULT '',
	name        TEXT NOT NULL DEFAULT '',
	avatar_url  TEXT NOT NULL DEFAULT '',
	is_admin    BOOLEAN NOT NULL DEFAULT FALSE,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (provider, provider_id)
);

CREATE TABLE IF NOT EXISTS sessions (
	token      TEXT PRIMARY KEY,
	user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS articles (
	id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	slug         TEXT NOT NULL UNIQUE,
	title        TEXT NOT NULL,
	markdown     TEXT NOT NULL,
	html         TEXT NOT NULL,
	summary      TEXT NOT NULL DEFAULT '',
	tags         TEXT NOT NULL DEFAULT '[]',
	draft        BOOLEAN NOT NULL DEFAULT TRUE,
	created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
	published_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS likes (
	user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	slug       TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (user_id, slug)
);

CREATE TABLE IF NOT EXISTS comments (
	id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	slug       TEXT NOT NULL,
	user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	body       TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_comments_slug ON comments(slug, created_at);
CREATE INDEX IF NOT EXISTS idx_likes_slug ON likes(slug);
`)
	return err
}

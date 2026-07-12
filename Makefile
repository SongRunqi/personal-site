# 个人网站 一键命令
# make dev    并行起 Go(:8080)与 Vite(:5173)
# make build  vite build → 拷贝产物 → 单二进制 bin/site
# make test   go test + tsc
# make lint   go vet + eslint

.PHONY: dev dev-server dev-web build test lint clean

dev:
	@$(MAKE) -j2 dev-server dev-web

dev-server:
	cd server && go run -tags dev .

dev-web:
	cd web && bun run dev

build:
	cd web && bun install --frozen-lockfile && bun run build
	rm -rf server/web/dist
	mkdir -p server/web
	cp -R web/dist server/web/dist
	cd server && go build -o ../bin/site .
	@echo "✓ 构建完成:bin/site"

test:
	cd server && go test ./...
	cd web && bunx tsc --noEmit

lint:
	cd server && go vet ./...
	cd web && bun run lint

clean:
	rm -rf bin server/web/dist web/dist

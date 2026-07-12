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
	rm -rf server/web/dist server/content
	mkdir -p server/web
	cp -R web/dist server/web/dist
	cp -R content server/content
	cd server && go build -o ../bin/site .
	@echo "✓ 构建完成:bin/site"

# -tags dev:测试不依赖 embed 产物(server/web/dist、server/content)
test:
	cd server && go test -tags dev ./...
	cd web && bunx tsc --noEmit

lint:
	cd server && go vet -tags dev ./...
	cd web && bun run lint

clean:
	rm -rf bin server/web/dist server/content web/dist

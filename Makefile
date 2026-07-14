# 个人网站 一键命令
# make dev    起本地 PG 容器 + Go(:8080)+ Vite(:5173)
# make build  vite build → 拷贝产物 → 单二进制 bin/site
# make test   起临时 PG 容器跑 go test + tsc
# make lint   go vet + eslint

PG_IMAGE = postgres:16-alpine
TEST_PG  = site-pg-test

.PHONY: dev dev-server dev-web db build test test-server test-web lint clean

# 本地开发数据库(数据存 docker 卷 site-pg-data,删容器不丢)
# 用 15432 端口,避开本机可能已有的 PostgreSQL
db:
	@docker start site-pg >/dev/null 2>&1 || docker run -d --name site-pg \
		-e POSTGRES_USER=site -e POSTGRES_PASSWORD=site -e POSTGRES_DB=site \
		-p 127.0.0.1:15432:5432 -v site-pg-data:/var/lib/postgresql/data $(PG_IMAGE) >/dev/null
	@until docker exec site-pg pg_isready -U site -q 2>/dev/null; do sleep 0.3; done
	@echo "✓ 开发数据库就绪(localhost:15432,库/用户/密码均为 site)"

dev: db
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

# -tags dev:测试不依赖 embed 产物;DB 测试跑在一次性的 PG 容器里
test: test-server test-web

test-server:
	@docker rm -f $(TEST_PG) >/dev/null 2>&1 || true
	@docker run -d --rm --name $(TEST_PG) \
		-e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=test \
		-p 127.0.0.1:55432:5432 $(PG_IMAGE) >/dev/null
	@until docker exec $(TEST_PG) pg_isready -U test -q 2>/dev/null; do sleep 0.3; done
	@cd server && TEST_DATABASE_URL="postgres://test:test@localhost:55432/test?sslmode=disable" \
		go test -tags dev -timeout 120s ./...; s=$$?; \
		docker rm -f $(TEST_PG) >/dev/null 2>&1; exit $$s

test-web:
	cd web && bunx tsc --noEmit

lint:
	cd server && go vet -tags dev ./...
	cd web && bun run lint

clean:
	rm -rf bin server/web/dist server/content web/dist

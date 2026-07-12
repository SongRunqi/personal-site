# 多阶段构建:bun 构建前端 → go 构建单二进制 → distroless 运行
# 构建:docker build -t personal-site .
# 运行:docker run -d -p 8080:8080 -e SITE_BASE_URL=https://<你的域名> personal-site

# ---- 前端 ----
FROM oven/bun:1 AS web
WORKDIR /app
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web ./
RUN bun run build

# ---- 后端 ----
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY server/go.mod server/go.sum ./server/
RUN cd server && go mod download
COPY server ./server
COPY content ./server/content
COPY --from=web /app/dist ./server/web/dist
RUN cd server && CGO_ENABLED=0 go build -ldflags="-s -w" -o /site . && mkdir /empty

# ---- 运行 ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /site /site
ENV ADDR=:8080
# 动态数据(SQLite、上传图片)挂到卷上,升级镜像不丢数据。
# 预建 /data 并归 nonroot,否则新卷是 root 的、进程写不进去。
COPY --from=build --chown=nonroot:nonroot /empty /data
ENV DATA_DIR=/data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/site"]

# 多阶段构建:bun 构建前端 → go 构建单二进制 → scratch 运行
# 运行层用 scratch(空镜像):不用访问 gcr.io 等国内/腾讯云连不上的仓库,
# 只依赖 docker.io 的 bun 和 golang 两个镜像。
# 部署:docker compose up -d --build(见 docker-compose.yml)

# ---- 前端 ----
FROM oven/bun:1 AS web
WORKDIR /app
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web ./
RUN bun run build

# ---- 后端 ----
FROM golang:1.25-alpine AS build
# GitHub OAuth 的出站 HTTPS 需要 CA 证书,连同二进制一起拷进 scratch
RUN apk add --no-cache ca-certificates
# 国内服务器连不上 proxy.golang.org,换国内镜像
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY server/go.mod server/go.sum ./server/
RUN cd server && go mod download
COPY server ./server
COPY content ./server/content
COPY --from=web /app/dist ./server/web/dist
RUN cd server && CGO_ENABLED=0 go build -ldflags="-s -w" -o /site . && mkdir /empty

# ---- 运行 ----
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /site /site
# 预建 /data(上传图片卷)并归非 root 用户,否则新卷进程写不进去
COPY --from=build --chown=65532:65532 /empty /data
ENV ADDR=:8080
ENV DATA_DIR=/data
USER 65532:65532
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/site"]

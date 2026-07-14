# 部署指南

目标环境:腾讯云香港轻量应用服务器(免备案)。域名还没买,下文一律用
`example.com` 占位,买好后全局替换即可。

架构:**Docker Compose 两个容器**——`app`(Go 单二进制,前端已 embed)+
`db`(PostgreSQL 16)。文章、点赞、评论、用户会话都在 PG 里;上传的图片在
`uploads` 卷里。发文章直接在网页后台写(`/write`),不需要 git push。

登录只开 GitHub,且**只有站长本人的账号能登录**(`ADMIN_GITHUB_LOGINS`
白名单,其他账号会在 OAuth 回调时被拒绝)。

## 0. GitHub OAuth App(做一次)

GitHub → Settings → Developer settings → OAuth Apps → New OAuth App:

- Homepage URL:`https://example.com`
- Authorization callback URL:`https://example.com/auth/github/callback`

拿到 Client ID,再 Generate a new client secret。

## 1. 服务器上装 Docker(做一次)

```bash
curl -fsSL https://get.docker.com | sh
```

## 2. 拉代码、配环境、起服务

```bash
git clone https://github.com/SongRunqi/personal-site.git ~/personal-site
cd ~/personal-site
cp .env.example .env
vim .env        # 填 SITE_BASE_URL、POSTGRES_PASSWORD、GitHub OAuth 两项
docker compose up -d --build
curl -s 127.0.0.1:8080/api/posts | head -c 200   # 验证
```

`.env` 各项:

| 变量 | 说明 |
|---|---|
| `SITE_BASE_URL` | `https://example.com`(RSS 链接、OAuth 回调都用它拼) |
| `POSTGRES_PASSWORD` | PG 密码,起服务前定好;换密码要同时改卷里的库 |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | 上面 OAuth App 的两个值 |
| `ADMIN_GITHUB_LOGINS` | 允许登录的 GitHub 用户名,默认 `SongRunqi` |
| `ADMIN_EMAILS` | 备用的站长邮箱白名单 |

## 3. 反代 + HTTPS(Caddy)

```bash
sudo apt install -y caddy
```

`/etc/caddy/Caddyfile`:

```caddyfile
example.com {
    encode gzip
    reverse_proxy 127.0.0.1:8080
}
```

```bash
sudo systemctl reload caddy
```

DNS 把 A 记录指到服务器公网 IP;控制台防火墙放行 80、443。

## 4. 日常使用

**发文章**:浏览器打开 `https://example.com/login` → 用 GitHub 登录 →
导航栏出现「写文章」→ 在 `/write` 管理、`/write/new` 写作。图片直接
粘贴/拖进编辑器,支持实时预览、快捷键(⌘S 存草稿、⌘Enter 发布)、
本地自动备份。**不再需要本地写 md 后 git push**(仓库 `content/posts/`
里的旧文章依然会显示,想改它们才需要动仓库)。

**更新代码**:

```bash
cd ~/personal-site && git pull && docker compose up -d --build
```

**备份**(PG 转储 + 图片卷):

```bash
docker compose exec db pg_dump -U site site | gzip > ~/site-db-$(date +%F).sql.gz
docker run --rm -v personal-site_uploads:/data -v ~:/backup alpine \
  tar czf /backup/site-uploads-$(date +%F).tgz -C /data .
```

**恢复**:`gunzip -c xxx.sql.gz | docker compose exec -T db psql -U site site`,
图片解包回卷即可。

## 5. 常用检查

```bash
docker compose ps                 # 两个容器都应是 healthy/running
docker compose logs -f app        # 应用日志(拒绝登录、渲染失败都在这)
docker compose logs db | tail     # PG 日志
curl -s https://example.com/feed.xml | head   # RSS
```

## 本地开发

`make dev` 会自动起一个本地 PG 容器(`site-pg`,映射到 15432 端口以避开
本机已有的 PostgreSQL,数据在 docker 卷里),再并行起 Go(:8080)和
Vite(:5173)。浏览器访问 :5173,
`/auth/dev/login?admin=1` 可假登录成站长(只编译进 dev 版)。
`make test` 用一次性 PG 容器跑测试,跑完即删。

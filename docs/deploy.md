# 部署指南

目标环境:腾讯云香港轻量应用服务器(免备案),与 onething 官网同一套思路。
域名还没买,下文一律用 `example.com` 占位,买好后全局替换即可。

整站是**一个二进制**(前端产物和 content/ 都 embed 在里面),部署没有任何
运行时依赖:没有 Node、没有数据库,扔一个文件上去就能跑。

配置只有两个环境变量:

| 变量 | 默认值 | 说明 |
|---|---|---|
| `ADDR` | `:8080` | 监听地址 |
| `SITE_BASE_URL` | `http://localhost:8080` | RSS 里绝对链接的前缀,生产填 `https://example.com` |

## 方式一:systemd 直跑二进制(推荐,最省事)

轻量服务器 1C1G 也绰绰有余。

### 1. 服务器上准备构建环境(只做一次)

```bash
# Go 1.25+
sudo apt install -y make git
curl -fsSL https://go.dev/dl/go1.25.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
# bun
curl -fsSL https://bun.sh/install | bash
```

### 2. 拉代码并构建

```bash
git clone https://github.com/SongRunqi/personal-site.git ~/personal-site
cd ~/personal-site && make build   # 产出 bin/site
sudo install -m 755 bin/site /usr/local/bin/personal-site
```

### 3. systemd 服务

`/etc/systemd/system/personal-site.service`:

```ini
[Unit]
Description=personal site
After=network.target

[Service]
ExecStart=/usr/local/bin/personal-site
Environment=ADDR=127.0.0.1:8080
Environment=SITE_BASE_URL=https://example.com
Restart=always
RestartSec=3
User=www-data
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now personal-site
curl -s 127.0.0.1:8080/api/posts | head -c 200   # 验证
```

注意 `ADDR=127.0.0.1:8080`:只监听本机,对外统一走反代。

## 方式二:Docker

服务器上只需要 Docker,不用装 Go / bun:

```bash
cd ~/personal-site
docker build -t personal-site .
docker run -d --name site --restart always \
  -p 127.0.0.1:8080:8080 \
  -e SITE_BASE_URL=https://example.com \
  personal-site
```

## 反代 + HTTPS

推荐 Caddy:自动申请、续期 Let's Encrypt 证书,配置只有三行。

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

域名 DNS 把 A 记录指到服务器公网 IP,等生效后 `https://example.com` 就通了。
(用 Nginx 也行,`proxy_pass http://127.0.0.1:8080` + certbot,常规做法,不赘述。)

轻量服务器记得在控制台防火墙放行 80、443 端口。

## 发一篇新文章

内容就是仓库里的文件,所以发布 = 提交 + 服务器重建:

```bash
# 本地
vim content/posts/my-new-post.md   # 写文章(带 front matter)
git add . && git commit -m "新文章" && git push

# 服务器(systemd 方式)
cd ~/personal-site && git pull && make build \
  && sudo install -m 755 bin/site /usr/local/bin/personal-site \
  && sudo systemctl restart personal-site

# 服务器(Docker 方式)
cd ~/personal-site && git pull \
  && docker build -t personal-site . \
  && docker rm -f site \
  && docker run -d --name site --restart always \
       -p 127.0.0.1:8080:8080 -e SITE_BASE_URL=https://example.com personal-site
```

想更省事,可以把服务器那几行存成 `~/redeploy.sh`,本地
`git push && ssh 服务器 ./redeploy.sh` 一条龙;以后有需要再上 GitHub
Actions 自动化,现阶段手动跑一下足够了。

## 常用检查

```bash
systemctl status personal-site      # 进程状态
journalctl -u personal-site -f      # 日志(解析失败的文章会在这里报)
curl -s https://example.com/feed.xml | head   # RSS
```

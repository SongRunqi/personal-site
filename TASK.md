# 任务:个人网站(Projects + 博客)

> 本文档是交给 Claude Code 的实施任务书。在本目录(`~/data/code/personal-site`)
> 下从零搭建我的个人网站。请先通读全文,按阶段推进;每个阶段完成后跑一遍
> 该阶段的验收命令再进入下一阶段。

## 一、我是谁 / 网站是什么

我是一名独立开发者(宋一天,GitHub: SongRunqi),网站的作用:

1. **首页** — 一句话介绍我 + 精选项目 + 最新博客。
2. **Projects** — 我的项目列表,当前有两个,以后会增加:
   - **onething(一事)** — 开源桌面 AI 助手(Electron + Vue),多模型接入、
     本地工具执行。仓库:https://github.com/SongRunqi/one-thing
   - **TransReader** — macOS 菜单栏翻译工具(Python)。
3. **博客** — Markdown 写作,支持代码高亮、中文排版;要有 RSS。
4. **关于** — 简介 + 联系方式(GitHub、邮箱 yitiansong4@gmail.com)。

语言:站点内容以中文为主。

## 二、技术栈与总体架构(定死,不要换)

- **后端:Go**(1.22+,标准库 `net/http` 的新版路由即可,不引重框架;
  Markdown 渲染用 `goldmark`,前置元数据用 YAML front matter)。
- **前端:React**(Vite + React 18 + TypeScript)。
- **内容即文件**:博客文章和项目数据都是仓库里的文件,不用数据库。
- **单二进制部署**:前端 `vite build` 产物通过 `embed.FS` 打进 Go 二进制,
  `go build` 出一个文件就能跑。

```
personal-site/
├── TASK.md                  # 本文档
├── content/
│   ├── posts/               # 博客文章 *.md(YAML front matter:title/date/tags/draft)
│   │   └── hello-world.md   # 种子文章,验证全链路
│   └── projects.yaml        # 项目数据(名称/一句话/链接/技术栈/状态)
├── server/                  # Go 后端(module: github.com/SongRunqi/personal-site)
│   ├── main.go
│   ├── content.go           # 启动时扫描 content/,内存索引,开发模式下监听变更
│   ├── handlers.go          # JSON API
│   ├── rss.go               # /feed.xml
│   └── web_embed.go         # //go:embed 前端产物 + SPA fallback
├── web/                     # React 前端(Vite + TS)
│   └── src/
│       ├── pages/           # Home / Projects / Blog / Post / About
│       └── ...
├── Makefile                 # dev / build / test 一键命令
└── Dockerfile               # 可选,部署用
```

**API 设计**(全部 `/api` 前缀,返回 JSON):

| 路由 | 说明 |
|---|---|
| `GET /api/posts` | 文章列表(不含正文;draft 的不返回) |
| `GET /api/posts/{slug}` | 单篇,正文为渲染后的 HTML(goldmark,开 GFM + 代码高亮) |
| `GET /api/projects` | 项目列表(读 projects.yaml) |
| `GET /feed.xml` | RSS 2.0 |
| 其余路径 | 静态文件 / SPA fallback 到 index.html |

**开发模式**:`make dev` 同时起 Go(:8080,直读 content/ 与磁盘文件)和
Vite(:5173,proxy `/api` → :8080)。生产模式只有 :8080 一个进程。

## 三、阶段划分(总览)

| 阶段 | 内容 | 验收 |
|---|---|---|
| P0 | 脚手架:go module、Vite 工程、Makefile、种子内容 | `make dev` 两端都起来 |
| P1 | Go 后端:content 索引、三个 API、RSS | `go test ./...` + curl 三个接口 |
| P2 | React 前端:5 个页面 + 路由 + 视觉 | 浏览器走通全部页面 |
| P3 | 打包一体化:embed 前端、单二进制、Dockerfile | 单二进制起服务,页面可用 |
| P4 | 部署准备:构建产物、部署文档 | 见 P4 说明 |

以下是各阶段细节。

## P0 脚手架

- `server/`:`go mod init github.com/SongRunqi/personal-site`。
- `web/`:`bunx create-vite@latest web --template react-ts`(本机有 bun,
  前端包管理统一用 bun)。
- `Makefile`:`make dev`(并行起两端)、`make build`、`make test`、`make lint`。
- 种子内容:`content/posts/hello-world.md`(带完整 front matter 示例)、
  `content/projects.yaml`(填入上面两个真实项目)。
- `.gitignore`、每阶段完成后做一次 git commit(中文 commit message 即可)。

## P1 Go 后端

- `content.go`:启动扫描 `content/posts/*.md`,解析 front matter
  (title、date、tags、draft、可选 summary;无 summary 则取正文前 160 字),
  slug = 文件名去扩展名。按 date 倒序。文件解析失败要报错日志但不崩服务。
- goldmark 配置:GFM、标题锚点、代码高亮(chroma,主题选一个浅色的)。
- `projects.yaml` 字段:`name, tagline, description, url, repo, stack[], status`。
- RSS:最近 20 篇,绝对链接的 base URL 从环境变量 `SITE_BASE_URL` 读,
  默认 `http://localhost:8080`。
- 测试:content 解析(front matter 各字段、draft 过滤、排序)、API happy path。

## P2 React 前端

页面:`/`(首页)、`/projects`、`/blog`、`/blog/:slug`、`/about`。
路由用 react-router。数据获取用原生 fetch + 简单 hooks 即可,不要引重型状态库。

**视觉方向(重要)**:延续我一贯的「纸墨画线风」——温暖纸底
(#F7F4EC 一系)、墨色文字、朱砂红点缀、楷体/宋体标题 + 无衬线正文、
手画感的分隔线与下划线。参考物:我在 onething 项目里做的官网
(`~/data/code/start-electron/site/index.html`,可以直接读它抄 token)。
不要做成通用的现代模板风(大渐变、玻璃拟态一律不要)。
正文排版认真做:中文行高 1.75 左右、合适的 measure(约 65ch)、
代码块横向滚动不撑破版心。响应式到手机可读。

## P3 打包一体化

- `web_embed.go`:`//go:embed dist` 前端产物;非 `/api` 路径先找静态文件,
  找不到 fallback 到 index.html;`content/` 生产模式同样 embed
  (或用 `-tags dev` 区分磁盘/embed 两种读取,选实现简单的)。
- `make build`:vite build → 拷贝产物 → `go build -o bin/site`。
- 验收:`./bin/site` 单进程起服务,五个页面 + RSS 全部可用。

## P4 部署准备(只做准备,不实际部署)

目标环境是腾讯云香港轻量服务器(免备案,和我另一个项目 onething 的
官网同一套思路)。本阶段产出:

- `Dockerfile`(多阶段:bun 构建前端 → go 构建 → distroless/alpine 运行)。
- `docs/deploy.md`:轻量服务器上 systemd 或 docker 跑法、Caddy/Nginx
  反代 + HTTPS、以及「发一篇新文章 = git push + 服务器重新构建/重启」的流程。
- 域名我还没买,文档里用占位符。

## 四、工作约定

- 计划呈现方式:先给全部阶段的总览,再逐阶段展开细节;不要挤牙膏式分批。
- 每阶段结束:跑测试/验收命令,git commit,再进下一阶段。
- 有真正需要我拍板的事(如域名、配色之外的大方向)再问我;
  常规技术选型按本文档执行,不要重新设计。

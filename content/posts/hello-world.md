---
title: 你好,世界
date: 2026-07-13
tags: [随笔, 建站]
draft: false
summary: 这个网站搭起来了。说说它是怎么搭的,以及我打算在这里写什么。
---

这个网站终于搭起来了。

## 为什么要自己搭

现成的博客平台很多,但我想要一个完全属于自己的角落:内容是仓库里的
Markdown 文件,样式是自己一笔一笔画的,部署是一个单文件二进制。
没有数据库,没有后台,发一篇文章就是 `git push`。

## 技术栈

- **后端**:Go,标准库 `net/http` 路由,Markdown 用 goldmark 渲染;
- **前端**:React + Vite + TypeScript;
- **部署**:前端产物用 `embed.FS` 打进 Go 二进制,一个文件跑天下。

代码高亮也是有的:

```go
func main() {
	fmt.Println("你好,世界")
}
```

## 打算写什么

项目开发记录、踩坑笔记,偶尔一点随想。目前手上有两个项目:
[onething](https://github.com/SongRunqi/one-thing)(开源桌面 AI 助手)
和 TransReader(macOS 菜单栏翻译工具),它们的开发笔记会陆续放上来。

欢迎常来。

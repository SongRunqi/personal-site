import { useEffect } from 'react'

export default function About() {
  useEffect(() => {
    document.title = '关于 · 宋一天'
  }, [])

  return (
    <div className="wrap about">
      <header className="page-head">
        <h1>关于</h1>
      </header>
      <div className="about-intro">
        <span className="seal-stamp" aria-hidden="true">
          一天
        </span>
        <div>
          <p>
            我是宋一天,一名独立开发者。一个人做产品:从写代码、画界面,到写文档、
            做官网,整条链路自己跑通。正在做的是开源桌面 AI 助手 onething(一事)
            和 macOS 菜单栏翻译工具 TransReader。
          </p>
          <p>
            这个网站也是自己写的:Go 后端加 React 前端,文章是仓库里的 Markdown
            文件,整站编译成一个二进制文件跑在服务器上。喜欢这种什么都摸得到底的感觉。
          </p>
        </div>
      </div>
      <ul className="contact-list">
        <li>
          <span className="label">GitHub</span>
          <a href="https://github.com/SongRunqi" target="_blank" rel="noopener noreferrer">
            github.com/SongRunqi
          </a>
        </li>
        <li>
          <span className="label">邮箱</span>
          <a href="mailto:yitiansong4@gmail.com">yitiansong4@gmail.com</a>
        </li>
        <li>
          <span className="label">订阅</span>
          <a href="/feed.xml">RSS(/feed.xml)</a>
        </li>
      </ul>
    </div>
  )
}

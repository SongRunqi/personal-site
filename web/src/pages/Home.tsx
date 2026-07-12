import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useFetch, formatDate, type PostMeta, type Project } from '../api'
import { ProjectCard } from './Projects'

export default function Home() {
  useEffect(() => {
    document.title = '宋一天 · 独立开发者'
  }, [])

  const posts = useFetch<PostMeta[]>('/api/posts')
  const projects = useFetch<Project[]>('/api/projects')

  return (
    <>
      <header className="wrap hero">
        <div>
          <p className="hero-eyebrow fade-up">独立开发者 · 写代码,也写字</p>
          <h1 className="fade-up d1">
            一天做好
            <span className="stroke">
              一件事
              <svg viewBox="0 0 200 24" aria-hidden="true">
                <path d="M4 14 C 40 20, 80 8, 118 13 S 180 18, 196 11" />
              </svg>
            </span>
          </h1>
          <p className="lede fade-up d2">
            我是宋一天(
            <a href="https://github.com/SongRunqi" target="_blank" rel="noopener noreferrer">
              SongRunqi
            </a>
            ),一个人做产品:开源桌面 AI 助手 <em>onething</em>、菜单栏翻译工具
            TransReader。做的过程里踩过的坑、想明白的事,都写在这里。
          </p>
          <div className="hero-actions fade-up d3">
            <Link className="btn-primary" to="/projects">
              看看项目
            </Link>
            <Link className="btn-ghost" to="/blog">
              读读博客 →
            </Link>
          </div>
        </div>
        <div className="motto fade-up d2" aria-hidden="true">
          <span className="motto-text">日拱一卒 功不唐捐</span>
          <span className="seal-stamp">
            一天
          </span>
        </div>
      </header>

      <section className="wrap section">
        <div className="section-head">
          <h2>手上的项目</h2>
          <div className="rule"></div>
          <Link className="tag" to="/projects" style={{ textDecoration: 'none' }}>
            全部项目 →
          </Link>
        </div>
        {projects.loading && <p className="state-note">加载中…</p>}
        {projects.error && <p className="state-note">项目列表加载失败,稍后再试。</p>}
        {projects.data && (
          <div className="project-list">
            {projects.data.slice(0, 2).map((p) => (
              <ProjectCard key={p.name} project={p} />
            ))}
          </div>
        )}
      </section>

      <section className="wrap section">
        <div className="section-head">
          <h2>最近在写</h2>
          <div className="rule"></div>
          <Link className="tag" to="/blog" style={{ textDecoration: 'none' }}>
            全部文章 →
          </Link>
        </div>
        {posts.loading && <p className="state-note">加载中…</p>}
        {posts.error && <p className="state-note">文章列表加载失败,稍后再试。</p>}
        {posts.data && posts.data.length === 0 && (
          <p className="state-note">还没有文章,第一篇正在路上。</p>
        )}
        {posts.data && (
          <ul className="post-list">
            {posts.data.slice(0, 3).map((p) => (
              <li key={p.slug}>
                <span className="post-item-date">{formatDate(p.date)}</span>
                <br />
                <Link className="post-item-title" to={`/blog/${p.slug}`}>
                  {p.title}
                </Link>
                <p className="post-item-summary">{p.summary}</p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  )
}

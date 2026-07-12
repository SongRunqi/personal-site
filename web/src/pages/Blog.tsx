import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useFetch, formatDate, type PostMeta } from '../api'

export default function Blog() {
  useEffect(() => {
    document.title = '博客 · 宋一天'
  }, [])

  const { data, error, loading } = useFetch<PostMeta[]>('/api/posts')

  return (
    <div className="wrap">
      <header className="page-head">
        <h1>博客</h1>
        <p>
          开发笔记、踩坑记录,偶尔一点随想。也可以用 <a href="/feed.xml">RSS</a> 订阅。
        </p>
      </header>
      <section className="section" style={{ borderTop: 'none', paddingTop: 24 }}>
        {loading && <p className="state-note">加载中…</p>}
        {error && <p className="state-note">文章列表加载失败,稍后再试。</p>}
        {data && data.length === 0 && <p className="state-note">还没有文章,第一篇正在路上。</p>}
        {data && (
          <ul className="post-list">
            {data.map((p) => (
              <li key={p.slug}>
                <span className="post-item-date">{formatDate(p.date)}</span>
                {p.tags && p.tags.length > 0 && (
                  <span className="post-tags">
                    {p.tags.map((t) => (
                      <span key={t}>{t}</span>
                    ))}
                  </span>
                )}
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
    </div>
  )
}

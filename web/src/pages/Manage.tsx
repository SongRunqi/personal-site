import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { requestJSON, formatDate, type ArticleDraft } from '../api'
import { useAuth } from '../auth'

// 文章管理页(/write):所有文章一览,新建、编辑、删除的入口。
export default function Manage() {
  const { user, loading } = useAuth()
  const [items, setItems] = useState<ArticleDraft[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    document.title = '文章管理 · 宋一天'
  }, [])

  useEffect(() => {
    if (!user?.isAdmin) return
    requestJSON<ArticleDraft[]>('/api/admin/articles')
      .then(setItems)
      .catch((e) => setError((e as Error).message))
  }, [user])

  async function remove(slug: string, title: string) {
    if (!window.confirm(`确定删除《${title}》?点赞和留言会一并失效。`)) return
    try {
      await requestJSON(`/api/admin/articles/${slug}`, { method: 'DELETE' })
      setItems(items.filter((a) => a.slug !== slug))
    } catch (e) {
      setError((e as Error).message)
    }
  }

  if (loading) return <div className="wrap state-note">加载中…</div>
  if (!user) {
    return (
      <div className="wrap state-note">
        写文章需要先<Link to="/login?return_to=/write">登录</Link>。
      </div>
    )
  }
  if (!user.isAdmin) {
    return <div className="wrap state-note">这个页面只有站长能用。</div>
  }

  const drafts = items.filter((a) => a.source !== 'file' && a.draft)
  const published = items.filter((a) => a.source !== 'file' && !a.draft)
  const filePosts = items.filter((a) => a.source === 'file')

  const row = (a: ArticleDraft) => (
    <li key={a.slug}>
      <span className={`manage-status ${a.source === 'file' ? 'file' : a.draft ? 'draft' : 'pub'}`}>
        {a.source === 'file' ? '仓库' : a.draft ? '草稿' : '已发布'}
      </span>
      <span className="manage-title">{a.title}</span>
      {a.date && <span className="manage-date">{formatDate(a.date)}</span>}
      {a.source === 'db' ? (
        <span className="manage-ops">
          {!a.draft && <Link to={`/blog/${a.slug}`}>查看</Link>}
          <Link to={`/write/${a.slug}`}>编辑</Link>
          <button type="button" onClick={() => remove(a.slug, a.title)}>
            删除
          </button>
        </span>
      ) : (
        <span className="manage-note">在 content/posts/ 里改</span>
      )}
    </li>
  )

  return (
    <div className="wrap manage-page">
      <header className="write-head">
        <h1>文章管理</h1>
        <div className="write-actions">
          <Link className="btn-primary btn-small" to="/write/new">
            写新文章
          </Link>
        </div>
      </header>
      {error && <p className="form-error">{error}</p>}

      {drafts.length > 0 && (
        <section className="manage-group">
          <div className="section-head">
            <h2>草稿</h2>
            <div className="rule"></div>
            <span className="tag">{drafts.length} 篇</span>
          </div>
          <ul className="manage-list">{drafts.map(row)}</ul>
        </section>
      )}

      <section className="manage-group">
        <div className="section-head">
          <h2>已发布</h2>
          <div className="rule"></div>
          <span className="tag">{published.length} 篇</span>
        </div>
        {published.length > 0 ? (
          <ul className="manage-list">{published.map(row)}</ul>
        ) : (
          <p className="state-note">还没有发布过文章,点右上角「写新文章」开始。</p>
        )}
      </section>

      {filePosts.length > 0 && (
        <section className="manage-group">
          <div className="section-head">
            <h2>仓库文章</h2>
            <div className="rule"></div>
            <span className="tag">只读</span>
          </div>
          <ul className="manage-list">{filePosts.map(row)}</ul>
        </section>
      )}
    </div>
  )
}

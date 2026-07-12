import { useEffect, useState, type FormEvent } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { requestJSON, type CommentItem } from '../api'
import { useAuth } from '../auth'

function fmtTime(iso: string): string {
  const d = new Date(iso)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

function Avatar({ name, url }: { name: string; url: string }) {
  if (url) {
    return <img className="comment-avatar" src={url} alt="" referrerPolicy="no-referrer" />
  }
  return <span className="comment-avatar fallback">{name.slice(0, 1)}</span>
}

export default function Comments({ slug }: { slug: string }) {
  const { user } = useAuth()
  const location = useLocation()
  const [comments, setComments] = useState<CommentItem[]>([])
  const [body, setBody] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    requestJSON<CommentItem[]>(`/api/posts/${slug}/comments`)
      .then(setComments)
      .catch(() => {})
  }, [slug, user])

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!body.trim() || busy) return
    setBusy(true)
    setError('')
    try {
      const c = await requestJSON<CommentItem>(`/api/posts/${slug}/comments`, {
        method: 'POST',
        body: JSON.stringify({ body }),
      })
      setComments([...comments, c])
      setBody('')
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function remove(id: number) {
    try {
      await requestJSON(`/api/comments/${id}`, { method: 'DELETE' })
      setComments(comments.filter((c) => c.id !== id))
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <section className="comments">
      <div className="section-head">
        <h2>留言</h2>
        <div className="rule"></div>
        <span className="tag">{comments.length > 0 ? `${comments.length} 条` : '还没有留言'}</span>
      </div>

      {comments.length > 0 && (
        <ul className="comment-list">
          {comments.map((c) => (
            <li key={c.id}>
              <Avatar name={c.author.name} url={c.author.avatarUrl} />
              <div className="comment-main">
                <div className="comment-head">
                  <span className="comment-name">{c.author.name}</span>
                  <span className="comment-time">{fmtTime(c.createdAt)}</span>
                  {c.mine && (
                    <button type="button" className="comment-del" onClick={() => remove(c.id)}>
                      删除
                    </button>
                  )}
                </div>
                <p className="comment-body">{c.body}</p>
              </div>
            </li>
          ))}
        </ul>
      )}

      {user ? (
        <form className="comment-form" onSubmit={submit}>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={4}
            maxLength={2000}
            placeholder="写点什么…"
          />
          {error && <p className="form-error">{error}</p>}
          <div className="comment-form-foot">
            <span className="comment-as">以 {user.name} 的身份留言</span>
            <button type="submit" className="btn-primary btn-small" disabled={busy || !body.trim()}>
              留言
            </button>
          </div>
        </form>
      ) : (
        <p className="comment-login-note">
          <Link to={`/login?return_to=${encodeURIComponent(location.pathname)}`}>登录</Link>
          后可以留言。
        </p>
      )}
    </section>
  )
}

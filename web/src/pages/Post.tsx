import { useEffect } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useFetch, formatDate, type Post as PostData } from '../api'
import { useAuth } from '../auth'
import Likes from '../components/Likes'
import Comments from '../components/Comments'

export default function Post() {
  const { slug } = useParams<{ slug: string }>()
  const { user } = useAuth()
  const { data, error, loading } = useFetch<PostData>(`/api/posts/${slug}`)

  useEffect(() => {
    document.title = data ? `${data.title} · 宋一天` : '博客 · 宋一天'
  }, [data])

  return (
    <div className="wrap">
      <article className="post-page">
        <Link className="post-back" to="/blog">
          ← 回博客目录
        </Link>
        {loading && <p className="state-note">加载中…</p>}
        {error && (
          <p className="state-note">
            {error === 'HTTP 404' ? '没有这篇文章。' : '文章加载失败,稍后再试。'}
          </p>
        )}
        {data && (
          <>
            <h1 className="post-title">{data.title}</h1>
            <p className="post-meta">
              {formatDate(data.date)}
              {data.tags && data.tags.length > 0 && (
                <span className="post-tags">
                  {data.tags.map((t) => (
                    <span key={t}>{t}</span>
                  ))}
                </span>
              )}
              {user?.isAdmin && data.source === 'db' && (
                <Link className="post-edit" to={`/write/${data.slug}`}>
                  编辑
                </Link>
              )}
            </p>
            <div className="prose" dangerouslySetInnerHTML={{ __html: data.html }} />
            {slug && (
              <>
                <Likes slug={slug} />
                <Comments slug={slug} />
              </>
            )}
          </>
        )}
      </article>
    </div>
  )
}

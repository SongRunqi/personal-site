import { useEffect, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { requestJSON } from '../api'
import { useAuth } from '../auth'

interface LikeState {
  count: number
  liked: boolean
}

export default function Likes({ slug }: { slug: string }) {
  const { user } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [state, setState] = useState<LikeState>({ count: 0, liked: false })
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    requestJSON<LikeState>(`/api/posts/${slug}/likes`)
      .then(setState)
      .catch(() => {})
  }, [slug])

  async function toggle() {
    if (!user) {
      navigate(`/login?return_to=${encodeURIComponent(location.pathname)}`)
      return
    }
    if (busy) return
    setBusy(true)
    try {
      setState(await requestJSON<LikeState>(`/api/posts/${slug}/like`, { method: 'POST' }))
    } catch {
      // 失败保持原状
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="like-row">
      <button
        type="button"
        className={`like-stamp${state.liked ? ' liked' : ''}`}
        onClick={toggle}
        aria-pressed={state.liked}
        title={user ? (state.liked ? '取消赞' : '赞一个') : '登录后点赞'}
      >
        赞
      </button>
      <span className="like-count">
        {state.count > 0 ? `${state.count} 人觉得不错` : '觉得不错?盖个章'}
      </span>
    </div>
  )
}

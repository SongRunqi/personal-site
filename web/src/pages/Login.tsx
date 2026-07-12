import { useEffect } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { useFetch } from '../api'
import { useAuth } from '../auth'

interface ProviderInfo {
  name: string
  configured: boolean
}

const providerLabel: Record<string, { label: string; mark: string }> = {
  google: { label: '用 Google 登录', mark: 'G' },
  github: { label: '用 GitHub 登录', mark: 'GH' },
}

export default function Login() {
  useEffect(() => {
    document.title = '登录 · 宋一天'
  }, [])

  const { user } = useAuth()
  const [params] = useSearchParams()
  const returnTo = params.get('return_to') ?? '/'
  const { data, loading } = useFetch<ProviderInfo[]>('/api/auth/providers')

  if (user) {
    return <Navigate to={returnTo} replace />
  }

  return (
    <div className="wrap login-page">
      <header className="page-head">
        <h1>登录</h1>
        <p>登录后可以点赞、留言。只取头像和昵称,不往你的账号里写任何东西。</p>
      </header>
      <div className="login-buttons">
        {loading && <p className="state-note">加载中…</p>}
        {data?.map((p) => {
          const meta = providerLabel[p.name] ?? { label: p.name, mark: '' }
          return p.configured ? (
            <a
              key={p.name}
              className="login-btn"
              href={`/auth/${p.name}/login?return_to=${encodeURIComponent(returnTo)}`}
            >
              <span className="login-mark">{meta.mark}</span>
              {meta.label}
            </a>
          ) : (
            <span key={p.name} className="login-btn disabled" title="该登录方式尚未配置">
              <span className="login-mark">{meta.mark}</span>
              {meta.label}(未配置)
            </span>
          )
        })}
      </div>
      <p className="login-note">
        登录即表示同意本站将你的公开昵称与头像用于展示点赞和留言。
      </p>
    </div>
  )
}

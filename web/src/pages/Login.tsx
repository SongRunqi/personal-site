import { useEffect } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { useFetch } from '../api'
import { useAuth } from '../auth'

interface ProviderInfo {
  name: string
  configured: boolean
}

export default function Login() {
  useEffect(() => {
    document.title = '登录 · 宋一天'
  }, [])

  const { user } = useAuth()
  const [params] = useSearchParams()
  const returnTo = params.get('return_to') ?? '/'
  const denied = params.get('error') === 'owner-only'
  const { data, loading } = useFetch<ProviderInfo[]>('/api/auth/providers')
  const github = data?.find((p) => p.name === 'github')

  if (user) {
    return <Navigate to={returnTo} replace />
  }

  return (
    <div className="wrap login-page">
      <header className="page-head">
        <h1>站长入口</h1>
        <p>这里是后台的门:用 GitHub 登录后可以在网页上直接写文章、发布、删评论。</p>
      </header>
      {denied && (
        <p className="form-error">这个 GitHub 账号不是站长,本站暂不开放访客登录。</p>
      )}
      <div className="login-buttons">
        {loading && <p className="state-note">加载中…</p>}
        {github &&
          (github.configured ? (
            <a
              className="login-btn"
              href={`/auth/github/login?return_to=${encodeURIComponent(returnTo)}`}
            >
              <span className="login-mark">GH</span>
              用 GitHub 登录
            </a>
          ) : (
            <span className="login-btn disabled" title="尚未配置 GITHUB_CLIENT_ID / SECRET">
              <span className="login-mark">GH</span>
              用 GitHub 登录(未配置)
            </span>
          ))}
      </div>
      <p className="login-note">只有站长本人的 GitHub 账号能进来;其他账号会被礼貌地请回。</p>
    </div>
  )
}

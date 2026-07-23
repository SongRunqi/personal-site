import { Link, NavLink, Route, Routes, useLocation } from 'react-router-dom'
import Home from './pages/Home'
import Projects from './pages/Projects'
import Blog from './pages/Blog'
import Post from './pages/Post'
import About from './pages/About'
import Login from './pages/Login'
import Write from './pages/Write'
import Manage from './pages/Manage'
import { AuthProvider, useAuth } from './auth'

function UserArea() {
  const { user, loading, logout } = useAuth()
  const location = useLocation()
  if (loading) return null
  if (!user) {
    return (
      <Link className="nav-login" to={`/login?return_to=${encodeURIComponent(location.pathname)}`}>
        登录
      </Link>
    )
  }
  return (
    <span className="nav-user">
      {user.isAdmin && (
        <NavLink to="/write" className="nav-write">
          写文章
        </NavLink>
      )}
      {user.avatarUrl ? (
        <img className="nav-avatar" src={user.avatarUrl} alt="" referrerPolicy="no-referrer" />
      ) : (
        <span className="nav-avatar fallback">{user.name.slice(0, 1)}</span>
      )}
      <span className="nav-name">{user.name}</span>
      <button type="button" className="nav-logout" onClick={() => void logout()}>
        退出
      </button>
    </span>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <div className="wrap">
        <nav className="nav">
          <NavLink className="brand" to="/">
            <span className="brand-name">宋一天</span>
            <span className="brand-sub">SONGRUNQI</span>
          </NavLink>
          <div className="nav-links">
            <NavLink to="/" end>
              首页
            </NavLink>
            <NavLink to="/projects">项目</NavLink>
            <NavLink to="/blog">博客</NavLink>
            <NavLink to="/about">关于</NavLink>
            <UserArea />
          </div>
        </nav>
      </div>

      <main>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/projects" element={<Projects />} />
          <Route path="/blog" element={<Blog />} />
          <Route path="/blog/:slug" element={<Post />} />
          <Route path="/about" element={<About />} />
          <Route path="/login" element={<Login />} />
          <Route path="/write" element={<Manage />} />
          <Route path="/write/new" element={<Write />} />
          <Route path="/write/:slug" element={<Write />} />
          <Route
            path="*"
            element={
              <div className="wrap state-note">
                这一页不存在。回<a href="/">首页</a>看看?
              </div>
            }
          />
        </Routes>
      </main>

      <div className="wrap">
        <footer className="footer">
          <span>宋一天 · 手写的一方角落</span>
        </footer>
      </div>
    </AuthProvider>
  )
}

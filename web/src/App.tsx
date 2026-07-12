import { NavLink, Route, Routes } from 'react-router-dom'
import Home from './pages/Home'
import Projects from './pages/Projects'
import Blog from './pages/Blog'
import Post from './pages/Post'
import About from './pages/About'

export default function App() {
  return (
    <>
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
          <span>
            <a href="https://github.com/SongRunqi" target="_blank" rel="noopener noreferrer">
              GitHub
            </a>{' '}
            · <a href="/feed.xml">RSS</a> · <a href="mailto:yitiansong4@gmail.com">邮箱</a>
          </span>
        </footer>
      </div>
    </>
  )
}

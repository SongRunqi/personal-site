import { useEffect } from 'react'
import { useFetch, type Project } from '../api'

const statusLabel: Record<string, string> = {
  active: '进行中',
  paused: '搁置',
  archived: '已归档',
}

export function ProjectCard({ project }: { project: Project }) {
  return (
    <article className="project-card">
      <h3>
        {project.name}
        <span className="project-status">{statusLabel[project.status] ?? project.status}</span>
      </h3>
      <p className="project-tagline">{project.tagline}</p>
      <p className="project-desc">{project.description}</p>
      <div className="project-stack">
        {project.stack.map((s) => (
          <span key={s}>{s}</span>
        ))}
      </div>
      <div className="project-links">
        {project.repo && (
          <a href={project.repo} target="_blank" rel="noopener noreferrer">
            源码
          </a>
        )}
        {project.url && project.url !== project.repo && (
          <a href={project.url} target="_blank" rel="noopener noreferrer">
            主页
          </a>
        )}
      </div>
    </article>
  )
}

export default function Projects() {
  useEffect(() => {
    document.title = '项目 · 宋一天'
  }, [])

  const { data, error, loading } = useFetch<Project[]>('/api/projects')

  return (
    <div className="wrap">
      <header className="page-head">
        <h1>项目</h1>
        <p>一个人,从想法到上线。每个项目都是自己真的在用的东西。</p>
      </header>
      <section className="section" style={{ borderTop: 'none', paddingTop: 40 }}>
        {loading && <p className="state-note">加载中…</p>}
        {error && <p className="state-note">项目列表加载失败,稍后再试。</p>}
        {data && (
          <div className="project-list">
            {data.map((p) => (
              <ProjectCard key={p.name} project={p} />
            ))}
          </div>
        )}
      </section>
    </div>
  )
}

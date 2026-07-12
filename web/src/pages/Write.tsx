import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ChangeEvent,
  type ClipboardEvent,
  type DragEvent,
} from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { requestJSON, type ArticleDraft } from '../api'
import { useAuth } from '../auth'

const empty: ArticleDraft = { slug: '', title: '', markdown: '', summary: '', tags: [], draft: true }

function suggestSlug(title: string): string {
  const ascii = title
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  if (ascii.length >= 3) return ascii.slice(0, 60)
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, '0')
  return `post-${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}`
}

export default function Write() {
  const { user, loading } = useAuth()
  const { slug: editSlug } = useParams<{ slug: string }>()
  const navigate = useNavigate()

  const [article, setArticle] = useState<ArticleDraft>(empty)
  const [slugTouched, setSlugTouched] = useState(!!editSlug)
  const [tagsText, setTagsText] = useState('')
  const [previewHTML, setPreviewHTML] = useState('')
  const [showPreview, setShowPreview] = useState(true)
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [existing, setExisting] = useState<ArticleDraft[]>([])
  const taRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    document.title = (editSlug ? '编辑文章' : '写文章') + ' · 宋一天'
  }, [editSlug])

  // 编辑模式:载入原文
  useEffect(() => {
    if (!editSlug || !user?.isAdmin) return
    requestJSON<ArticleDraft>(`/api/admin/articles/${editSlug}`)
      .then((a) => {
        setArticle(a)
        setTagsText((a.tags ?? []).join(', '))
      })
      .catch((e) => setError((e as Error).message))
  }, [editSlug, user])

  // 新建模式:列出已有文章方便进入编辑
  useEffect(() => {
    if (editSlug || !user?.isAdmin) return
    requestJSON<ArticleDraft[]>('/api/admin/articles')
      .then(setExisting)
      .catch(() => {})
  }, [editSlug, user])

  // 防抖预览
  useEffect(() => {
    if (!user?.isAdmin || !showPreview) return
    const t = setTimeout(() => {
      requestJSON<{ html: string }>('/api/admin/preview', {
        method: 'POST',
        body: JSON.stringify({ markdown: article.markdown }),
      })
        .then((d) => setPreviewHTML(d.html))
        .catch(() => {})
    }, 400)
    return () => clearTimeout(t)
  }, [article.markdown, showPreview, user])

  const insert = useCallback(
    (before: string, after = '', placeholder = '') => {
      const ta = taRef.current
      if (!ta) return
      const { selectionStart: start, selectionEnd: end, value } = ta
      const selected = value.slice(start, end) || placeholder
      const next = value.slice(0, start) + before + selected + after + value.slice(end)
      setArticle((a) => ({ ...a, markdown: next }))
      requestAnimationFrame(() => {
        ta.focus()
        ta.setSelectionRange(start + before.length, start + before.length + selected.length)
      })
    },
    [setArticle],
  )

  async function uploadImage(file: File) {
    const fd = new FormData()
    fd.append('file', file)
    setStatus('图片上传中…')
    try {
      const res = await fetch('/api/admin/upload', { method: 'POST', body: fd })
      const data = await res.json()
      if (!res.ok) throw new Error(data?.error ?? `HTTP ${res.status}`)
      insert(`![${file.name.replace(/\.[^.]*$/, '')}](${data.url})\n`, '', '')
      setStatus('图片已插入')
    } catch (e) {
      setError((e as Error).message)
      setStatus('')
    }
  }

  function onPaste(e: ClipboardEvent<HTMLTextAreaElement>) {
    const file = Array.from(e.clipboardData.files).find((f) => f.type.startsWith('image/'))
    if (file) {
      e.preventDefault()
      void uploadImage(file)
    }
  }

  function onDrop(e: DragEvent<HTMLTextAreaElement>) {
    const file = Array.from(e.dataTransfer.files).find((f) => f.type.startsWith('image/'))
    if (file) {
      e.preventDefault()
      void uploadImage(file)
    }
  }

  function onPickImage(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) void uploadImage(file)
    e.target.value = ''
  }

  async function save(draft: boolean) {
    setBusy(true)
    setError('')
    const payload: ArticleDraft = {
      ...article,
      draft,
      slug: article.slug.trim() || suggestSlug(article.title),
      tags: tagsText
        .split(/[,,、]/)
        .map((t) => t.trim())
        .filter(Boolean),
    }
    try {
      const saved = editSlug
        ? await requestJSON<ArticleDraft>(`/api/admin/articles/${editSlug}`, {
            method: 'PUT',
            body: JSON.stringify(payload),
          })
        : await requestJSON<ArticleDraft>('/api/admin/articles', {
            method: 'POST',
            body: JSON.stringify(payload),
          })
      setArticle(saved)
      setTagsText((saved.tags ?? []).join(', '))
      setStatus(draft ? '草稿已保存' : '已发布')
      if (!editSlug || saved.slug !== editSlug) {
        navigate(`/write/${saved.slug}`, { replace: true })
      }
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function removeArticle() {
    if (!editSlug || !window.confirm('确定删除这篇文章?点赞和留言会一并失效。')) return
    try {
      await requestJSON(`/api/admin/articles/${editSlug}`, { method: 'DELETE' })
      navigate('/write')
      setArticle(empty)
      setTagsText('')
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

  const toolbar: Array<{ label: string; title: string; fn: () => void }> = [
    { label: 'H2', title: '二级标题', fn: () => insert('\n## ', '\n', '标题') },
    { label: 'H3', title: '三级标题', fn: () => insert('\n### ', '\n', '标题') },
    { label: 'B', title: '加粗', fn: () => insert('**', '**', '加粗') },
    { label: 'I', title: '斜体', fn: () => insert('*', '*', '斜体') },
    { label: '“”', title: '引用', fn: () => insert('\n> ', '\n', '引用') },
    { label: '‹›', title: '行内代码', fn: () => insert('`', '`', 'code') },
    { label: '```', title: '代码块', fn: () => insert('\n```go\n', '\n```\n', '代码') },
    { label: '🔗', title: '链接', fn: () => insert('[', '](https://)', '链接文字') },
    { label: '•', title: '列表', fn: () => insert('\n- ', '', '条目') },
    { label: '—', title: '分隔线', fn: () => insert('\n---\n') },
    { label: '▦', title: '表格', fn: () => insert('\n| 列一 | 列二 |\n|---|---|\n| 内容 | 内容 |\n') },
  ]

  return (
    <div className="wrap write-page">
      <header className="write-head">
        <h1>{editSlug ? '编辑文章' : '写文章'}</h1>
        <div className="write-actions">
          {status && <span className="write-status">{status}</span>}
          <label className="preview-toggle">
            <input
              type="checkbox"
              checked={showPreview}
              onChange={(e) => setShowPreview(e.target.checked)}
            />
            预览
          </label>
          <button type="button" className="btn-ghost" disabled={busy} onClick={() => save(true)}>
            存草稿
          </button>
          <button
            type="button"
            className="btn-primary btn-small"
            disabled={busy || !article.title.trim()}
            onClick={() => save(false)}
          >
            发布
          </button>
          {editSlug && (
            <button type="button" className="btn-danger" onClick={removeArticle}>
              删除
            </button>
          )}
        </div>
      </header>

      {error && <p className="form-error">{error}</p>}

      <div className="write-meta">
        <input
          className="write-title"
          value={article.title}
          onChange={(e) => {
            const title = e.target.value
            setArticle((a) => ({
              ...a,
              title,
              slug: slugTouched ? a.slug : suggestSlug(title),
            }))
          }}
          placeholder="文章标题"
        />
        <div className="write-fields">
          <label>
            slug
            <input
              value={article.slug}
              onChange={(e) => {
                setSlugTouched(true)
                setArticle((a) => ({ ...a, slug: e.target.value }))
              }}
              placeholder="my-first-post"
            />
          </label>
          <label>
            标签
            <input
              value={tagsText}
              onChange={(e) => setTagsText(e.target.value)}
              placeholder="随笔, 开发(逗号分隔)"
            />
          </label>
          <label className="write-summary">
            摘要
            <input
              value={article.summary}
              onChange={(e) => setArticle((a) => ({ ...a, summary: e.target.value }))}
              placeholder="留空则自动取正文前 160 字"
            />
          </label>
        </div>
      </div>

      <div className="write-toolbar" role="toolbar" aria-label="排版工具">
        {toolbar.map((b) => (
          <button key={b.title} type="button" title={b.title} onClick={b.fn}>
            {b.label}
          </button>
        ))}
        <label className="toolbar-upload" title="插入图片(也可以直接粘贴 / 拖进正文)">
          🖼 图片
          <input type="file" accept="image/png,image/jpeg,image/gif,image/webp" onChange={onPickImage} />
        </label>
      </div>

      <div className={`write-panes${showPreview ? '' : ' single'}`}>
        <textarea
          ref={taRef}
          className="write-editor"
          value={article.markdown}
          onChange={(e) => setArticle((a) => ({ ...a, markdown: e.target.value }))}
          onPaste={onPaste}
          onDrop={onDrop}
          placeholder={'正文用 Markdown 写。\n\n图片可以直接粘贴或拖进来,会自动上传并插入。'}
          spellCheck={false}
        />
        {showPreview && (
          <div className="write-preview">
            {article.markdown.trim() ? (
              <div className="prose" dangerouslySetInnerHTML={{ __html: previewHTML }} />
            ) : (
              <p className="state-note">预览会显示在这里,和文章页同一套排版。</p>
            )}
          </div>
        )}
      </div>

      {!editSlug && existing.length > 0 && (
        <section className="write-existing">
          <div className="section-head">
            <h2>已有文章</h2>
            <div className="rule"></div>
          </div>
          <ul className="manage-list">
            {existing.map((a) => (
              <li key={a.slug}>
                <span className={`manage-status ${a.source === 'file' ? 'file' : a.draft ? 'draft' : 'pub'}`}>
                  {a.source === 'file' ? '仓库' : a.draft ? '草稿' : '已发布'}
                </span>
                <span className="manage-title">{a.title}</span>
                {a.source === 'db' ? (
                  <Link to={`/write/${a.slug}`}>编辑</Link>
                ) : (
                  <span className="manage-note">在 content/posts/ 里改</span>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  )
}

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ChangeEvent,
  type ClipboardEvent,
  type DragEvent,
  type KeyboardEvent,
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

// 编辑器页:/write/new 新建,/write/:slug 编辑。列表与删除在 /write(文章管理)。
export default function Write() {
  const { user, loading } = useAuth()
  const { slug: editSlug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const autosaveKey = `write-autosave:${editSlug ?? 'new'}`

  const [article, setArticle] = useState<ArticleDraft>(empty)
  const [slugTouched, setSlugTouched] = useState(!!editSlug)
  const [tagsText, setTagsText] = useState('')
  const [previewHTML, setPreviewHTML] = useState('')
  const [showPreview, setShowPreview] = useState(true)
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [recovered, setRecovered] = useState<ArticleDraft | null>(null)
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
        setDirty(false)
      })
      .catch((e) => setError((e as Error).message))
  }, [editSlug, user])

  // 崩溃/误关恢复:发现本地有未保存的稿子时给出恢复入口
  useEffect(() => {
    const raw = localStorage.getItem(autosaveKey)
    if (raw) {
      try {
        setRecovered(JSON.parse(raw) as ArticleDraft)
      } catch {
        localStorage.removeItem(autosaveKey)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autosaveKey])

  // 有改动时每 2 秒往 localStorage 存一份
  useEffect(() => {
    if (!dirty) return
    const t = setTimeout(() => {
      localStorage.setItem(autosaveKey, JSON.stringify({ ...article, tags: tagsText.split(/[,,、]/).map((s) => s.trim()).filter(Boolean) }))
    }, 2000)
    return () => clearTimeout(t)
  }, [article, tagsText, dirty, autosaveKey])

  // 离开页面前提醒未保存
  useEffect(() => {
    if (!dirty) return
    const h = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', h)
    return () => window.removeEventListener('beforeunload', h)
  }, [dirty])

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

  const touch = useCallback(() => setDirty(true), [])

  const insert = useCallback(
    (before: string, after = '', placeholder = '') => {
      const ta = taRef.current
      if (!ta) return
      const { selectionStart: start, selectionEnd: end, value } = ta
      const selected = value.slice(start, end) || placeholder
      const next = value.slice(0, start) + before + selected + after + value.slice(end)
      setArticle((a) => ({ ...a, markdown: next }))
      setDirty(true)
      requestAnimationFrame(() => {
        ta.focus()
        ta.setSelectionRange(start + before.length, start + before.length + selected.length)
      })
    },
    [setArticle],
  )

  async function uploadImages(files: File[]) {
    for (const file of files) {
      const fd = new FormData()
      fd.append('file', file)
      setStatus(`上传 ${file.name}…`)
      try {
        const res = await fetch('/api/admin/upload', { method: 'POST', body: fd })
        const data = await res.json()
        if (!res.ok) throw new Error(data?.error ?? `HTTP ${res.status}`)
        insert(`![${file.name.replace(/\.[^.]*$/, '')}](${data.url})\n`, '', '')
      } catch (e) {
        setError((e as Error).message)
        setStatus('')
        return
      }
    }
    setStatus(files.length > 1 ? `已插入 ${files.length} 张图片` : '图片已插入')
  }

  function imageFiles(list: FileList): File[] {
    return Array.from(list).filter((f) => f.type.startsWith('image/'))
  }

  function onPaste(e: ClipboardEvent<HTMLTextAreaElement>) {
    const files = imageFiles(e.clipboardData.files)
    if (files.length > 0) {
      e.preventDefault()
      void uploadImages(files)
    }
  }

  function onDrop(e: DragEvent<HTMLTextAreaElement>) {
    const files = imageFiles(e.dataTransfer.files)
    if (files.length > 0) {
      e.preventDefault()
      void uploadImages(files)
    }
  }

  function onPickImage(e: ChangeEvent<HTMLInputElement>) {
    if (e.target.files) void uploadImages(imageFiles(e.target.files))
    e.target.value = ''
  }

  // 快捷键:⌘/Ctrl+S 保存、⌘/Ctrl+Enter 发布、⌘/Ctrl+B/I 粗斜体、Tab 缩进
  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    const mod = e.metaKey || e.ctrlKey
    if (mod && e.key === 's') {
      e.preventDefault()
      void save(article.draft)
    } else if (mod && e.key === 'Enter') {
      e.preventDefault()
      void save(false)
    } else if (mod && e.key === 'b') {
      e.preventDefault()
      insert('**', '**', '加粗')
    } else if (mod && e.key === 'i') {
      e.preventDefault()
      insert('*', '*', '斜体')
    } else if (e.key === 'Tab') {
      e.preventDefault()
      insert('  ')
    }
  }

  async function save(draft: boolean) {
    if (busy || !article.title.trim()) return
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
      setDirty(false)
      localStorage.removeItem(autosaveKey)
      const t = new Date()
      const p = (n: number) => String(n).padStart(2, '0')
      setStatus(`${draft ? '草稿已保存' : '已发布'} ${p(t.getHours())}:${p(t.getMinutes())}`)
      if (!editSlug || saved.slug !== editSlug) {
        navigate(`/write/${saved.slug}`, { replace: true })
      }
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
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
    { label: 'B', title: '加粗(⌘B)', fn: () => insert('**', '**', '加粗') },
    { label: 'I', title: '斜体(⌘I)', fn: () => insert('*', '*', '斜体') },
    { label: '“”', title: '引用', fn: () => insert('\n> ', '\n', '引用') },
    { label: '‹›', title: '行内代码', fn: () => insert('`', '`', 'code') },
    { label: '```', title: '代码块', fn: () => insert('\n```go\n', '\n```\n', '代码') },
    { label: '🔗', title: '链接', fn: () => insert('[', '](https://)', '链接文字') },
    { label: '•', title: '列表', fn: () => insert('\n- ', '', '条目') },
    { label: '—', title: '分隔线', fn: () => insert('\n---\n') },
    { label: '▦', title: '表格', fn: () => insert('\n| 列一 | 列二 |\n|---|---|\n| 内容 | 内容 |\n') },
  ]

  const chars = [...article.markdown].length

  return (
    <div className="wrap write-page">
      <header className="write-head">
        <div className="write-head-left">
          <Link className="post-back" to="/write">
            ← 文章管理
          </Link>
          <h1>{editSlug ? '编辑文章' : '写文章'}</h1>
        </div>
        <div className="write-actions">
          {status && <span className="write-status">{status}</span>}
          {dirty && <span className="write-status dirty">未保存</span>}
          <label className="preview-toggle">
            <input
              type="checkbox"
              checked={showPreview}
              onChange={(e) => setShowPreview(e.target.checked)}
            />
            预览
          </label>
          <button
            type="button"
            className="btn-ghost"
            disabled={busy || !article.title.trim()}
            onClick={() => save(true)}
            title="⌘/Ctrl+S"
          >
            存草稿
          </button>
          <button
            type="button"
            className="btn-primary btn-small"
            disabled={busy || !article.title.trim()}
            onClick={() => save(false)}
            title="⌘/Ctrl+Enter"
          >
            {article.draft ? '发布' : '更新'}
          </button>
        </div>
      </header>

      {error && <p className="form-error">{error}</p>}
      {recovered && (
        <p className="recover-note">
          发现一份未保存的本地稿子(《{recovered.title || '无标题'}》)。
          <button
            type="button"
            onClick={() => {
              setArticle(recovered)
              setTagsText((recovered.tags ?? []).join(', '))
              setSlugTouched(!!recovered.slug)
              setDirty(true)
              setRecovered(null)
            }}
          >
            恢复它
          </button>
          <button
            type="button"
            onClick={() => {
              localStorage.removeItem(autosaveKey)
              setRecovered(null)
            }}
          >
            丢弃
          </button>
        </p>
      )}

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
            touch()
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
                touch()
              }}
              placeholder="my-first-post"
            />
          </label>
          <label>
            标签
            <input
              value={tagsText}
              onChange={(e) => {
                setTagsText(e.target.value)
                touch()
              }}
              placeholder="随笔, 开发(逗号分隔)"
            />
          </label>
          <label className="write-summary">
            摘要
            <input
              value={article.summary}
              onChange={(e) => {
                setArticle((a) => ({ ...a, summary: e.target.value }))
                touch()
              }}
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
        <label className="toolbar-upload" title="插入图片(也可以直接粘贴 / 拖进正文,支持多张)">
          🖼 图片
          <input
            type="file"
            multiple
            accept="image/png,image/jpeg,image/gif,image/webp"
            onChange={onPickImage}
          />
        </label>
        <span className="char-count">{chars} 字</span>
      </div>

      <div className={`write-panes${showPreview ? '' : ' single'}`}>
        <textarea
          ref={taRef}
          className="write-editor"
          value={article.markdown}
          onChange={(e) => {
            setArticle((a) => ({ ...a, markdown: e.target.value }))
            touch()
          }}
          onPaste={onPaste}
          onDrop={onDrop}
          onKeyDown={onKeyDown}
          placeholder={'正文用 Markdown 写。\n\n图片可以直接粘贴或拖进来(支持多张),会自动上传并插入。\n⌘/Ctrl+S 存草稿,⌘/Ctrl+Enter 发布。'}
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
    </div>
  )
}

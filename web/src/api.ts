import { useEffect, useState } from 'react'

export interface PostMeta {
  slug: string
  title: string
  date: string
  tags: string[] | null
  summary: string
  source?: 'file' | 'db'
}

export interface Me {
  id: number
  name: string
  avatarUrl: string
  isAdmin: boolean
  provider: string
}

export interface CommentItem {
  id: number
  body: string
  createdAt: string
  author: { name: string; avatarUrl: string }
  mine: boolean
}

export interface ArticleDraft {
  slug: string
  title: string
  markdown: string
  summary: string
  tags: string[]
  draft: boolean
  date?: string
  publishedAt?: string
  source?: 'file' | 'db'
}

// requestJSON:带 cookie 的 JSON 请求,失败时抛出后端给的中文错误信息。
export async function requestJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
    ...init,
  })
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    throw new Error(data?.error ?? `HTTP ${res.status}`)
  }
  return data as T
}

export interface Post extends PostMeta {
  html: string
}

export interface Project {
  name: string
  tagline: string
  description: string
  url: string
  repo: string
  stack: string[]
  status: string
}

interface FetchState<T> {
  data?: T
  error?: string
  loading: boolean
}

export function useFetch<T>(url: string): FetchState<T> {
  const [state, setState] = useState<FetchState<T>>({ loading: true })

  useEffect(() => {
    let alive = true
    setState({ loading: true })
    fetch(url)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        return res.json() as Promise<T>
      })
      .then((data) => alive && setState({ data, loading: false }))
      .catch((err: Error) => alive && setState({ error: err.message, loading: false }))
    return () => {
      alive = false
    }
  }, [url])

  return state
}

export function formatDate(iso: string): string {
  const d = new Date(iso)
  return `${d.getUTCFullYear()} 年 ${d.getUTCMonth() + 1} 月 ${d.getUTCDate()} 日`
}

import { useEffect, useState } from 'react'

export interface PostMeta {
  slug: string
  title: string
  date: string
  tags: string[] | null
  summary: string
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

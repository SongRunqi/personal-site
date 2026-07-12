import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { requestJSON, type Me } from './api'

interface AuthState {
  user: Me | null
  loading: boolean
  logout: () => Promise<void>
}

const AuthCtx = createContext<AuthState>({ user: null, loading: true, logout: async () => {} })

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<Me | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    requestJSON<{ user: Me | null }>('/api/me')
      .then((d) => setUser(d.user))
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  const logout = useCallback(async () => {
    await requestJSON('/auth/logout', { method: 'POST' })
    setUser(null)
  }, [])

  return <AuthCtx.Provider value={{ user, loading, logout }}>{children}</AuthCtx.Provider>
}

export function useAuth() {
  return useContext(AuthCtx)
}

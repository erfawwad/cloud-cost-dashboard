import { createContext, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { api, setToken, clearToken, getToken } from '../api/client'
import type { User } from '../types'

interface AuthContextValue {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!getToken()) {
      setLoading(false)
      return
    }
    api
      .get<User>('/api/me')
      .then((res) => setUser(res.data))
      .catch(() => clearToken())
      .finally(() => setLoading(false))
  }, [])

  async function login(email: string, password: string) {
    const res = await api.post('/api/auth/login', { email, password })
    setToken(res.data.token)
    setUser(res.data.user)
  }

  function logout() {
    clearToken()
    setUser(null)
  }

  return <AuthContext.Provider value={{ user, loading, login, logout }}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import api from './api'

interface User {
  id: string
  username: string
  is_admin: boolean
  jad: number
  jad_locked: number
}

interface AuthCtx {
  user: User | null
  token: string | null
  login: (username: string, password: string) => Promise<void>
  register: (username: string, password: string) => Promise<void>
  logout: () => void
  refreshUser: () => Promise<void>
}

const Ctx = createContext<AuthCtx>(null!)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'))

  useEffect(() => {
    if (token) refreshUser()
  }, [token])

  async function refreshUser() {
    try {
      const res = await api.get('/me')
      setUser(res.data)
    } catch {
      logout()
    }
  }

  async function login(username: string, password: string) {
    const res = await api.post('/auth/login', { username, password })
    localStorage.setItem('token', res.data.token)
    setToken(res.data.token)
    setUser(res.data.user)
  }

  async function register(username: string, password: string) {
    const res = await api.post('/auth/register', { username, password })
    localStorage.setItem('token', res.data.token)
    setToken(res.data.token)
    setUser(res.data.user)
  }

  function logout() {
    localStorage.removeItem('token')
    setToken(null)
    setUser(null)
  }

  return (
    <Ctx.Provider value={{ user, token, login, register, logout, refreshUser }}>
      {children}
    </Ctx.Provider>
  )
}

export const useAuth = () => useContext(Ctx)

import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await login(email, password)
      navigate('/')
    } catch {
      setError('Invalid email or password')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <form onSubmit={handleSubmit} className="card" style={{ padding: 32, width: 340 }}>
        <h1 style={{ fontSize: 20, margin: '0 0 24px' }}>Cloud Cost Dashboard</h1>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <label style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Email
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              style={{ display: 'block', width: '100%', marginTop: 4 }}
            />
          </label>
          <label style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Password
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              style={{ display: 'block', width: '100%', marginTop: 4 }}
            />
          </label>
          {error && <div style={{ color: 'var(--status-critical)', fontSize: 13 }}>{error}</div>}
          <button
            type="submit"
            disabled={submitting}
            style={{
              marginTop: 8,
              padding: '8px 12px',
              background: 'var(--series-1)',
              color: 'white',
              border: 'none',
              borderRadius: 6,
              cursor: 'pointer',
              fontWeight: 600,
            }}
          >
            {submitting ? 'Signing in…' : 'Sign in'}
          </button>
        </div>
      </form>
    </div>
  )
}

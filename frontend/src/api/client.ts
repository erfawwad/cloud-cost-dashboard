import axios from 'axios'

const TOKEN_KEY = 'ccd_token'

// `??` (not `||`) matters here: an explicitly empty VITE_API_URL means "same
// origin" (production behind a reverse proxy that routes /api to the
// backend) — `||` would wrongly treat "" as unset and fall back to localhost.
export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL ?? 'http://localhost:8080',
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem(TOKEN_KEY)
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  },
)

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

import axios from 'axios'

const BASE = import.meta.env.VITE_API_URL || ''

const api = axios.create({ baseURL: BASE, timeout: 10000 })

// Attach access token to every request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// Auto-refresh on 401 (only for authenticated requests that have a refresh token)
api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config
    const refresh = localStorage.getItem('refresh_token')
    const hadAccessToken = !!localStorage.getItem('access_token')
    if (error.response?.status === 401 && !original._retry && refresh && hadAccessToken) {
      original._retry = true
      try {
        const { data } = await axios.post(`${BASE}/api/v1/auth/refresh`, { refresh_token: refresh })
        localStorage.setItem('access_token', data.access_token)
        localStorage.setItem('refresh_token', data.refresh_token)
        original.headers.Authorization = `Bearer ${data.access_token}`
        return api(original)
      } catch {
        localStorage.clear()
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

// ─── Auth ──────────────────────────────────────────────────────────────────
export const authApi = {
  requestOTP: (email) => api.post('/api/v1/auth/request-otp', { email }),
  verifyOTP: (email, code) => api.post('/api/v1/auth/verify-otp', { email, code }),
  logout: () => api.post('/api/v1/auth/logout'),
}

// ─── Blog ──────────────────────────────────────────────────────────────────
export const blogApi = {
  getPublic: (page = 1, tag = '') =>
    api.get('/api/v1/posts/public', { params: { page, tag } }),
  getPost: (id) => api.get(`/api/v1/posts/${id}`),
  getMyPosts: (page = 1) => api.get('/api/v1/posts/my/all', { params: { page } }),
  createPost: (data) => api.post('/api/v1/posts', data),
  updatePost: (id, data) => api.put(`/api/v1/posts/${id}`, data),
  deletePost: (id) => api.delete(`/api/v1/posts/${id}`),
  toggleLike: (id) => api.post(`/api/v1/posts/${id}/like`),
}

// ─── Feed ──────────────────────────────────────────────────────────────────
export const feedApi = {
  getFeed: (page = 1) => api.get('/api/v1/feed', { params: { page } }),
  getInterests: () => api.get('/api/v1/feed/interests'),
  setInterests: (tags) => api.put('/api/v1/feed/interests', { tags }),
}

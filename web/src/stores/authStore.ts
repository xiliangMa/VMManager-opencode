import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import axios from 'axios'

interface User {
  id: string
  username: string
  email: string
  role: string
  language: string
  timezone: string
}

interface AuthState {
  isAuthenticated: boolean
  user: User | null
  token: string | null
  refreshToken: string | null
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  updateUser: (user: Partial<User>) => void
}

const API_URL = import.meta.env.VITE_API_URL || '/api/v1'

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      isAuthenticated: false,
      user: null,
      token: null,
      refreshToken: null,

      login: async (username: string, password: string) => {
        try {
          const response = await axios.post(`${API_URL}/auth/login`, {
            username,
            password
          })

          const { token, refresh_token, user } = response.data.data

          set({
            isAuthenticated: true,
            user,
            token,
            refreshToken: refresh_token
          })
        } catch (error) {
          throw error
        }
      },

      logout: () => {
        set({
          isAuthenticated: false,
          user: null,
          token: null,
          refreshToken: null
        })
      },

      updateUser: (userData: Partial<User>) => {
        set((state) => ({
          user: state.user ? { ...state.user, ...userData } : null
        }))
      }
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        token: state.token,
        refreshToken: state.refreshToken,
        user: state.user,
        isAuthenticated: state.isAuthenticated
      })
    }
  )
)

axios.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

axios.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true

      try {
        const refreshToken = useAuthStore.getState().refreshToken
        if (refreshToken) {
          const response = await axios.post(`${API_URL}/auth/refresh`, {
            refresh_token: refreshToken
          })

          const { token } = response.data.data
          useAuthStore.getState().token = token

          originalRequest.headers.Authorization = `Bearer ${token}`
          return axios(originalRequest)
        }
      } catch (refreshError) {
        useAuthStore.getState().logout()
        window.location.href = '/login'
      }
    }

    return Promise.reject(error)
  }
)

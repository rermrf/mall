import { create } from 'zustand'

interface AuthState {
  isLoggedIn: boolean
  checkAuth: () => void
  setLoggedIn: (v: boolean) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  isLoggedIn: !!localStorage.getItem('access_token'),
  checkAuth: () => {
    set({ isLoggedIn: !!localStorage.getItem('access_token') })
  },
  setLoggedIn: (v: boolean) => set({ isLoggedIn: v }),
  clearAuth: () => {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    set({ isLoggedIn: false })
  },
}))

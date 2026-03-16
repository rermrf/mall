import { create } from 'zustand'

interface AuthState {
  isLoggedIn: boolean
  checkAuth: () => void
  setLoggedIn: (v: boolean) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>((set) => {
  // Listen for auth changes from other tabs
  if (typeof window !== 'undefined') {
    window.addEventListener('storage', (e) => {
      if (e.key === 'access_token') {
        if (!e.newValue) {
          set({ isLoggedIn: false })
          window.location.href = '/login'
        } else {
          set({ isLoggedIn: true })
        }
      }
    })
  }

  return {
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
  }
})

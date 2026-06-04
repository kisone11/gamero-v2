import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { UserInfo } from '@/types/api'
import { tokenStorage } from '@/lib/http'

interface AuthState {
  user: UserInfo | null
  isAuthenticated: boolean
  login: (user: UserInfo, tokens: { access: string; refresh: string }) => void
  logout: () => void
  updateUser: (partial: Partial<UserInfo>) => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      login: (user, tokens) => {
        tokenStorage.setAccess(tokens.access)
        tokenStorage.setRefresh(tokens.refresh)
        set({ user, isAuthenticated: true })
      },
      logout: () => {
        tokenStorage.clear()
        set({ user: null, isAuthenticated: false })
      },
      updateUser: (partial) =>
        set((s) => ({ user: s.user ? { ...s.user, ...partial } : null })),
    }),
    {
      name: 'gamero-auth',
      partialize: (s) => ({ user: s.user, isAuthenticated: s.isAuthenticated }),
    },
  ),
)

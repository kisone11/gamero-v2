import { create } from 'zustand'

interface NotifState {
  unreadCount: number
  setUnread: (n: number) => void
  increment: () => void
  decrement: () => void
}

export const useNotifStore = create<NotifState>((set) => ({
  unreadCount: 0,
  setUnread: (n) => set({ unreadCount: n }),
  increment: () => set((s) => ({ unreadCount: s.unreadCount + 1 })),
  decrement: () => set((s) => ({ unreadCount: Math.max(0, s.unreadCount - 1) })),
}))

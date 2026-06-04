import { create } from 'zustand'

export interface ToastItem {
  id: string
  type: 'success' | 'error' | 'info' | 'warning'
  message: string
}

interface ToastState {
  toasts: ToastItem[]
  success: (msg: string) => void
  error: (msg: string) => void
  info: (msg: string) => void
  warning: (msg: string) => void
  remove: (id: string) => void
}

export const useToastStore = create<ToastState>((set) => {
  const push = (type: ToastItem['type']) => (message: string) => {
    const id = Date.now().toString(36) + Math.random().toString(36).slice(2)
    set((s) => ({ toasts: [...s.toasts, { id, type, message }] }))
    setTimeout(() => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })), 4000)
  }
  return {
    toasts: [],
    success: push('success'),
    error: push('error'),
    info: push('info'),
    warning: push('warning'),
    remove: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
  }
})

// 便捷调用
export const toast = {
  success: (m: string) => useToastStore.getState().success(m),
  error: (m: string) => useToastStore.getState().error(m),
  info: (m: string) => useToastStore.getState().info(m),
  warning: (m: string) => useToastStore.getState().warning(m),
}

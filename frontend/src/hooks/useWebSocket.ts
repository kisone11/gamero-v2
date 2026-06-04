import { useEffect, useRef, useCallback } from 'react'
import { useAuthStore } from '@/stores/authStore'
import { useNotifStore } from '@/stores/notifStore'
import { tokenStorage } from '@/lib/http'
import type { WsMessage } from '@/types/api'

// 退避重连间隔（ms）
const WS_RECONNECT_INTERVALS = [1000, 2000, 5000, 10000, 30000]
const WS_PING_INTERVAL = 25_000

export function useWebSocket(onMessage?: (msg: WsMessage) => void) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const wsRef = useRef<WebSocket | null>(null)
  const retryRef = useRef(0)
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const pingRef = useRef<ReturnType<typeof setInterval>>(undefined)
  const onMsgRef = useRef(onMessage)

  // 保持 onMessage 引用最新，不触发 effect 重跑
  onMsgRef.current = onMessage

  const connect = useCallback(() => {
    if (!isAuthenticated) return

    const token = tokenStorage.getAccess()
    if (!token) return

    const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = isDev ? `${window.location.hostname}:8080` : window.location.host
    const wsUrl = `${protocol}//${host}/api/v1/ws?token=${encodeURIComponent(token)}`
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      retryRef.current = 0
      clearInterval(pingRef.current)
      pingRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }))
        }
      }, WS_PING_INTERVAL)
    }

    ws.onmessage = (e: MessageEvent<string>) => {
      try {
        const msg = JSON.parse(e.data) as WsMessage
        // 全局未读数同步
        if (msg.type === 'unread_count') {
          useNotifStore.getState().setUnread(msg.payload.count)
        }
        onMsgRef.current?.(msg)
      } catch {
        // 忽略解析错误
      }
    }

    ws.onclose = () => {
      clearInterval(pingRef.current)
      // 从 store 直接读取最新值，避免闭包捕获到退出前的陈旧 isAuthenticated
      const stillAuthed = useAuthStore.getState().isAuthenticated
      if (!stillAuthed) return
      const delay =
        WS_RECONNECT_INTERVALS[
        Math.min(retryRef.current++, WS_RECONNECT_INTERVALS.length - 1)
        ]
      timerRef.current = setTimeout(connect, delay)
    }

    ws.onerror = () => ws.close()
  }, [isAuthenticated])

  useEffect(() => {
    connect()
    return () => {
      clearTimeout(timerRef.current)
      clearInterval(pingRef.current)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [connect])

  return wsRef
}

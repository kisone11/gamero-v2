import { Outlet } from 'react-router-dom'
import { Navbar } from './navbar'
import { useWebSocket } from '@/hooks/useWebSocket'
import { useNotification } from '@/hooks/useNotification'
import type { WsMessage } from '@/types/api'

export function AppLayout() {
  const { notify } = useNotification()

  const handleWsMessage = (msg: WsMessage) => {
    if (msg.type === 'notification') {
      notify(msg.payload.title, {
        body: msg.payload.content,
        icon: '/favicon.ico',
        tag: `notif-${msg.payload.id}`,
      })
    }
  }

  useWebSocket(handleWsMessage)
  return (
    <div className="min-h-screen">
      <Navbar />
      <main className="relative z-10">
        <Outlet />
      </main>
    </div>
  )
}

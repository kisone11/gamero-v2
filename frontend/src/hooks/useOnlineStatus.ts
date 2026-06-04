import { useState, useEffect } from 'react'

export interface OnlineStatus {
  online: boolean
  /** 上次离线时间 */
  lastOfflineAt: Date | null
  /** 上次在线时间 */
  lastOnlineAt: Date | null
}

export function useOnlineStatus(): OnlineStatus {
  const [state, setState] = useState<OnlineStatus>(() => ({
    online: navigator.onLine,
    lastOfflineAt: null,
    lastOnlineAt: navigator.onLine ? new Date() : null,
  }))

  useEffect(() => {
    const goOnline = () =>
      setState((prev) => ({ online: true, lastOfflineAt: prev.lastOfflineAt, lastOnlineAt: new Date() }))
    const goOffline = () =>
      setState((prev) => ({ online: false, lastOfflineAt: new Date(), lastOnlineAt: prev.lastOnlineAt }))

    window.addEventListener('online', goOnline)
    window.addEventListener('offline', goOffline)
    return () => {
      window.removeEventListener('online', goOnline)
      window.removeEventListener('offline', goOffline)
    }
  }, [])

  return state
}

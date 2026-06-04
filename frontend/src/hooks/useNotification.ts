import { useState, useCallback } from 'react'

export function useNotification() {
  const [permission, setPermission] = useState(Notification.permission)

  const requestPermission = useCallback(async () => {
    const result = await Notification.requestPermission()
    setPermission(result)
  }, [])

  const notify = useCallback(
    (title: string, options?: NotificationOptions) => {
      if (permission === 'granted' && document.hidden) {
        new Notification(title, options)
      }
    },
    [permission],
  )

  return { permission, requestPermission, notify }
}

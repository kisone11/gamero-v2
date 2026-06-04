import { useEffect, useCallback, useRef } from 'react'

export interface KeyboardShortcut {
  key: string
  ctrlKey?: boolean
  metaKey?: boolean
  shiftKey?: boolean
  altKey?: boolean
  handler: (event: KeyboardEvent) => void
  enabled?: boolean
}

export function useKeyboard(shortcuts: KeyboardShortcut | KeyboardShortcut[]) {
  const list = Array.isArray(shortcuts) ? shortcuts : [shortcuts]

  const shortcutsRef = useRef(list)
  shortcutsRef.current = list

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    for (const s of shortcutsRef.current) {
      if (s.enabled === false) continue

      const matchKey = s.key.toLowerCase() === event.key.toLowerCase()
      const matchCtrl = s.ctrlKey ? event.ctrlKey : !s.ctrlKey
      const matchMeta = s.metaKey ? event.metaKey : !s.metaKey
      const matchShift = s.shiftKey ? event.shiftKey : !s.shiftKey
      const matchAlt = s.altKey ? event.altKey : !s.altKey

      if (matchKey && matchCtrl && matchMeta && matchShift && matchAlt) {
        event.preventDefault()
        s.handler(event)
        return
      }
    }
  }, [])

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}

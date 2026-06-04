import { useEffect, useRef, type RefObject } from 'react'

export interface UseClickOutsideOptions {
  handler: (event: MouseEvent | TouchEvent) => void
  enabled?: boolean
}

export function useClickOutside<T extends HTMLElement = HTMLElement>(
  options: UseClickOutsideOptions,
): RefObject<T | null> {
  const ref = useRef<T | null>(null)
  const { handler, enabled = true } = options

  useEffect(() => {
    if (!enabled) return

    const listener = (event: MouseEvent | TouchEvent) => {
      const el = ref.current
      if (!el || el.contains(event.target as Node)) return
      handler(event)
    }

    document.addEventListener('mousedown', listener)
    document.addEventListener('touchstart', listener)
    return () => {
      document.removeEventListener('mousedown', listener)
      document.removeEventListener('touchstart', listener)
    }
  }, [handler, enabled])

  return ref
}

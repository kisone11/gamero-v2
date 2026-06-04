import { useState, useEffect } from 'react'

/**
 * 内置断点别名，也可以直接传任意合法的 media query 字符串。
 *
 * - 'sm'   → (min-width: 640px)
 * - 'md'   → (min-width: 768px)
 * - 'lg'   → (min-width: 1024px)
 * - 'xl'   → (min-width: 1280px)
 * - '2xl'  → (min-width: 1536px)
 * - 'dark' → (prefers-color-scheme: dark)
 * - 'light'→ (prefers-color-scheme: light)
 * - 'motion-reduce' → (prefers-reduced-motion: reduce)
 * - 'portrait'  → (orientation: portrait)
 * - 'landscape' → (orientation: landscape)
 */
export type BreakpointAlias =
  | 'sm'
  | 'md'
  | 'lg'
  | 'xl'
  | '2xl'
  | 'dark'
  | 'light'
  | 'motion-reduce'
  | 'portrait'
  | 'landscape'

const ALIAS_MAP: Record<BreakpointAlias, string> = {
  sm: '(min-width: 640px)',
  md: '(min-width: 768px)',
  lg: '(min-width: 1024px)',
  xl: '(min-width: 1280px)',
  '2xl': '(min-width: 1536px)',
  dark: '(prefers-color-scheme: dark)',
  light: '(prefers-color-scheme: light)',
  'motion-reduce': '(prefers-reduced-motion: reduce)',
  portrait: '(orientation: portrait)',
  landscape: '(orientation: landscape)',
}

function resolveQuery(query: string | BreakpointAlias): string {
  return (ALIAS_MAP as Record<string, string>)[query] ?? query
}

/**
 * useMediaQuery
 *
 * 响应式断点 hook，支持别名或任意 CSS media query 字符串。
 * 在服务端（SSR）返回 false，客户端同步读取当前匹配状态并订阅变化。
 *
 * @param query  断点别名或 CSS media query 字符串
 * @returns      当前是否匹配
 *
 * @example
 * ```tsx
 * const isMobile = useMediaQuery('md') // 宽度 < 768px 时为 false
 * const isDark = useMediaQuery('dark')
 * const isCustom = useMediaQuery('(min-width: 900px) and (max-width: 1100px)')
 * ```
 */
export function useMediaQuery(query: string | BreakpointAlias): boolean {
  const resolved = resolveQuery(query)

  const [matches, setMatches] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false
    return window.matchMedia(resolved).matches
  })

  useEffect(() => {
    if (typeof window === 'undefined') return

    const mql = window.matchMedia(resolved)
    setMatches(mql.matches)

    const handler = (e: MediaQueryListEvent) => setMatches(e.matches)
    // 优先使用 addEventListener（现代浏览器）
    if (mql.addEventListener) {
      mql.addEventListener('change', handler)
      return () => mql.removeEventListener('change', handler)
    } else {
      // 兼容旧版 Safari
      mql.addListener(handler)
      return () => mql.removeListener(handler)
    }
  }, [resolved])

  return matches
}

export default useMediaQuery

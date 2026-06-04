// ============================================================
// Hooks barrel export
// ============================================================

// 已有 hooks
export { useDebounce } from './useDebounce'
export { useDocumentTitle } from './useDocumentTitle'
export { useWebSocket } from './useWebSocket'

// 新增 hooks
export { useLocalStorage } from './useLocalStorage'
export type { UseLocalStorageOptions, UseLocalStorageReturn } from './useLocalStorage'

export { useMediaQuery } from './useMediaQuery'
export type { BreakpointAlias } from './useMediaQuery'

export { useCopyToClipboard } from './useCopyToClipboard'
export type { CopyState, UseCopyToClipboardReturn } from './useCopyToClipboard'

export { useClickOutside } from './useClickOutside'
export type { UseClickOutsideOptions } from './useClickOutside'

export { useKeyboard } from './useKeyboard'
export type { KeyboardShortcut } from './useKeyboard'

export { useThrottle } from './useThrottle'

export { usePrevious } from './usePrevious'

export { useToggle } from './useToggle'
export type { UseToggleReturn } from './useToggle'

export { useCountdown } from './useCountdown'
export type { UseCountdownOptions, UseCountdownReturn } from './useCountdown'

export { useIntersectionObserver } from './useIntersectionObserver'
export type { UseIntersectionObserverOptions, UseIntersectionObserverReturn } from './useIntersectionObserver'

export { useLockBodyScroll } from './useLockBodyScroll'

export { useWindowSize } from './useWindowSize'
export type { WindowSize } from './useWindowSize'

export { useOnlineStatus } from './useOnlineStatus'
export type { OnlineStatus } from './useOnlineStatus'

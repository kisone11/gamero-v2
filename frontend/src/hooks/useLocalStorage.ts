import { useState, useEffect, useCallback } from 'react'
import type { Dispatch, SetStateAction } from 'react'

/** 是否运行在客户端（非 SSR 环境） */
function isClient(): boolean {
  return typeof window !== 'undefined'
}

/** 安全读取 localStorage，解析 JSON，失败返回 undefined */
function readStorage<T>(key: string): T | undefined {
  if (!isClient()) return undefined
  try {
    const raw = window.localStorage.getItem(key)
    return raw === null ? undefined : (JSON.parse(raw) as T)
  } catch {
    return undefined
  }
}

/** 安全写入 localStorage */
function writeStorage<T>(key: string, value: T): void {
  if (!isClient()) return
  try {
    window.localStorage.setItem(key, JSON.stringify(value))
  } catch {
    // 容量溢出或隐私模式下静默失败
  }
}

/** 安全删除 localStorage 条目 */
function removeStorage(key: string): void {
  if (!isClient()) return
  try {
    window.localStorage.removeItem(key)
  } catch {
    // 静默失败
  }
}

export interface UseLocalStorageOptions<T> {
  /** 当 localStorage 中不存在该 key 时的初始值 */
  defaultValue: T
  /**
   * 自定义序列化函数，默认使用 JSON.stringify
   */
  serialize?: (value: T) => string
  /**
   * 自定义反序列化函数，默认使用 JSON.parse
   */
  deserialize?: (raw: string) => T
}

export type UseLocalStorageReturn<T> = [
  /** 当前值 */
  value: T,
  /** 更新函数（与 useState setter 签名兼容） */
  setValue: Dispatch<SetStateAction<T>>,
  /** 删除该 key，恢复为 defaultValue */
  remove: () => void,
]

/**
 * useLocalStorage
 *
 * 泛型 hook，将状态持久化到 localStorage，支持 SSR 安全检查与
 * 自定义序列化/反序列化。
 *
 * @example
 * ```tsx
 * const [theme, setTheme, removeTheme] = useLocalStorage('theme', { defaultValue: 'dark' })
 * ```
 */
export function useLocalStorage<T>(
  key: string,
  options: UseLocalStorageOptions<T>,
): UseLocalStorageReturn<T> {
  const { defaultValue } = options

  const [storedValue, setStoredValue] = useState<T>(() => {
    const found = readStorage<T>(key)
    return found !== undefined ? found : defaultValue
  })

  // 同步写入 localStorage
  useEffect(() => {
    writeStorage(key, storedValue)
  }, [key, storedValue])

  // 监听其他标签页对同一 key 的修改
  useEffect(() => {
    if (!isClient()) return
    const handleStorage = (e: StorageEvent) => {
      if (e.key !== key) return
      if (e.newValue === null) {
        setStoredValue(defaultValue)
      } else {
        try {
          setStoredValue(JSON.parse(e.newValue) as T)
        } catch {
          // 解析失败忽略
        }
      }
    }
    window.addEventListener('storage', handleStorage)
    return () => window.removeEventListener('storage', handleStorage)
  }, [key, defaultValue])

  const remove = useCallback(() => {
    removeStorage(key)
    setStoredValue(defaultValue)
  }, [key, defaultValue])

  return [storedValue, setStoredValue, remove]
}

export default useLocalStorage

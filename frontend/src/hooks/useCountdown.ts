import { useState, useEffect, useCallback, useRef } from 'react'

export interface UseCountdownOptions {
  /** 倒计时总秒数 */
  total: number
  /** 到达 0 时的回调 */
  onEnd?: () => void
  /** 是否自动开始（默认 true） */
  autoStart?: boolean
  /** 每次 tick 的间隔（ms，默认 1000） */
  interval?: number
}

export interface UseCountdownReturn {
  /** 剩余秒数 */
  remaining: number
  /** 是否正在倒计时 */
  running: boolean
  /** 开始 / 恢复 */
  start: () => void
  /** 暂停 */
  pause: () => void
  /** 重置到初始 total 并停止 */
  reset: () => void
}

export function useCountdown(options: UseCountdownOptions): UseCountdownReturn {
  const { total, onEnd, autoStart = true, interval = 1000 } = options
  const [remaining, setRemaining] = useState(total)
  const [running, setRunning] = useState(autoStart)
  const onEndRef = useRef(onEnd)
  onEndRef.current = onEnd

  useEffect(() => {
    if (!running || remaining <= 0) return

    const timer = setInterval(() => {
      setRemaining((prev) => {
        const next = prev - 1
        if (next <= 0) {
          clearInterval(timer)
          setRunning(false)
          onEndRef.current?.()
          return 0
        }
        return next
      })
    }, interval)

    return () => clearInterval(timer)
  }, [running, remaining, interval])

  const start = useCallback(() => setRunning(true), [])
  const pause = useCallback(() => setRunning(false), [])
  const reset = useCallback(() => {
    setRunning(false)
    setRemaining(total)
  }, [total])

  return { remaining, running, start, pause, reset }
}

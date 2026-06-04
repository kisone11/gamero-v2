import { useEffect, useRef, useCallback } from 'react'

export interface UseInfiniteScrollOptions {
  /** 是否正在加载中 */
  loading: boolean
  /** 是否还有更多数据 */
  hasMore: boolean
  /** 加载更多数据的回调 */
  onLoadMore: () => void
  /** 触发加载的阈值（px，距离底部多远时触发，默认 200） */
  threshold?: number
  /** rootMargin，默认 '0px 0px 200px 0px'，与 threshold 二选一 */
  rootMargin?: string
}

export interface UseInfiniteScrollReturn {
  /** 绑定到滚动容器的 ref */
  sentinelRef: React.RefObject<HTMLDivElement | null>
}

export function useInfiniteScroll(options: UseInfiniteScrollOptions): UseInfiniteScrollReturn {
  const { loading, hasMore, onLoadMore, threshold = 200, rootMargin } = options
  const sentinelRef = useRef<HTMLDivElement | null>(null)
  const callbackRef = useRef(onLoadMore)
  callbackRef.current = onLoadMore

  const handleIntersect = useCallback(
    (entries: IntersectionObserverEntry[]) => {
      const [entry] = entries
      if (entry.isIntersecting && !loading && hasMore) {
        callbackRef.current()
      }
    },
    [loading, hasMore],
  )

  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return

    const margin = rootMargin ?? `0px 0px ${threshold}px 0px`
    const observer = new IntersectionObserver(handleIntersect, {
      rootMargin: margin,
      threshold: 0,
    })
    observer.observe(el)

    return () => observer.disconnect()
  }, [handleIntersect, threshold, rootMargin])

  return { sentinelRef }
}

import { useEffect, useRef, useState, type RefObject } from 'react'

export interface UseIntersectionObserverOptions {
  threshold?: number | number[]
  root?: Element | null
  rootMargin?: string
  triggerOnce?: boolean
}

export interface UseIntersectionObserverReturn {
  ref: RefObject<HTMLDivElement | null>
  isIntersecting: boolean
  intersectionRatio: number
}

export function useIntersectionObserver(
  options: UseIntersectionObserverOptions = {},
): UseIntersectionObserverReturn {
  const { threshold = 0, root = null, rootMargin = '0px', triggerOnce = false } = options
  const ref = useRef<HTMLDivElement | null>(null)
  const [isIntersecting, setIsIntersecting] = useState(false)
  const [intersectionRatio, setIntersectionRatio] = useState(0)

  useEffect(() => {
    const el = ref.current
    if (!el) return

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsIntersecting(entry.isIntersecting)
        setIntersectionRatio(entry.intersectionRatio)
        if (entry.isIntersecting && triggerOnce) {
          observer.unobserve(el)
        }
      },
      { threshold, root, rootMargin },
    )

    observer.observe(el)
    return () => observer.disconnect()
  }, [threshold, root, rootMargin, triggerOnce])

  return { ref, isIntersecting, intersectionRatio }
}

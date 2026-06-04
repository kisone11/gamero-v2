import { useState, useEffect } from 'react'

export interface WindowSize {
  width: number
  height: number
}

export function useWindowSize(): WindowSize {
  const [size, setSize] = useState<WindowSize>(() => ({
    width: typeof window !== 'undefined' ? window.innerWidth : 0,
    height: typeof window !== 'undefined' ? window.innerHeight : 0,
  }))

  useEffect(() => {
    let frameId: number
    const handleResize = () => {
      cancelAnimationFrame(frameId)
      frameId = requestAnimationFrame(() => {
        setSize({ width: window.innerWidth, height: window.innerHeight })
      })
    }

    window.addEventListener('resize', handleResize)
    return () => {
      window.removeEventListener('resize', handleResize)
      cancelAnimationFrame(frameId)
    }
  }, [])

  return size
}

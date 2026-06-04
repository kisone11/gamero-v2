import { useState, useCallback } from 'react'

export type UseToggleReturn = [boolean, () => void]

export function useToggle(initial = false): UseToggleReturn {
  const [on, setOn] = useState(initial)
  const toggle = useCallback(() => setOn((prev) => !prev), [])
  return [on, toggle]
}

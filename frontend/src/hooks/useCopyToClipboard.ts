import { useState, useCallback } from 'react'

export interface CopyState {
  value: string
  copied: boolean
}

export type UseCopyToClipboardReturn = [CopyState, (text: string) => Promise<void>]

export function useCopyToClipboard(resetInterval = 2000): UseCopyToClipboardReturn {
  const [state, setState] = useState<CopyState>({ value: '', copied: false })

  const copy = useCallback(
    async (text: string) => {
      try {
        await navigator.clipboard.writeText(text)
        setState({ value: text, copied: true })
        setTimeout(() => setState((prev) => ({ ...prev, copied: false })), resetInterval)
      } catch {
        setState({ value: text, copied: false })
      }
    },
    [resetInterval],
  )

  return [state, copy]
}

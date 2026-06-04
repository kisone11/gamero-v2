import { useEffect } from 'react'

const BASE_TITLE = 'GameRo'

export function useDocumentTitle(title?: string) {
  useEffect(() => {
    document.title = title ? `${title} — ${BASE_TITLE}` : BASE_TITLE
  }, [title])
}

export default useDocumentTitle

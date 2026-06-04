import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

interface PaginationProps {
  page: number
  pages: number
  onChange: (page: number) => void
}

export function Pagination({ page, pages, onChange }: PaginationProps) {
  if (pages <= 1) return null

  const range = getRange(page, pages)

  return (
    <nav className="flex items-center justify-center gap-1 py-8">
      <button
        onClick={() => onChange(page - 1)}
        disabled={page <= 1}
        className="p-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-white/[0.04] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
      >
        <ChevronLeft className="w-4 h-4" />
      </button>

      {range.map((p, i) =>
        p === '...' ? (
          <span key={`d-${i}`} className="w-9 h-9 flex items-center justify-center text-[13px] text-text-muted">...</span>
        ) : (
          <button
            key={p}
            onClick={() => onChange(p as number)}
            className={cn(
              'w-9 h-9 rounded-lg text-[13px] font-medium transition-all duration-150',
              p === page
                ? 'bg-white/[0.06] text-text-primary'
                : 'text-text-muted hover:text-text-primary hover:bg-white/[0.03]'
            )}
          >
            {p}
          </button>
        )
      )}

      <button
        onClick={() => onChange(page + 1)}
        disabled={page >= pages}
        className="p-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-white/[0.04] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
      >
        <ChevronRight className="w-4 h-4" />
      </button>
    </nav>
  )
}

function getRange(current: number, total: number): (number | '...')[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
  if (current <= 3) return [1, 2, 3, 4, '...', total]
  if (current >= total - 2) return [1, '...', total - 3, total - 2, total - 1, total]
  return [1, '...', current - 1, current, current + 1, '...', total]
}

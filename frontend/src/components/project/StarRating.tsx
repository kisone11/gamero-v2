import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'

export function StarRating({
  value, onChange, size = 'md', readonly = false,
}: {
  value: number
  onChange?: (v: number) => void
  size?: 'sm' | 'md' | 'lg'
  readonly?: boolean
}) {
  const sizeClass = size === 'sm' ? 'h-4 w-4' : size === 'lg' ? 'h-7 w-7' : 'h-5 w-5'
  return (
    <div className="flex items-center gap-0.5">
      {[1, 2, 3, 4, 5].map(i => (
        <button
          key={i}
          type="button"
          disabled={readonly}
          onClick={() => onChange?.(i)}
          className={cn(
            'transition-colors',
            readonly ? 'cursor-default' : 'cursor-pointer hover:scale-110',
            i <= value ? 'text-amber' : 'text-border-default',
          )}
        >
          <Star className={cn(sizeClass, 'fill-current')} />
        </button>
      ))}
    </div>
  )
}

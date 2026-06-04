import { cn } from '@/lib/utils'

export function Select({
  value,
  onChange,
  options,
  className,
}: {
  value: string
  onChange: (value: string) => void
  options: { value: string; label: string }[]
  className?: string
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={cn(
        'h-10 px-3 bg-surface-void border border-white/[0.04] rounded-xl',
        'text-text-primary text-body',
        'focus:outline-none focus:border-amber focus:shadow-[0_0_0_2px_rgba(245,166,35,0.2)]',
        className,
      )}
    >
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  )
}

import { cn } from '@/lib/utils'

export function AdminToolbar({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children?: React.ReactNode
}) {
  return (
    <div className="mb-4 flex flex-col gap-3 border-b border-white/[0.04] pb-4 lg:flex-row lg:items-end lg:justify-between">
      <div className="min-w-0">
        <h3 className="text-[15px] font-semibold text-text-primary">{title}</h3>
        {description && <p className="mt-1 max-w-2xl text-[12px] leading-5 text-text-muted">{description}</p>}
      </div>
      {children && <div className="flex shrink-0 flex-wrap items-center gap-2">{children}</div>}
    </div>
  )
}

export function AdminFilterGroup({
  options,
  value,
  onChange,
}: {
  options: Array<{ value: string; label: string; count?: number }>
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="flex max-w-full items-center gap-1 overflow-x-auto rounded-lg border border-white/[0.06] bg-white/[0.025] p-1">
      {options.map((option) => {
        const active = value === option.value
        return (
          <button
            key={option.value}
            onClick={() => onChange(option.value)}
            className={cn(
              'h-8 shrink-0 rounded-md px-3 text-[12px] font-medium transition-colors',
              active ? 'bg-amber text-surface-void' : 'text-text-muted hover:bg-white/[0.04] hover:text-text-secondary',
            )}
          >
            {option.label}
            {option.count !== undefined && (
              <span className={cn('ml-1.5 font-mono text-[10px]', active ? 'text-surface-void/70' : 'text-text-muted/70')}>
                {option.count}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}

export function AdminListPanel({ children }: { children: React.ReactNode }) {
  return <div className="overflow-hidden rounded-lg border border-white/[0.06] bg-surface-card/90">{children}</div>
}

export function AdminListHeader({ columns }: { columns: string[] }) {
  return (
    <div className="hidden border-b border-white/[0.04] bg-white/[0.025] px-4 py-2.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-text-muted lg:grid lg:grid-cols-[1fr_160px_180px]">
      {columns.map((column) => (
        <span key={column}>{column}</span>
      ))}
    </div>
  )
}

export function AdminListRow({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return <div className={cn('border-b border-white/[0.035] px-4 py-3 last:border-b-0', className)}>{children}</div>
}

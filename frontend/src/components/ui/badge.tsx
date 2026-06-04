import { cn } from '@/lib/utils'

export type BadgeVariant = 'default' | 'primary' | 'amber' | 'coral' | 'success' | 'warning' | 'danger' | 'info'

interface BadgeProps {
  variant?: BadgeVariant
  size?: 'sm' | 'md'
  children: React.ReactNode
  className?: string
}

const styles: Record<BadgeVariant, string> = {
  default: 'bg-white/[0.04] text-text-muted border-white/[0.04]',
  primary: 'bg-amber/10 text-amber border-amber/15',
  amber: 'bg-amber/10 text-amber border-amber/15',
  coral: 'bg-coral/10 text-coral border-coral/15',
  success: 'bg-success/10 text-success border-success/15',
  warning: 'bg-warning/10 text-warning border-warning/15',
  danger: 'bg-danger/10 text-danger border-danger/15',
  info: 'bg-cyan/10 text-cyan border-cyan/15',
}

export function Badge({ variant = 'default', size = 'sm', children, className }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center font-medium border rounded-md',
        size === 'sm' ? 'px-2 py-0.5 text-[10px]' : 'px-2.5 py-0.5 text-[11px]',
        styles[variant],
        className
      )}
    >
      {children}
    </span>
  )
}

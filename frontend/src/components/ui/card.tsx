import { cn } from '@/lib/utils'

interface CardProps {
  children: React.ReactNode
  className?: string
  hover?: boolean
  padding?: 'none' | 'sm' | 'md' | 'lg'
  overflow?: boolean
  group?: boolean
}

const paddings = { none: '', sm: 'p-3', md: 'p-4', lg: 'p-5' }

export function Card({ children, className, hover = true, padding = 'md', overflow = false, group = false }: CardProps) {
  return (
    <div
      className={cn(
        'bg-surface-card backdrop-blur-sm border border-white/[0.04] rounded-xl',
        paddings[padding],
        overflow && 'overflow-hidden',
        group && 'group',
        hover && 'hover:border-amber/12 hover:shadow-card-hover hover:-translate-y-[1px] transition-all duration-200',
        className
      )}
    >
      {children}
    </div>
  )
}

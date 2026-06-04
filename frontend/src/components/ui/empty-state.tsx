import { cn } from '@/lib/utils'
import { Button } from './button'
import { Gamepad2 } from 'lucide-react'

interface EmptyStateProps {
  icon?: React.ReactNode
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
  }
  className?: string
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center py-16 px-4 text-center', className)}>
      {icon ? (
        <div className="mb-5">{icon}</div>
      ) : (
        <div className="w-16 h-16 mb-5 rounded-2xl bg-white/[0.03] flex items-center justify-center">
          <span className="text-2xl opacity-20"><Gamepad2 className="w-8 h-8" /></span>
        </div>
      )}
      <h3 className="text-[15px] font-semibold text-text-secondary mb-1.5">{title}</h3>
      {description && (
        <p className="text-[13px] text-text-muted max-w-sm mb-6">{description}</p>
      )}
      {action && (
        <Button variant="secondary" size="sm" onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  )
}

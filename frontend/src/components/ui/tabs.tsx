import { cn } from '@/lib/utils'

interface Tab {
  value: string
  label: string
  count?: number
}

interface TabsProps {
  tabs: Tab[]
  value: string
  onChange: (value: string) => void
  className?: string
}

export function Tabs({ tabs, value, onChange, className }: TabsProps) {
  return (
    <div className={cn('flex border-b border-white/[0.04]', className)}>
      {tabs.map((tab) => (
        <button
          key={tab.value}
          onClick={() => onChange(tab.value)}
          className={cn(
            'relative px-4 py-3 text-[13px] font-medium transition-colors',
            value === tab.value
              ? 'text-text-primary'
              : 'text-text-muted hover:text-text-secondary'
          )}
        >
          {tab.label}
          {tab.count !== undefined && (
            <span className={cn(
              'ml-1.5 text-[11px]',
              value === tab.value ? 'text-text-muted' : 'text-text-muted/60'
            )}>
              {tab.count}
            </span>
          )}
          {value === tab.value && (
            <span className="absolute bottom-0 left-3 right-3 h-[2px] bg-amber rounded-full" />
          )}
        </button>
      ))}
    </div>
  )
}

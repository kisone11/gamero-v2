import { cn } from '@/lib/utils'
import type { RecruitmentStatus } from '@/types/enums'

const STATUS_MAP: Record<RecruitmentStatus, { label: string; pulseClass: string; textColor: string }> = {
  open:    { label: '招募中', pulseClass: 'status-pulse--active', textColor: 'text-success' },
  closed:  { label: '已关闭', pulseClass: 'status-pulse--idle',  textColor: 'text-warning' },
  expired: { label: '已过期', pulseClass: 'status-pulse--idle',  textColor: 'text-muted' },
}

export function StatusBadge({ status }: { status: RecruitmentStatus }) {
  const s = STATUS_MAP[status] ?? STATUS_MAP.closed
  return (
    <span className={cn('status-pulse text-[12px] font-mono', s.pulseClass, s.textColor)}>
      {s.label}
    </span>
  )
}

import { CheckCircle, Clock, Eye, XCircle, HelpCircle, Hourglass } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ApplicationStatus } from '@/types/enums'

export function ApplicationTimeline({ status }: { status: ApplicationStatus }) {
  const steps = [
    { key: 'submitted', label: '已提交', icon: CheckCircle, done: true },
    { key: 'viewed', label: '已查看', icon: Eye, done: status !== 'pending' },
    { key: 'reviewing', label: '审核中', icon: Clock, done: false, active: status === 'pending' },
    {
      key: 'result',
      label: status === 'approved' ? '已通过' : status === 'rejected' ? '未通过' : '待定',
      icon: status === 'approved' ? CheckCircle : status === 'rejected' ? XCircle : HelpCircle,
      done: status === 'approved' || status === 'rejected',
    },
  ]

  const pct = status === 'approved' || status === 'rejected' ? '84%' : status !== 'pending' ? '56%' : '28%'

  return (
    <div className="flex items-center justify-between py-4 relative">
      <div className="absolute top-1/2 left-[8%] right-[8%] h-[2px] bg-white/[0.04] -translate-y-1/2 z-0" />
      <div className="absolute top-1/2 left-[8%] h-[2px] -translate-y-1/2 z-0 transition-all duration-500"
        style={{ width: pct, background: 'linear-gradient(90deg, #5cb884, #f5a623)' }} />
      {steps.map((step) => (
        <div key={step.key} className="text-center relative z-10 flex flex-col items-center">
          <div className={cn('w-9 h-9 rounded-full flex items-center justify-center text-sm',
            step.done && 'bg-success text-white',
            step.active && !step.done && 'bg-amber/20 border-2 border-amber text-amber',
            !step.done && !step.active && 'bg-white/[0.03] border-2 border-white/[0.06] text-text-muted')}>
            {step.active && !step.done ? <Hourglass className="w-4 h-4 animate-pulse-dot" /> : <step.icon className="w-4 h-4" />}
          </div>
          <span className={cn('text-[10px] font-medium mt-1.5', step.done ? 'text-success' : step.active ? 'text-amber' : 'text-text-muted')}>
            {step.label}
          </span>
        </div>
      ))}
    </div>
  )
}

import { Link } from 'react-router-dom'
import { MapPin, Calendar, Users, Flame, Code2, Palette, ClipboardList, Music, Gamepad2 } from 'lucide-react'
import { StatusBadge } from './StatusBadge'
import { cn } from '@/lib/utils'
import type { RecruitmentListItem } from '@/types/api'
import type { RecruitmentPosition, CooperationType, RecruitmentStatus } from '@/types/enums'

const POSITION_META: Record<RecruitmentPosition, { label: string; icon: React.ComponentType<{ className?: string }>; color: string }> = {
  program: { label: '程序', icon: Code2, color: 'bg-blue-500/10 text-blue-400 border-blue-500/20' },
  art:     { label: '美术', icon: Palette, color: 'bg-pink-500/10 text-pink-400 border-pink-500/20' },
  design:  { label: '策划', icon: ClipboardList, color: 'bg-purple-500/10 text-purple-400 border-purple-500/20' },
  sound:   { label: '音效', icon: Music, color: 'bg-green-500/10 text-green-400 border-green-500/20' },
}

const COOP_LABELS: Record<CooperationType, string> = { online: '线上', offline: '线下', hybrid: '混合' }

function daysUntil(iso: string): number { return Math.ceil((new Date(iso).getTime() - Date.now()) / 86400000) }

function effectiveStatus(s: RecruitmentStatus, expireAt: string): RecruitmentStatus {
  return s === 'expired' || new Date(expireAt) < new Date() ? 'expired' : s
}

export function RecruitCard({ item, relationLabel }: { item: RecruitmentListItem; relationLabel?: '已参与' | '已申请' }) {
  const status = effectiveStatus(item.status, item.expire_at)
  const days = daysUntil(item.expire_at)
  const isUrgent = status === 'open' && days <= 3 && days > 0

  return (
    <Link to={`/recruit/${item.id}`}
      className="group relative block bg-surface-card border border-white/[0.04] rounded-xl p-5
                 hover:border-amber/20 hover:shadow-[0_4px_24px_rgba(0,0,0,0.25)]
                 hover:-translate-y-[1px] transition-all duration-200">
      {relationLabel && (
        <span className={cn(
          'absolute right-4 top-4 px-2.5 py-1 rounded-full text-[11px] font-semibold border',
          relationLabel === '已参与'
            ? 'bg-success/10 text-success border-success/20'
            : 'bg-amber/10 text-amber border-amber/20',
        )}>
          {relationLabel}
        </span>
      )}
      <div className="flex items-start gap-3">
        <div className="w-12 h-12 rounded-[10px] bg-gradient-to-br from-surface-deep to-surface-hover
                        flex items-center justify-center text-lg shrink-0 group-hover:scale-105 transition-transform duration-200">
          {item.cover_url
            ? <img src={item.cover_url} alt="" className="w-full h-full rounded-[10px] object-cover" />
            : (() => { const Ic = POSITION_META[item.position]?.icon; return Ic ? <Ic className="w-5 h-5" /> : <Gamepad2 className="w-5 h-5" /> })()}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 mb-1.5">
            <span className={cn('inline-block px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wider border rounded-md', POSITION_META[item.position]?.color)}>
              {POSITION_META[item.position]?.label}
            </span>
            {isUrgent
              ? <span className="status-pulse status-pulse--urgent text-[11px] text-warning font-mono">即将截止</span>
              : <StatusBadge status={status} />}
            {item.headcount > 2 && status === 'open' && (
              <span className="ml-auto flex items-center gap-0.5 text-[10px] text-amber font-mono"><Flame className="w-3 h-3" />热门</span>
            )}
          </div>
          <h3 className="text-[15px] font-semibold text-text-primary mb-1.5 group-hover:text-amber transition-colors">{item.project_name}</h3>
          {item.description && <p className="text-[13px] text-text-secondary leading-relaxed line-clamp-2 mb-2.5">{item.description}</p>}
          <div className="flex items-center gap-4 flex-wrap text-[12px] text-text-muted font-mono">
            <span className="flex items-center gap-1"><Users className="h-3.5 w-3.5" />需要 {item.headcount} 人</span>
            <span className="flex items-center gap-1"><MapPin className="h-3.5 w-3.5" />{COOP_LABELS[item.cooperation_type]}</span>
            <span className="flex items-center gap-1"><Calendar className="h-3.5 w-3.5" />{status === 'expired' ? '已过期' : `剩余 ${days} 天`}</span>
          </div>
        </div>
      </div>
    </Link>
  )
}

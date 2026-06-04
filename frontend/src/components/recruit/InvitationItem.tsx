import { Avatar, Badge } from '@/components/ui'
import { Clock, Crown } from 'lucide-react'
import type { InvitationDetail } from '@/types/api'
import type { InvitationStatus, RecruitmentPosition } from '@/types/enums'

const POS_LABELS: Record<RecruitmentPosition, string> = { program: '程序', art: '美术', design: '策划', sound: '音效' }
const STATUS_MAP: Record<InvitationStatus, { label: string; variant: 'warning' | 'success' | 'danger' | 'default' }> = {
  pending: { label: '待回复', variant: 'warning' }, accepted: { label: '已接受', variant: 'success' },
  declined: { label: '已拒绝', variant: 'danger' }, withdrawn: { label: '已撤销', variant: 'default' },
  expired: { label: '已过期', variant: 'default' },
}

function daysUntil(iso: string): number { return Math.ceil((new Date(iso).getTime() - Date.now()) / 86400000) }

interface Props { inv: InvitationDetail; actions?: React.ReactNode }

export function InvitationItem({ inv, actions }: Props) {
  const remaining = inv.expire_at ? daysUntil(inv.expire_at) : null
  const s = STATUS_MAP[inv.status] ?? STATUS_MAP.pending

  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 min-w-0 flex-1">
          <div className="relative shrink-0">
            <Avatar src={inv.inviter_avatar_url} name={inv.inviter_nickname} size="md" />
            <span className="absolute -bottom-0.5 -right-0.5 bg-surface-card rounded-full px-1 border border-amber/20"><Crown className="w-3 h-3 text-amber" /></span>
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <Badge variant={s.variant} size="sm">{s.label}</Badge>
              <span className="text-[11px] text-text-muted">{new Date(inv.created_at).toLocaleDateString('zh-CN')}</span>
            </div>
            <h3 className="text-[14px] font-semibold text-text-primary">
              <span className="text-amber">{inv.inviter_nickname}</span> 邀请你加入 <strong>{inv.project_name}</strong>
            </h3>
            <p className="text-[13px] text-text-secondary mt-0.5">职位：{POS_LABELS[inv.position] || inv.position}</p>
            {inv.message && (
              <div className="mt-2 p-3 bg-white/[0.02] border border-white/[0.03] rounded-lg text-[12px] text-text-secondary italic">"{inv.message}"</div>
            )}
            {remaining !== null && remaining > 0 && inv.status === 'pending' && (
              <p className="flex items-center gap-1 text-[11px] text-text-muted mt-2"><Clock className="w-3 h-3" />此邀请将在 {remaining} 天后过期</p>
            )}
          </div>
        </div>
        {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
      </div>
    </div>
  )
}

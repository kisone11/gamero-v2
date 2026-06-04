import { Link } from 'react-router-dom'
import { Avatar } from '@/components/ui'
import type { ApplicationDetail } from '@/types/api'
import type { RecruitmentPosition } from '@/types/enums'

const POS_LABELS: Record<RecruitmentPosition, string> = { program: '程序', art: '美术', design: '策划', sound: '音效' }

interface Props {
  app: ApplicationDetail
  actions?: React.ReactNode
  className?: string
}

export function ApplicationItem({ app, actions, className = '' }: Props) {
  return (
    <div className={`flex items-center gap-4 p-4 bg-surface-card border border-white/[0.04] rounded-xl ${className}`}>
      <Avatar src={app.applicant_avatar_url} name={app.applicant_nickname} size="md" userId={app.applicant_id} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <Link to={`/u/${app.applicant_username || app.applicant_id}`}
            className="text-[14px] font-semibold text-text-primary hover:text-amber transition-colors">
            {app.applicant_nickname}
          </Link>
          <span className="text-[11px] text-text-muted font-mono">@{app.applicant_username}</span>
        </div>
        {app.message && <p className="text-[13px] text-text-secondary mt-0.5">{app.message}</p>}
        <p className="text-[11px] text-text-muted mt-1">
          {new Date(app.created_at).toLocaleDateString('zh-CN')} 申请 · {POS_LABELS[app.position] || app.position}
        </p>
      </div>
      {actions && <div className="flex gap-2 shrink-0">{actions}</div>}
    </div>
  )
}

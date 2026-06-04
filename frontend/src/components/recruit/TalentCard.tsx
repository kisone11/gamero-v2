import { Link } from 'react-router-dom'
import { Mail, MapPin, Users, ExternalLink } from 'lucide-react'
import { Avatar, Badge } from '@/components/ui'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { TalentListItem } from '@/types/api'
import type { SkillLevel } from '@/types/enums'

const SKILL_LEVEL_LABELS: Record<SkillLevel, string> = { beginner: '入门', intermediate: '熟练', advanced: '精通' }
const SKILL_LEVEL_COLORS: Record<SkillLevel, string> = {
  advanced: 'bg-amber/10 text-amber border-amber/15',
  intermediate: 'bg-blue-500/8 text-blue-400 border-blue-500/12',
  beginner: 'bg-white/[0.03] text-text-muted border-white/[0.04]',
}

export function TalentCard({ talent, onInvite }: { talent: TalentListItem; onInvite: (t: TalentListItem) => void }) {
  return (
    <div className="bg-surface-card backdrop-blur-sm border border-white/[0.04] rounded-xl p-5 flex flex-col h-full
                    hover:border-amber/12 hover:shadow-[0_4px_24px_rgba(0,0,0,0.25)]
                    hover:-translate-y-[1px] transition-all duration-200 group">
      {/* Header: avatar + name + level */}
      <div className="flex items-center gap-3 mb-3">
        <div className="relative shrink-0">
          <Link to={`/u/${talent.username || talent.id}`}>
            <Avatar src={talent.avatar_url} name={talent.nickname} size="lg" />
          </Link>
          <span className={cn(
            'absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full border-2 border-surface-void',
            talent.is_available ? 'bg-success shadow-[0_0_6px_rgba(92,184,132,0.4)]' : 'bg-text-muted',
          )} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <Link to={`/u/${talent.username || talent.id}`} className="hover:text-amber transition-colors">
              <h3 className="text-[15px] font-semibold text-text-primary truncate group-hover:text-amber transition-colors">
                {talent.nickname}
              </h3>
            </Link>
            {talent.skills?.[0] && (
              <span className={cn('text-[9px] px-1.5 py-0.5 rounded font-semibold border', SKILL_LEVEL_COLORS[talent.skills[0].level] || '')}>
                {SKILL_LEVEL_LABELS[talent.skills[0].level]}
              </span>
            )}
          </div>
          <p className="text-[11px] text-text-muted font-mono">@{talent.username}</p>
        </div>
      </div>

      {/* Bio */}
      {talent.bio && (
        <p className="text-[13px] text-text-secondary leading-relaxed line-clamp-2 mb-2.5">{talent.bio}</p>
      )}

      {/* Skills */}
      {talent.skills.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mb-2.5 flex-1">
          {talent.skills.slice(0, 5).map((skill) => (
            <span
              key={skill.id}
              className={cn(
                'px-2 py-0.5 text-[10px] rounded-md border font-medium',
                SKILL_LEVEL_COLORS[skill.level] || '',
              )}
            >
              {skill.name}
              <span className="ml-1 opacity-60">{SKILL_LEVEL_LABELS[skill.level]}</span>
            </span>
          ))}
          {talent.skills.length > 5 && (
            <span className="text-[10px] text-text-muted self-center">+{talent.skills.length - 5}</span>
          )}
        </div>
      )}

      {/* Footer */}
      <div className="flex items-center justify-between mt-auto pt-3 border-t border-white/[0.04]">
        <div className="flex items-center gap-2 text-[11px] text-text-muted">
          <span className="flex items-center gap-1">
            <Users className="h-3 w-3" />
            {talent.project_count}
          </span>
          {talent.location && (
            <span className="flex items-center gap-0.5">
              <MapPin className="h-3 w-3" />
              {talent.location}
            </span>
          )}
          {talent.coop_preference && (
            <Badge variant="info" size="sm">
              {talent.coop_preference === 'online' ? '在线' : talent.coop_preference === 'offline' ? '线下' : '混合'}
            </Badge>
          )}
          {talent.is_available ? (
            <Badge variant="success" size="sm">可合作</Badge>
          ) : (
            <Badge variant="default" size="sm">暂不合作</Badge>
          )}
        </div>
        <div className="flex gap-1.5">
          <Link to={`/u/${talent.username || talent.id}`}>
            <Button variant="ghost" size="sm" className="text-[11px] px-2">
              <ExternalLink className="h-3 w-3" />
            </Button>
          </Link>
          <Button size="sm" variant="outline" onClick={() => onInvite(talent)}>
            <Mail className="h-3 w-3" /> 邀请
          </Button>
        </div>
      </div>
    </div>
  )
}

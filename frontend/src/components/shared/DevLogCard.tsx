import { Link } from 'react-router-dom'
import { ScrollText, Heart, MessageSquare, Eye } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Avatar } from '@/components/ui/avatar'
import { ImageGallery } from '@/components/shared/ImageGallery'
import { timeAgo } from '@/lib/time'
import type { DevLog, DevLogDetail } from '@/types/api'

interface DevLogCardProps {
  log: DevLogDetail | DevLog
  showAuthor?: boolean
  showProject?: boolean
}

export function DevLogCard({ log, showAuthor = false, showProject = false }: DevLogCardProps) {
  const author = (log as DevLogDetail).author
  const projectName = (log as DevLogDetail).project_name
  const projectSlug = (log as DevLogDetail).project_slug

  return (
    <Link to={`/devlog/${log.id}`}
      className="block bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden
                 hover:border-amber/12 hover:shadow-[0_4px_24px_rgba(0,0,0,0.25)]
                 hover:-translate-y-[1px] transition-all duration-200">
      <div className="p-5">
        <div className="flex items-start gap-4">
          <div className="w-10 h-10 rounded-xl bg-coral/10 flex items-center justify-center shrink-0 ring-1 ring-coral/20">
            <ScrollText className="h-5 w-5 text-coral" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 mb-1.5">
              <Badge variant={log.log_type === 'release' ? 'primary' : 'info'} size="sm">
                {log.log_type === 'release' ? '发布' : '日志'}
              </Badge>
              {showAuthor && author && (
                <>
                  <Avatar src={author.avatar_url} name={author.nickname} size="xs" />
                  <span className="text-[13px] text-text-muted">{author.nickname}</span>
                </>
              )}
              {showProject && projectName && (
                <Link
                  to={`/p/${projectSlug}`}
                  onClick={(e) => e.stopPropagation()}
                  className="text-[13px] text-amber hover:underline"
                >
                  {projectName}
                </Link>
              )}
              <span className="text-[12px] text-text-muted font-mono">{timeAgo(log.created_at)}</span>
            </div>
            <h3 className="text-[15px] font-semibold text-text-primary mb-1.5 line-clamp-1">{log.title}</h3>
            {log.content && (
              <p className="text-[13px] text-text-secondary line-clamp-2 mb-2.5">
                {(() => {
                  const text = (log.content || '').replace(/<[^>]*>/g, '')
                  return text.length > 150 ? text.slice(0, 150) + '...' : text
                })()}
              </p>
            )}
            <div className="flex items-center gap-4 text-[12px] text-text-muted mt-2">
              <span className="flex items-center gap-1"><Heart className="h-3 w-3" />{log.like_count}</span>
              <span className="flex items-center gap-1"><MessageSquare className="h-3 w-3" />{log.comment_count}</span>
              <span className="flex items-center gap-1"><Eye className="h-3 w-3" />{log.view_count}</span>
            </div>
          </div>
        </div>
      </div>
      {(log.image_urls?.length ?? 0) > 0 && (
        <div className="border-t border-white/[0.04]" onClick={(e) => e.preventDefault()}>
          <ImageGallery images={log.image_urls!} thumbnail maxShow={3} />
        </div>
      )}
    </Link>
  )
}

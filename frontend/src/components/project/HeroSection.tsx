import { ArrowLeft, Share2, Edit, Trash2, Heart, Eye, Users } from 'lucide-react'
import { Button, Badge } from '@/components/ui'
import { ReportDialog } from '@/components/shared/report-dialog'
import { PROJECT_STATUS_CONFIG, GENRE_LABELS } from '@/lib/constants'
import type { ProjectDetail } from '@/types/api'

export function HeroSection({
  project, isOwner, isFollowed, onFollow, onShare, onEdit, onDelete, followLoading, currentUser,
}: {
  project: ProjectDetail
  isOwner: boolean
  isFollowed?: boolean
  onFollow: () => void
  onShare: () => void
  onEdit: () => void
  onDelete: () => void
  followLoading: boolean
  currentUser: { id: number } | null
}) {
  const status = PROJECT_STATUS_CONFIG[project.status]

  return (
    <div className="relative overflow-hidden">
      {/* Cover */}
      <div className="relative h-72 md:h-96 w-full">
        {project.cover_url ? (
          <>
            <img src={project.cover_url} alt="" className="w-full h-full object-cover" />
            <div className="absolute inset-0 bg-gradient-to-t from-surface-void via-surface-void/50 to-transparent" />
          </>
        ) : (
          <div className="w-full h-full bg-gradient-to-br from-amber/20 via-surface-raised to-surface-card" />
        )}

        {/* Back */}
        <button
          onClick={() => window.history.back()}
          className="absolute top-4 left-4 p-2 rounded-xl bg-black/30 text-white/80 hover:text-white hover:bg-black/50 transition-colors"
          aria-label="返回"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>

        {/* Actions top-right */}
        <div className="absolute top-4 right-4 flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={onShare} className="text-white/80 hover:text-white hover:bg-white/10">
            <Share2 className="h-4 w-4" />
            <span className="hidden sm:inline ml-1">分享</span>
          </Button>
          {isOwner && (
            <>
              <Button variant="ghost" size="sm" onClick={onEdit} className="text-white/80 hover:text-white hover:bg-white/10">
                <Edit className="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="sm" onClick={onDelete} className="text-danger/80 hover:text-danger hover:bg-danger/10">
                <Trash2 className="h-4 w-4" />
              </Button>
            </>
          )}
          {!isOwner && currentUser && (
            <ReportDialog targetType="project" targetId={project.id} />
          )}
        </div>

        {/* Info overlay */}
        <div className="absolute bottom-0 left-0 right-0 p-6 md:p-10">
          <h1 className="text-[32px] font-bold text-white mb-2 drop-shadow-lg">{project.name}</h1>
          <p className="text-body text-white/80 max-w-2xl line-clamp-2 mb-4 drop-shadow">
            {project.description}
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={status.variant} size="md">{status.label}</Badge>
            <Badge variant="default" size="md">{GENRE_LABELS[project.genre]}</Badge>
            {project.style_tags?.slice(0, 5).map(tag => (
              <Badge key={tag} variant="info" size="sm">{tag}</Badge>
            ))}
          </div>
        </div>
      </div>

      {/* Stats bar */}
      <div className="bg-surface-card border-b border-white/[0.04]">
        <div className="max-w-6xl mx-auto px-6 py-3 flex items-center justify-between">
          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2 text-text-secondary">
              <Heart className="h-4 w-4 text-amber" />
              <span className="text-body font-mono font-semibold text-text-primary">{project.follower_count}</span>
              <span className="text-small text-text-muted">关注</span>
            </div>
            <div className="flex items-center gap-2 text-text-secondary">
              <Eye className="h-4 w-4" />
              <span className="text-body font-mono font-semibold text-text-primary">{project.view_count ?? '-'}</span>
              <span className="text-small text-text-muted">浏览</span>
            </div>
            <div className="flex items-center gap-2 text-text-secondary">
              <Users className="h-4 w-4" />
              <span className="text-body font-mono font-semibold text-text-primary">{project.members?.length ?? 0}</span>
              <span className="text-small text-text-muted">成员</span>
            </div>
          </div>
          <Button variant={isFollowed ? 'secondary' : 'primary'} size="sm" loading={followLoading} onClick={onFollow}>
            {isFollowed ? '已关注' : '关注项目'}
          </Button>
        </div>
      </div>
    </div>
  )
}

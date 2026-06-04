import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { FileText, Heart, MessageSquare, Eye, Plus } from 'lucide-react'
import { logApi } from '@/api/log'
import { Button, Badge, EmptyState, Pagination } from '@/components/ui'
import { timeAgo } from '@/lib/time'
import { ImageGallery } from '@/components/shared/ImageGallery'
import type { DevLogDetail } from '@/types/api'

export function ProjectLogsTab({
  projectId, isOwner, projectSlug,
}: {
  projectId: number
  isOwner?: boolean
  projectSlug?: string
}) {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)

  const { data, isLoading } = useQuery({
    queryKey: ['project-logs', projectId, page],
    queryFn: () => logApi.list({ project_id: projectId, page, page_size: 10 }),
  })

  if (isLoading) {
    return (
      <div className="py-6 space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="bg-surface-card border border-white/[0.04] rounded-xl p-4 space-y-3">
            <div className="flex items-center gap-2">
              <div className="w-5 h-5 rounded bg-white/[0.04] animate-pulse" />
              <div className="h-3 w-16 bg-white/[0.04] rounded animate-pulse" />
            </div>
            <div className="h-4 w-3/4 bg-white/[0.04] rounded animate-pulse" />
            <div className="h-3 w-full bg-white/[0.04] rounded animate-pulse" />
            <div className="flex gap-4">
              <div className="h-3 w-12 bg-white/[0.04] rounded animate-pulse" />
              <div className="h-3 w-12 bg-white/[0.04] rounded animate-pulse" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  const logs = data?.list ?? []
  const pages = data?.pages ?? 1

  if (logs.length === 0) {
    return (
      <div className="py-6">
        <EmptyState
          icon={<FileText className="w-7 h-7 text-text-muted" />}
          title="暂无开发日志"
          description="该项目还没有发布任何开发日志"
          action={isOwner && projectSlug ? { label: '写开发日志', onClick: () => navigate(`/projects/${projectId}/logs/new`) } : undefined}
        />
      </div>
    )
  }

  return (
    <div className="py-6 space-y-3">
      {isOwner && projectSlug && (
        <div className="flex justify-end">
          <Link to={`/projects/${projectId}/logs/new`}>
            <Button size="sm"><Plus className="w-3.5 h-3.5" />写开发日志</Button>
          </Link>
        </div>
      )}
      {logs.map(log => (
        <Link key={log.id} to={`/devlog/${log.id}`}
          className="block bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden hover:border-amber/12 hover:shadow-card-hover hover:-translate-y-[1px] transition-all duration-200">
          <div className="p-5">
            <div className="flex items-start gap-4">
              <div className="w-10 h-10 rounded-xl bg-coral/10 flex items-center justify-center shrink-0 ring-1 ring-coral/20">
                <FileText className="h-5 w-5 text-coral" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 mb-1.5">
                  <Badge variant={log.log_type === 'release' ? 'primary' : 'info'} size="sm">
                    {log.log_type === 'release' ? '发布' : '日志'}
                  </Badge>
                  <span className="text-small text-text-muted font-mono">{timeAgo(log.created_at)}</span>
                </div>
                <h3 className="text-h3 text-text-primary mb-1.5 line-clamp-1">{log.title}</h3>
                <div className="flex items-center gap-4 text-small text-text-muted mt-2">
                  <span className="flex items-center gap-1">
                    <Heart className="h-3 w-3" />{log.like_count}
                  </span>
                  <span className="flex items-center gap-1">
                    <MessageSquare className="h-3 w-3" />{log.comment_count}
                  </span>
                  <span className="flex items-center gap-1">
                    <Eye className="h-3 w-3" />{log.view_count}
                  </span>
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
      ))}
      <Pagination page={page} pages={pages} onChange={setPage} />
    </div>
  )
}

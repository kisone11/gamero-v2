import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Edit3, ScrollText, Heart, Eye, MessageSquare, Zap } from 'lucide-react'
import { logApi } from '@/api/log'
import { useAuthStore } from '@/stores/authStore'
import { Badge, Pagination } from '@/components/ui'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { timeAgo } from '@/lib/time'
import type { DevLogDetail } from '@/types/api'

// ============================================================
// Constants
// ============================================================

const STATUS_FILTERS = [
  { value: '' as const, label: '全部' },
  { value: 'published' as const, label: '已发布' },
  { value: 'draft' as const, label: '草稿' },
]

// ============================================================
// Helpers
// ============================================================

function LogSkeleton() {
  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-2">
      <div className="flex items-center gap-2">
        <div className="h-4 w-16 bg-white/[0.04] rounded animate-pulse" />
        <div className="h-4 w-16 bg-white/[0.04] rounded animate-pulse" />
      </div>
      <div className="h-5 w-3/4 bg-white/[0.04] rounded animate-pulse" />
      <div className="h-3 w-1/3 bg-white/[0.04] rounded animate-pulse" />
      <div className="flex gap-4">
        <div className="h-3 w-12 bg-white/[0.04] rounded animate-pulse" />
        <div className="h-3 w-12 bg-white/[0.04] rounded animate-pulse" />
        <div className="h-3 w-12 bg-white/[0.04] rounded animate-pulse" />
      </div>
    </div>
  )
}

// ============================================================
// MyLogsPage
// ============================================================

export default function MyLogsPage() {
  const me = useAuthStore((s) => s.user)
  const [status, setStatus] = useState<string>('')
  const [page, setPage] = useState(1)

  const query = useQuery({
    queryKey: ['my-logs', { status, page }],
    queryFn: () =>
      logApi.list({
        author_id: me?.id,
        page,
        page_size: 20,
        status: status || undefined,
        sort: 'latest',
      }),
    enabled: !!me?.id,
  })

  const logs = query.data?.list ?? []
  const pages = query.data?.pages ?? 1

  // Not logged in
  if (!me) {
    return (
      <div className="max-w-4xl mx-auto px-6 py-20 flex flex-col items-center justify-center text-center">
        <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
          <Zap className="w-7 h-7 text-text-muted" />
        </div>
        <p className="text-[15px] font-semibold text-text-secondary mb-1">请先登录</p>
        <p className="text-[13px] text-text-muted">需要登录才能查看我的日志</p>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto px-6 py-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-[22px] font-bold text-text-primary">我的日志</h1>
      </div>

      {/* Status filter */}
      <div className="flex items-center gap-2 mb-6">
        {STATUS_FILTERS.map((opt) => (
          <button
            key={opt.value}
            onClick={() => { setStatus(opt.value); setPage(1) }}
            className={cn(
              'px-4 py-2 text-[12px] font-semibold border rounded-xl transition-colors',
              status === opt.value
                ? 'bg-amber text-surface-void border-amber'
                : 'text-text-muted border-white/[0.04] hover:text-text-secondary',
            )}
          >
            {opt.label}
          </button>
        ))}
      </div>

      {/* Loading */}
      {query.isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => <LogSkeleton key={i} />)}
        </div>
      )}

      {/* Error */}
      {query.isError && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Zap className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
          <p className="text-[13px] text-text-muted mb-6">无法加载日志列表</p>
          <Button variant="secondary" size="sm" onClick={() => query.refetch()}>重试</Button>
        </div>
      )}

      {/* Empty */}
      {!query.isLoading && !query.isError && logs.length === 0 && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <ScrollText className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无日志</p>
          <p className="text-[13px] text-text-muted">
            {status === 'draft' ? '没有草稿日志' : status === 'published' ? '没有已发布的日志' : '还没有写过日志'}
          </p>
        </div>
      )}

      {/* List */}
      {!query.isLoading && !query.isError && logs.length > 0 && (
        <>
          <div className="space-y-3">
            {logs.map((log: any) => {
              const logDetail = log as DevLogDetail
              return (
                <div key={logDetail.id} className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <Badge variant={logDetail.log_type === 'release' ? 'success' : 'info'} size="sm">
                          {logDetail.log_type === 'release' ? '发布' : '日志'}
                        </Badge>
                        <Badge variant={logDetail.status === 'published' ? 'success' : 'warning'} size="sm">
                          {logDetail.status === 'published' ? '已发布' : '草稿'}
                        </Badge>
                        <span className="text-[12px] text-text-muted font-mono">{timeAgo(logDetail.created_at)}</span>
                      </div>

                      <h3 className="text-[15px] font-semibold text-text-primary line-clamp-1 mb-1">
                        {logDetail.title || '无标题'}
                      </h3>

                      {logDetail.project_name && (
                        <span className="text-[13px] text-amber">{logDetail.project_name}</span>
                      )}

                      {logDetail.content && (
                        <p className="text-[13px] text-text-secondary mt-1 line-clamp-1">
                          {logDetail.content.replace(/<[^>]*>/g, '').slice(0, 150)}
                        </p>
                      )}

                      <div className="flex items-center gap-3 mt-2 text-[12px] text-text-muted font-mono">
                        <span className="flex items-center gap-1">
                          <Heart className="h-3 w-3" /> {logDetail.like_count}
                        </span>
                        <span className="flex items-center gap-1">
                          <Eye className="h-3 w-3" /> {logDetail.view_count}
                        </span>
                        <span className="flex items-center gap-1">
                          <MessageSquare className="h-3 w-3" /> {logDetail.comment_count}
                        </span>
                      </div>
                    </div>

                    <Link to={`/devlog/${logDetail.id}/edit`} className="shrink-0">
                      <Button variant="outline" size="sm">
                        <Edit3 className="h-4 w-4" />编辑
                      </Button>
                    </Link>
                  </div>
                </div>
              )
            })}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}
    </div>
  )
}

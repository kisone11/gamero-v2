import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Flag, AlertTriangle } from 'lucide-react'
import { communityApi } from '@/api/community'
import { Pagination } from '@/components/ui'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { timeAgo } from '@/lib/time'
import type { Report } from '@/types/api'

// ============================================================
// Constants
// ============================================================

const TARGET_TYPE_LABELS: Record<string, string> = {
  post: '帖子',
  log: '开发日志',
  project: '项目',
  comment: '评论',
  user: '用户',
}

const REPORT_STATUS_CONFIG: Record<string, { label: string; variant: 'warning' | 'success' | 'default' | 'danger' }> = {
  pending: { label: '待处理', variant: 'warning' },
  escalated: { label: '已升级', variant: 'danger' },
  handled: { label: '已处理', variant: 'success' },
  rejected: { label: '已驳回', variant: 'default' },
}

function SkeletonCard() {
  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3">
      <div className="flex items-center gap-2">
        <div className="h-5 w-14 bg-white/[0.04] rounded animate-pulse" />
        <div className="h-5 w-14 bg-white/[0.04] rounded animate-pulse" />
        <div className="h-3 w-20 bg-white/[0.04] rounded animate-pulse" />
      </div>
      <div className="h-4 w-3/4 bg-white/[0.04] rounded animate-pulse" />
      <div className="h-3 w-1/3 bg-white/[0.04] rounded animate-pulse" />
    </div>
  )
}

// ============================================================
// MyReportsPage
// ============================================================

export default function MyReportsPage() {
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['my-reports', page],
    queryFn: () => communityApi.getMyReports(page, 20),
  })

  const reports = data?.list ?? []
  const pages = data?.pages ?? 1

  return (
    <div className="max-w-4xl mx-auto px-6 py-6">
      <h1 className="text-[22px] font-bold text-text-primary mb-6">我的举报</h1>

      {/* Loading */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <SkeletonCard key={i} />
          ))}
        </div>
      )}

      {/* Error */}
      {isError && !isLoading && (
        <div className="py-16 flex flex-col items-center justify-center text-center">
          <div className="w-12 h-12 mx-auto mb-3 rounded-xl bg-white/[0.03] flex items-center justify-center">
            <AlertTriangle className="w-5 h-5 text-text-muted" />
          </div>
          <p className="text-[13px] text-text-muted mb-4">加载失败</p>
          <Button variant="secondary" size="sm" onClick={() => refetch()}>重试</Button>
        </div>
      )}

      {/* Empty */}
      {!isLoading && !isError && reports.length === 0 && (
        <div className="py-16 flex flex-col items-center justify-center text-center">
          <div className="w-12 h-12 mx-auto mb-3 rounded-xl bg-white/[0.03] flex items-center justify-center">
            <Flag className="w-5 h-5 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无举报</p>
          <p className="text-[13px] text-text-muted">你还没有提交过举报</p>
        </div>
      )}

      {/* List */}
      {!isLoading && !isError && reports.length > 0 && (
        <>
          <div className="space-y-3">
            {reports.map((report: Report) => {
              const statusCfg = REPORT_STATUS_CONFIG[report.status] ?? { label: report.status, variant: 'default' as const }
              return (
                <div
                  key={report.id}
                  className="bg-surface-card border border-white/[0.04] rounded-xl p-5"
                >
                  <div className="flex items-start gap-4">
                    <div className="w-9 h-9 rounded-lg bg-white/[0.04] flex items-center justify-center shrink-0 mt-0.5">
                      <Flag className="w-4 h-4 text-coral" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap mb-1.5">
                        <Badge variant={statusCfg.variant} size="sm">
                          {statusCfg.label}
                        </Badge>
                        <Badge variant="default" size="sm">
                          {TARGET_TYPE_LABELS[report.target_type] ?? report.target_type}
                        </Badge>
                        <span className="text-[11px] text-text-muted font-mono">
                          #{report.id}
                        </span>
                      </div>
                      <p className="text-[14px] text-text-primary mb-1">
                        {report.reason}
                      </p>
                      {report.supplement && (
                        <p className="text-[13px] text-text-muted mb-1.5">
                          {report.supplement}
                        </p>
                      )}
                      <div className="flex items-center gap-3 text-[11px] text-text-muted">
                        <span>目标 ID: {report.target_id}</span>
                        <span>{timeAgo(report.created_at)}</span>
                        {report.handler_name && (
                          <span>处理人: {report.handler_name}</span>
                        )}
                      </div>
                    </div>
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

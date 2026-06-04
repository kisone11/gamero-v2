import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Flag, CheckCircle, XCircle, Ban, ChevronDown, ChevronRight, ExternalLink, AlertTriangle } from 'lucide-react'
import { adminApi } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'
import { Pagination } from '@/components/ui/pagination'
import { timeAgo } from '@/lib/time'
import { toast } from '@/stores/toastStore'
import { ReportSkeleton } from './AdminSkeletons'
import type { Report } from '@/types/api'

const TARGET_TYPE_LABELS: Record<string, string> = {
  post: '帖子',
  log: '开发日志',
  project: '项目',
  comment: '评论',
  user: '用户',
}

type ReportAction = 'resolve' | 'dismiss' | 'ban_project' | 'ban_user'
type ReportActionButton = {
  action: ReportAction
  label: string
  variant: 'primary' | 'ghost' | 'danger'
  icon: React.ReactNode
}

const REPORT_STATUS_CONFIG: Record<string, { label: string; variant: 'warning' | 'success' | 'default' | 'danger' }> = {
  pending: { label: '待处理', variant: 'warning' },
  escalated: { label: '已升级', variant: 'danger' },
  handled: { label: '已处理', variant: 'success' },
  rejected: { label: '已驳回', variant: 'default' },
}

const REPORT_STATUS_FILTERS = [
  { value: '', label: '全部' },
  { value: 'pending', label: '待处理' },
  { value: 'handled', label: '已处理' },
  { value: 'rejected', label: '已驳回' },
]

function ViewContentLink({ report }: { report: Report }) {
  const navigate = useNavigate()

  const handleClick = async (e: React.MouseEvent) => {
    e.preventDefault()
    const t = report.target_type
    if (t === 'post') navigate(`/post/${report.target_id}`)
    else if (t === 'log') navigate(`/devlog/${report.target_id}`)
    else if (t === 'user') navigate(`/u/${report.target_id}`)
    else if (t === 'project') navigate(`/p/${report.target_id}`)
    else if (t === 'comment') navigate(`/post/${report.target_id}`)
  }

  return (
    <button
      onClick={handleClick}
      className="inline-flex items-center gap-1 text-caption text-amber hover:text-amber-dim transition-colors mb-1.5 cursor-pointer bg-transparent border-none p-0"
    >
      <ExternalLink className="w-3 h-3" /> 查看被举报内容
    </button>
  )
}

export function ReportsTab() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const [noteInputs, setNoteInputs] = useState<Record<number, string>>({})
  const [loadingReportIds, setLoadingReportIds] = useState<Set<number>>(new Set())

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'reports', page, statusFilter],
    queryFn: () =>
      adminApi.listReports({
        page,
        page_size: 15,
        status: (statusFilter || undefined) as 'pending' | 'handled' | 'rejected' | undefined,
      }),
  })

  const reports = data?.list ?? []
  const pages = data?.pages ?? 1

  const handleMut = useMutation({
    mutationFn: async ({ id, action, note, targetType, targetId }: { id: number; action: ReportAction; note?: string; targetType?: string; targetId?: number }) => {
      setLoadingReportIds((prev) => new Set(prev).add(id))
      try {
        if (action === 'ban_project' && targetType === 'project' && targetId) {
          await adminApi.banProject(targetId)
          await adminApi.handleReport(id, 'resolve', note)
        } else {
          await adminApi.handleReport(id, action, note)
        }
      } finally {
        setLoadingReportIds((prev) => { const next = new Set(prev); next.delete(id); return next })
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'reports'] })
      toast.success('操作成功')
      setExpandedId(null)
      setNoteInputs({})
    },
    onError: (err: unknown) => toast.error((err as { message?: string })?.message || '操作失败'),
  })

  const getActions = (report: Report): ReportActionButton[] => {
    const base: ReportActionButton[] = [
      { action: 'resolve' as const, label: '标记处理', variant: 'primary' as const, icon: <CheckCircle className="h-3.5 w-3.5" /> },
      { action: 'dismiss' as const, label: '驳回', variant: 'ghost' as const, icon: <XCircle className="h-3.5 w-3.5" /> },
    ]
    if (report.target_type === 'project') {
      base.push({ action: 'ban_project' as const, label: '封禁项目', variant: 'danger' as const, icon: <Ban className="h-3.5 w-3.5" /> })
    } else {
      base.push({ action: 'ban_user' as const, label: '封禁用户', variant: 'danger' as const, icon: <Ban className="h-3.5 w-3.5" /> })
    }
    return base
  }

  const toggleExpand = (id: number) => {
    setExpandedId(expandedId === id ? null : id)
  }

  return (
    <div>
      {/* Status filter tabs */}
      <div className="flex items-center gap-1 mb-4">
        {REPORT_STATUS_FILTERS.map((f) => (
          <button
            key={f.value}
            onClick={() => { setStatusFilter(f.value); setPage(1); setExpandedId(null) }}
            className={`px-3 py-1.5 text-small font-medium rounded-lg transition-colors ${
              statusFilter === f.value
                ? 'bg-white/[0.06] text-text-primary'
                : 'text-text-muted hover:text-text-secondary hover:bg-white/[0.03]'
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <ReportSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Error */}
      {isError && !isLoading && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <AlertTriangle className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-h3 text-text-secondary mb-1">加载失败</p>
          <p className="text-body text-text-muted mb-6">无法加载举报列表，请重试</p>
          <Button variant="secondary" size="sm" onClick={() => refetch()}>重新加载</Button>
        </div>
      )}

      {/* Empty */}
      {!isLoading && !isError && reports.length === 0 && (
        <EmptyState
          icon={<Flag className="w-7 h-7 text-text-muted" />}
          title="暂无举报"
          description="当前没有待处理的举报"
        />
      )}

      {/* List */}
      {!isLoading && !isError && reports.length > 0 && (
        <>
          <div className="space-y-2">
            {reports.map((report: Report) => {
              const statusCfg = REPORT_STATUS_CONFIG[report.status] ?? { label: report.status, variant: 'default' as const }
              const isExpanded = expandedId === report.id
              const currentNote = noteInputs[report.id] ?? ''

              return (
                <Card key={report.id} padding="md" hover={false}>
                  <div className="flex items-start gap-4">
                    <div className="w-8 h-8 rounded-lg bg-white/[0.04] flex items-center justify-center shrink-0 mt-0.5">
                      <Flag className="w-4 h-4 text-coral" />
                    </div>
                    <div className="flex-1 min-w-0">
                      {/* Header row */}
                      <div className="flex items-center gap-2 flex-wrap mb-1">
                        <Badge variant={statusCfg.variant} size="sm">{statusCfg.label}</Badge>
                        <Badge variant="default" size="sm">
                          {TARGET_TYPE_LABELS[report.target_type] ?? report.target_type}
                        </Badge>
                        <span className="text-caption text-text-muted font-mono">
                          #{report.id}
                        </span>
                        <button
                          onClick={() => toggleExpand(report.id)}
                          className="ml-auto flex items-center gap-1 text-caption text-text-muted hover:text-text-secondary transition-colors"
                        >
                          {isExpanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
                          {isExpanded ? '收起' : '详情'}
                        </button>
                      </div>

                      {/* View content link */}
                      <ViewContentLink report={report} />

                      {/* Reason */}
                      <p className="text-body text-text-primary mb-1 line-clamp-2">
                        {report.reason}
                      </p>

                      {/* Suppliment (always shown if not expanded, truncated) */}
                      {report.supplement && !isExpanded && (
                        <p className="text-small text-text-muted mb-1 italic line-clamp-1">
                          补充说明: {report.supplement}
                        </p>
                      )}

                      {/* Meta info */}
                      <div className="flex items-center gap-3 text-caption text-text-muted">
                        <span>举报者 ID: {report.reporter_id}</span>
                        <span>目标 ID: {report.target_id}</span>
                        <span>{timeAgo(report.created_at)}</span>
                      </div>

                      {/* Handled-by info */}
                      {report.handler_name && (
                        <p className="text-caption text-text-muted mt-1">
                          处理人: {report.handler_name}
                        </p>
                      )}

                      {/* Existing note display */}
                      {report.note && !isExpanded && (
                        <p className="text-caption text-text-muted mt-1 italic">
                          处理备注: {report.note}
                        </p>
                      )}

                      {/* ========== Expanded section ========== */}
                      {isExpanded && (
                        <div className="mt-3 pt-3 border-t border-white/[0.04] space-y-3">
                          {/* Full supplement */}
                          {report.supplement && (
                            <div>
                              <p className="text-caption text-text-muted mb-1 font-medium">补充说明</p>
                              <p className="text-body text-text-primary bg-white/[0.02] rounded-lg px-3 py-2">
                                {report.supplement}
                              </p>
                            </div>
                          )}

                          {/* Full note display */}
                          {report.note && (
                            <div>
                              <p className="text-caption text-text-muted mb-1 font-medium">处理备注</p>
                              <p className="text-body text-text-primary bg-white/[0.02] rounded-lg px-3 py-2">
                                {report.note}
                              </p>
                            </div>
                          )}

                          {/* Action note textarea + buttons */}
                          {report.status === 'pending' && (
                            <div className="space-y-3">
                              <textarea
                                placeholder="处理备注（选填）"
                                value={currentNote}
                                onChange={(e) => setNoteInputs(prev => ({ ...prev, [report.id]: e.target.value }))}
                                rows={2}
                                className="w-full px-3 py-2 bg-surface-void border border-white/[0.06] rounded-lg text-body text-text-primary placeholder:text-text-muted resize-none focus:outline-none focus:border-amber/40"
                              />
                              <div className="flex items-center gap-2">
                                {getActions(report).map((act) => (
                                  <Button
                                    key={act.action}
                                    variant={act.variant}
                                    size="sm"
                                    loading={loadingReportIds.has(report.id)}
                                    onClick={() => handleMut.mutate({ id: report.id, action: act.action, note: currentNote.trim() || undefined, targetType: report.target_type, targetId: report.target_id })}
                                  >
                                    {act.icon}
                                    {act.label}
                                  </Button>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                </Card>
              )
            })}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}
    </div>
  )
}

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Send, Zap, ChevronRight, X, CheckCircle2 } from 'lucide-react'
import { recruitApi } from '@/api/recruit'
import { Button, Pagination } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'
import { cn } from '@/lib/utils'
import { timeAgo } from '@/lib/time'
import type { ApplicationDetail } from '@/types/api'
import type { ApplicationStatus, RecruitmentPosition } from '@/types/enums'

const POS_LABELS: Record<RecruitmentPosition, string> = { program: '程序', art: '美术', design: '策划', sound: '音效' }
const STATUS_LABELS: Record<ApplicationStatus, string> = { pending: '审核中', approved: '已通过', rejected: '未通过', withdrawn: '已撤销' }

function Skeleton() { return <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3 skeleton-shimmer"><div className="flex items-center gap-2"><div className="h-4 w-14 rounded bg-white/[0.04]" /><div className="h-3 w-16 rounded bg-white/[0.04]" /></div><div className="h-4 w-3/4 rounded bg-white/[0.04]" /><div className="h-3 w-1/2 rounded bg-white/[0.04]" /></div> }

export default function MyApplicationsPage() {
  const qc = useQueryClient()
  const [page, setPage] = useState(1)
  const [withdrawTargetId, setWithdrawTargetId] = useState<number | null>(null)
  const query = useQuery({ queryKey: ['my-applications', page], queryFn: () => recruitApi.listMyApplications(page, 20) })
  const apps = query.data?.list ?? []
  const pages = query.data?.pages ?? 1

  const withdrawMut = useMutation({
    mutationFn: (id: number) => recruitApi.withdrawApplication(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['my-applications'] }); toast.success('申请已撤销') },
    onError: (e: any) => toast.error(e?.message || '撤销失败'),
  })

  return (
    <div className="max-w-4xl mx-auto px-6 py-6 animate-fade-in">
      <h1 className="text-[22px] font-bold text-text-primary mb-6">我的申请</h1>
      {query.isLoading && <div className="space-y-3">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} />)}</div>}
      {query.isError && (
        <div className="py-16 flex flex-col items-center justify-center text-center">
          <Zap className="w-6 h-6 text-text-muted mb-3" /><p className="text-[13px] text-text-muted mb-4">加载失败</p>
          <Button variant="secondary" size="sm" onClick={() => query.refetch()}>重试</Button>
        </div>
      )}
      {!query.isLoading && !query.isError && apps.length === 0 && (
        <div className="py-16 flex flex-col items-center justify-center text-center">
          <Send className="w-6 h-6 text-text-muted mb-3" /><p className="text-[15px] font-semibold text-text-secondary mb-1">暂无申请</p>
          <p className="text-[13px] text-text-muted mb-4">你还没有发送任何申请</p>
          <Link to="/recruit"><Button variant="secondary" size="sm">去招募广场</Button></Link>
        </div>
      )}
      {!query.isLoading && !query.isError && apps.length > 0 && (
        <>
          <div className="space-y-3">
            {apps.map((app: ApplicationDetail & { project_name?: string; message?: string }, i: number) => (
              <div key={app.id}
                className={cn('rounded-xl p-4 border animate-slide-up',
                  app.status === 'approved' ? 'bg-success/[0.03] border-success/10' : app.status === 'rejected' ? 'bg-surface-card border-white/[0.04] opacity-70' : 'bg-surface-card border-white/[0.04]')}
                style={{ animationDelay: `${i * 50}ms`, animationFillMode: 'backwards' }}>
                <div className="flex items-center justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 mb-1.5">
                      {app.status === 'pending' ? <span className="status-pulse status-pulse--urgent text-[11px] text-warning font-mono">审核中</span>
                        : app.status === 'approved' ? <span className="text-[11px] text-success font-semibold"><CheckCircle2 className="w-3.5 h-3.5 inline mr-0.5" />已通过</span>
                        : <span className="text-[11px] font-mono text-text-muted">{STATUS_LABELS[app.status]}</span>}
                      <span className="text-[11px] text-text-muted">{timeAgo(app.created_at)}</span>
                    </div>
                    <h3 className="text-[14px] font-semibold text-text-primary">{app.project_name || `项目 #${app.recruitment_id}`}</h3>
                    <p className="text-[13px] text-text-secondary">申请职位：{POS_LABELS[app.position] || app.position}</p>
                    {app.message && <p className="text-[12px] text-text-muted mt-1 line-clamp-1">{app.message}</p>}
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    {app.status === 'pending' && (
                      <Button variant="outline" size="sm" className="text-danger"
                        onClick={() => setWithdrawTargetId(app.id)}><X className="h-3.5 w-3.5" /> 撤销</Button>
                    )}
                    {app.status === 'approved' && (
                      <Link to={`/recruit/${app.recruitment_id}`}><Button variant="outline" size="sm"><ChevronRight className="h-3.5 w-3.5" /> 查看招募</Button></Link>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
          {/* Withdraw confirmation dialog */}
          <Dialog open={withdrawTargetId !== null} onOpenChange={(o) => { if (!o) setWithdrawTargetId(null) }}>
            <DialogContent>
              <DialogHeader><DialogTitle>撤销申请</DialogTitle><DialogDescription>确定要撤销此申请吗？</DialogDescription></DialogHeader>
              <DialogFooter>
                <Button variant="ghost" onClick={() => setWithdrawTargetId(null)}>取消</Button>
                <Button variant="danger" loading={withdrawMut.isPending} onClick={() => {
                  if (withdrawTargetId !== null) {
                    withdrawMut.mutate(withdrawTargetId)
                    setWithdrawTargetId(null)
                  }
                }}>确认撤销</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </>
      )}
    </div>
  )
}

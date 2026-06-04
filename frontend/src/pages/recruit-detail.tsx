import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, MapPin, Calendar, Send, XCircle, CheckCircle, Trash2, RotateCcw, ChevronRight, Code2, Palette, ClipboardList, Music, Gamepad2, Search, ClipboardCheck, PartyPopper, Sparkles, Inbox, Edit } from 'lucide-react'
import { recruitApi } from '@/api/recruit'
import { useAuthStore } from '@/stores/authStore'
import { Button, Card, Skeleton, Badge } from '@/components/ui'
import { Textarea } from '@/components/ui/textarea'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { StatusBadge } from '@/components/recruit/StatusBadge'
import { ApplicationItem } from '@/components/recruit/ApplicationItem'
import { ApplicationTimeline } from '@/components/recruit/ApplicationTimeline'
import { toast } from '@/stores/toastStore'
import { cn } from '@/lib/utils'
import type { RecruitmentPosition, CooperationType, RecruitmentStatus } from '@/types/enums'
import type { ApplicationDetail } from '@/types/api'

const POS_META: Record<RecruitmentPosition, { label: string; icon: React.ComponentType<{ className?: string }> }> = {
  program: { label: '程序', icon: Code2 }, art: { label: '美术', icon: Palette },
  design: { label: '策划', icon: ClipboardList }, sound: { label: '音效', icon: Music },
}
const COOP_LABELS: Record<CooperationType, string> = { online: '线上', offline: '线下', hybrid: '混合' }

function daysUntil(iso: string): number { return Math.ceil((new Date(iso).getTime() - Date.now()) / 86400000) }
function effStatus(d: { status: RecruitmentStatus; expire_at: string }): RecruitmentStatus {
  return d.status === 'expired' || new Date(d.expire_at).getTime() < Date.now() ? 'expired' : d.status
}

export default function RecruitDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const me = useAuthStore((s) => s.user)

  const [applyOpen, setApplyOpen] = useState(false)
  const [message, setMessage] = useState('')
  const [applied, setApplied] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [rejectingId, setRejectingId] = useState<number | null>(null)

  const { data: d, isLoading } = useQuery({
    queryKey: ['recruitment', id], queryFn: () => recruitApi.get(Number(id)), enabled: !!id,
  })
  const isOwner = !!d && !!me && me.id === d.owner_id
  const status = d ? effStatus(d) : 'open'
  const isOpen = status === 'open'

  const { data: apps } = useQuery({
    queryKey: ['recruitment-applications', id], queryFn: () => recruitApi.getApplications(Number(id)), enabled: !!id && isOwner,
  })

  const applyMut = useMutation({
    mutationFn: () => recruitApi.apply(Number(id), { position: d!.position, message: message.trim() || undefined }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['recruitment', id] }); setApplied(true); toast.success('申请已发送') },
    onError: (e: any) => toast.error(e?.response?.data?.message || '申请失败'),
  })
  const closeMut = useMutation({
    mutationFn: () => recruitApi.close(Number(id)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['recruitment', id] }); toast.success('招募已关闭') },
    onError: () => toast.error('操作失败'),
  })
  const reopenMut = useMutation({
    mutationFn: () => recruitApi.reopen(Number(id)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['recruitment', id] }); toast.success('招募已重新开启') },
    onError: () => toast.error('操作失败'),
  })
  const deleteMut = useMutation({
    mutationFn: () => recruitApi.delete(Number(id)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['recruitments'] }); toast.success('招募已删除'); navigate('/recruit') },
    onError: () => toast.error('删除失败'),
  })
  const approveMut = useMutation({
    mutationFn: (appId: number) => recruitApi.approveApplication(appId),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['recruitment-applications', id] }); toast.success('已通过') },
    onError: () => toast.error('操作失败'),
  })
  const rejectMut = useMutation({
    mutationFn: (appId: number) => recruitApi.rejectApplication(appId),
    onSuccess: () => { setRejectingId(null); qc.invalidateQueries({ queryKey: ['recruitment-applications', id] }); toast.success('已拒绝') },
    onError: () => toast.error('操作失败'),
  })

  if (isLoading) return (
    <div className="max-w-3xl mx-auto px-6 py-8 space-y-4">
      <Skeleton className="h-5 w-28" /><Skeleton className="h-48 w-full rounded-xl" />
      <div className="grid grid-cols-4 gap-3">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-16 rounded-xl" />)}</div>
      <Skeleton className="h-32 w-full rounded-xl" />
    </div>
  )
  if (!d) return (
    <div className="max-w-3xl mx-auto px-6 py-16 text-center">
      <Search className="w-10 h-10 text-text-muted mb-4 mx-auto" />
      <p className="text-[15px] text-text-muted mb-2">招募不存在或已删除</p>
      <Link to="/recruit" className="text-amber text-[13px] hover:underline">返回招募广场</Link>
    </div>
  )

  const appList = (apps?.list ?? []) as ApplicationDetail[]

  return (
    <div className="max-w-3xl mx-auto px-6 py-8">
      <Link to="/recruit" className="inline-flex items-center gap-1.5 text-[13px] text-text-muted hover:text-amber transition-colors mb-6">
        <ArrowLeft className="w-4 h-4" /> 招募广场
      </Link>

      {d.project_id > 0 && (
        <Link to={`/p/${d.project_slug || d.project_id}`}
          className="flex items-center gap-3 mb-6 p-4 bg-surface-card border border-white/[0.04] rounded-xl hover:border-amber/20 transition-all duration-200 group">
          <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-surface-deep to-surface-hover flex items-center justify-center text-xl shrink-0">
            {d.cover_url ? <img src={d.cover_url} alt="" className="w-full h-full rounded-lg object-cover" /> : (() => { const Ic = POS_META[d.position]?.icon; return Ic ? <Ic className="w-5 h-5" /> : <Gamepad2 className="w-5 h-5" /> })()}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-[11px] text-text-muted uppercase tracking-wider">所属项目</p>
            <p className="text-[15px] font-semibold text-text-primary group-hover:text-amber transition-colors">{d.project_name || `项目 #${d.project_id}`}</p>
          </div>
          <ChevronRight className="w-4 h-4 text-text-muted group-hover:text-amber transition-colors" />
        </Link>
      )}

      <div className="flex items-start justify-between mb-2">
        <h1 className="text-[22px] font-bold text-text-primary">招募 {POS_META[d.position]?.label || d.position}<span className="text-[14px] font-normal text-text-muted ml-2">×{d.headcount}</span></h1>
        <StatusBadge status={status} />
      </div>
      <div className="flex items-center gap-4 text-[13px] text-text-muted mb-6">
        <span className="flex items-center gap-1"><MapPin className="w-3.5 h-3.5" />{COOP_LABELS[d.cooperation_type] || d.cooperation_type}</span>
        <span className="flex items-center gap-1"><Calendar className="w-3.5 h-3.5" />{new Date(d.expire_at).toLocaleDateString('zh-CN')} 截止</span>
        <span>{new Date(d.created_at).toLocaleDateString('zh-CN')} 发布</span>
      </div>

      {isOwner && (
        <div className="flex gap-2 mb-6">
          <Link to={`/recruit/${d.id}/edit`}>
            <Button variant="outline" size="sm" icon={<Edit className="w-3.5 h-3.5" />}>编辑招募</Button>
          </Link>
          {isOpen ? (
            <Button variant="outline" size="sm" loading={closeMut.isPending} onClick={() => closeMut.mutate()} icon={<XCircle className="w-3.5 h-3.5" />}>关闭招募</Button>
          ) : status === 'closed' ? (
            <Button variant="outline" size="sm" loading={reopenMut.isPending} onClick={() => reopenMut.mutate()} icon={<RotateCcw className="w-3.5 h-3.5" />}>重新开启</Button>
          ) : null}
          <Button variant="ghost" size="sm" onClick={() => setDeleteOpen(true)} icon={<Trash2 className="w-3.5 h-3.5 text-danger" />}>删除</Button>
        </div>
      )}

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
        {[['合作方式', COOP_LABELS[d.cooperation_type]], ['联系方式', d.contact_type === 'platform' ? '平台私信' : `${d.contact_type === 'wechat' ? '微信' : 'QQ'}: ${d.contact_info || '未填写'}`], ['需求人数', `${d.headcount} 人`], ['有效期', status === 'expired' ? '已过期' : `${Math.max(0, daysUntil(d.expire_at))} 天`]].map(([l, v]) => (
          <div key={l} className="bg-surface-card border border-white/[0.04] rounded-xl p-3">
            <p className="text-[10px] text-text-muted uppercase tracking-wider mb-0.5">{l}</p>
            <p className="text-[13px] font-semibold text-text-primary">{v}</p>
          </div>
        ))}
      </div>

      {d.description && (
        <Card className="mb-6">
          <h2 className="text-[15px] font-semibold text-text-primary mb-3">需求描述</h2>
          <p className="text-[14px] text-text-secondary leading-relaxed whitespace-pre-wrap">{d.description}</p>
        </Card>
      )}

      {!isOwner && isOpen && me && !applied && (
        <div className="mb-6">
          <Button onClick={() => setApplyOpen(!applyOpen)} icon={<Send className="w-4 h-4" />}>申请加入</Button>
          {applyOpen && (
            <div className="mt-4 bg-surface-card border border-white/[0.04] rounded-xl p-5 animate-slide-up">
              <div className="bg-amber/[0.04] border border-amber/10 rounded-lg p-3 mb-4 flex items-start gap-3">
                <ClipboardCheck className="w-5 h-5 shrink-0 text-amber" />
                <div>
                  <p className="text-[13px] font-semibold text-text-primary">申请前确认</p>
                  <p className="text-[12px] text-text-secondary mt-0.5">
                    你正在申请 <strong className="text-text-primary">{d.project_name}</strong> 的 <strong className="text-amber">{POS_META[d.position]?.label}</strong> 岗位 · {COOP_LABELS[d.cooperation_type]}合作
                  </p>
                </div>
              </div>
              <div className="mb-4">
                <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider block mb-1.5">申请留言</label>
                <Textarea placeholder="介绍一下你的技能和相关经验..." value={message} onChange={(e) => setMessage(e.target.value)} rows={3} maxLength={500} />
                <p className="text-right text-[11px] text-text-muted mt-1">{message.length}/500</p>
              </div>
              <div className="flex gap-3">
                <Button variant="outline" size="sm" onClick={() => setApplyOpen(false)}>取消</Button>
                <Button size="sm" loading={applyMut.isPending} onClick={() => applyMut.mutate()} icon={<Sparkles className="w-4 h-4" />}>提交申请</Button>
              </div>
            </div>
          )}
        </div>
      )}

      {!isOwner && isOpen && me && applied && (
        <div className="mb-6 bg-success/[0.04] border border-success/10 rounded-xl p-6 text-center animate-bounce-in">
          <PartyPopper className="w-8 h-8 text-success mb-2 mx-auto" />
          <h3 className="text-[16px] font-bold text-success mb-1">申请已发送！</h3>
          <p className="text-[13px] text-text-secondary mb-4">项目负责人将在 3-7 天内回复</p>
          <div className="flex gap-3 justify-center">
            <Link to="/me/applications"><Button variant="primary" size="sm" icon={<ClipboardCheck className="w-4 h-4" />}>查看我的申请</Button></Link>
            <Link to="/recruit"><Button variant="secondary" size="sm">← 返回招募广场</Button></Link>
          </div>
        </div>
      )}

      {!isOwner && isOpen && !me && <Link to="/login"><Button variant="secondary" className="mb-6">登录后申请</Button></Link>}

      {isOwner && (
        <div className="mt-8">
          <h2 className="text-[16px] font-bold text-text-primary mb-4">申请列表 {apps?.total != null ? `(${apps.total})` : ''}</h2>
          {appList.length > 0 ? (
            <div className="space-y-3">
              {appList.map((app) => (
                <div key={app.id} className={cn('rounded-xl border transition-all duration-300', app.status === 'approved' ? 'bg-success/[0.03] border-success/10' : 'bg-surface-card border-white/[0.04]')}>
                  <ApplicationItem app={app} actions={
                    app.status === 'pending' ? (
                      <div className="flex gap-2">
                        <Button variant="primary" size="sm" loading={approveMut.isPending} onClick={() => approveMut.mutate(app.id)} icon={<CheckCircle className="w-3.5 h-3.5" />}>通过</Button>
                        <Button variant="outline" size="sm" onClick={() => setRejectingId(rejectingId === app.id ? null : app.id)} icon={<XCircle className="w-3.5 h-3.5" />} className="text-danger">拒绝</Button>
                      </div>
                    ) : (
                      <Badge variant={app.status === 'approved' ? 'success' : app.status === 'rejected' ? 'danger' : 'default'}>
                        {app.status === 'approved' ? '已通过' : app.status === 'rejected' ? '已拒绝' : app.status === 'withdrawn' ? '已撤回' : app.status}
                      </Badge>
                    )
                  } />
                  {rejectingId === app.id && (
                    <div className="mt-0 px-4 pb-4 animate-slide-up">
                      <div className="bg-danger/[0.04] border border-danger/10 rounded-lg p-3 flex items-center justify-between">
                        <span className="text-[12px] text-danger">确定拒绝此申请？</span>
                        <div className="flex gap-2">
                          <Button variant="ghost" size="sm" onClick={() => setRejectingId(null)}>取消</Button>
                          <Button variant="danger" size="sm" loading={rejectMut.isPending} onClick={() => rejectMut.mutate(app.id)}>确认拒绝</Button>
                        </div>
                      </div>
                    </div>
                  )}
                  {app.status === 'approved' && (
                    <div className="px-4 pb-4 text-[12px] text-success text-center animate-fade-in"><CheckCircle className="w-3 h-3 inline mr-0.5" />已向 {app.applicant_nickname} 发送通知 · 可以联系对方了</div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="py-12 text-center"><Inbox className="w-7 h-7 text-text-muted mb-3 mx-auto" /><p className="text-[13px] text-text-muted">暂无申请</p></div>
          )}
        </div>
      )}

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>确认删除</DialogTitle><DialogDescription>确定要删除此招募吗？<strong className="text-danger">此操作不可撤销</strong>，所有相关申请也将被删除。</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteOpen(false)}>取消</Button>
            <Button variant="danger" loading={deleteMut.isPending} onClick={() => { deleteMut.mutate(); setDeleteOpen(false) }}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

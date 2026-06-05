import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Flame, Pencil, Plus, ShieldCheck, Trash2 } from 'lucide-react'
import { projectApi } from '@/api/project'
import { Badge, Button, Card, Input, Skeleton, Textarea } from '@/components/ui'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'
import type { ProjectRisk, ProjectRiskCategory, ProjectRiskLevel, ProjectRiskReq, ProjectRiskStatus } from '@/types/api'

const categoryLabels: Record<ProjectRiskCategory, string> = { tech: '技术', schedule: '进度', art: '美术', team: '团队', scope: '范围', market: '市场', other: '其他' }
const levelLabels: Record<ProjectRiskLevel, string> = { low: '低', medium: '中', high: '高', critical: '严重' }
const statusLabels: Record<ProjectRiskStatus, string> = { open: '待处理', mitigating: '缓解中', resolved: '已解决' }
const levelVariant: Record<ProjectRiskLevel, 'default' | 'warning' | 'danger' | 'coral'> = { low: 'default', medium: 'warning', high: 'danger', critical: 'coral' }
const emptyRisk: ProjectRiskReq = { title: '', description: '', mitigation: '', category: 'tech', level: 'medium', status: 'open', due_date: '' }

function toFormRisk(risk: ProjectRisk): ProjectRiskReq & { id: number } {
  return { id: risk.id, title: risk.title, description: risk.description ?? '', mitigation: risk.mitigation ?? '', category: risk.category, level: risk.level, status: risk.status, due_date: risk.due_date ? risk.due_date.slice(0, 10) : '' }
}

export function ProjectRiskRadarTab({ projectId, isOwner, currentUserId }: { projectId: number; isOwner: boolean; currentUserId?: number }) {
  const queryClient = useQueryClient()
  const queryKey = ['project-risks', projectId]
  const [editingRisk, setEditingRisk] = useState<(ProjectRiskReq & { id?: number }) | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ProjectRisk | null>(null)

  const risksQuery = useQuery({ queryKey, queryFn: () => projectApi.listRisks(projectId) })
  const risks = risksQuery.data ?? []
  const summary = useMemo(() => ({ open: risks.filter(r => r.status !== 'resolved').length, critical: risks.filter(r => r.level === 'critical' && r.status !== 'resolved').length, resolved: risks.filter(r => r.status === 'resolved').length }), [risks])

  const saveMutation = useMutation({
    mutationFn: (risk: ProjectRiskReq & { id?: number }) => {
      const payload: ProjectRiskReq = { title: risk.title?.trim(), description: risk.description?.trim(), mitigation: risk.mitigation?.trim(), category: risk.category, level: risk.level, status: risk.status, due_date: risk.due_date || '' }
      return risk.id ? projectApi.updateRisk(projectId, risk.id, payload) : projectApi.createRisk(projectId, payload)
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey }); setEditingRisk(null); toast.success('风险已保存') },
    onError: (e: any) => toast.error(e?.message || '风险保存失败'),
  })
  const deleteMutation = useMutation({
    mutationFn: (riskId: number) => projectApi.deleteRisk(projectId, riskId),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey }); setDeleteTarget(null); toast.success('风险已删除') },
    onError: () => toast.error('删除失败'),
  })

  const saveRisk = () => {
    if (!editingRisk?.title?.trim()) { toast.error('请输入风险标题'); return }
    saveMutation.mutate(editingRisk)
  }

  return (
    <div className="py-6 space-y-4">
      <div className="flex items-center justify-between">
        <div><h3 className="text-h3 text-text-primary">项目风险雷达</h3><p className="text-[13px] text-text-muted mt-1">追踪技术、进度、美术、团队和市场风险，提前制定缓解方案</p></div>
        <Button size="sm" onClick={() => setEditingRisk({ ...emptyRisk })}><Plus className="w-4 h-4" />新增风险</Button>
      </div>

      <div className="grid grid-cols-3 gap-3">
        <Card hover={false} padding="md"><p className="text-[11px] text-text-muted">未解决</p><p className="text-[24px] font-bold text-text-primary font-mono">{summary.open}</p></Card>
        <Card hover={false} padding="md"><p className="text-[11px] text-text-muted">严重风险</p><p className="text-[24px] font-bold text-danger font-mono">{summary.critical}</p></Card>
        <Card hover={false} padding="md"><p className="text-[11px] text-text-muted">已解决</p><p className="text-[24px] font-bold text-success font-mono">{summary.resolved}</p></Card>
      </div>

      {risksQuery.isLoading ? <div className="space-y-3">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-28 rounded-xl" />)}</div>
      : risksQuery.isError ? <Card hover={false}><p className="text-[13px] text-danger text-center py-8">风险加载失败，请确认你是项目成员</p></Card>
      : risks.length === 0 ? <Card hover={false} className="py-14 text-center"><ShieldCheck className="w-7 h-7 text-text-muted mx-auto mb-3" /><p className="text-[14px] font-semibold text-text-secondary mb-1">暂无风险</p><p className="text-[12px] text-text-muted">记录风险能帮助团队提前避坑</p></Card>
      : <div className="space-y-3">{risks.map((risk) => {
        const canManage = isOwner || risk.creator_id === currentUserId
        return <Card key={risk.id} hover={false} padding="lg" className={risk.status === 'resolved' ? 'opacity-75' : ''}>
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2 mb-2"><Badge variant={levelVariant[risk.level]} size="sm">{levelLabels[risk.level]}</Badge><Badge variant={risk.status === 'resolved' ? 'success' : risk.status === 'mitigating' ? 'warning' : 'default'} size="sm">{statusLabels[risk.status]}</Badge><Badge variant="info" size="sm">{categoryLabels[risk.category]}</Badge>{risk.level === 'critical' && <Flame className="w-4 h-4 text-danger" />}</div>
              <h4 className="text-[15px] font-semibold text-text-primary">{risk.title}</h4>
              {risk.description && <p className="text-[12px] text-text-secondary mt-1 whitespace-pre-wrap">{risk.description}</p>}
              {risk.mitigation && <p className="text-[12px] text-amber mt-2">缓解：{risk.mitigation}</p>}
              {risk.due_date && <p className="text-[11px] text-text-muted mt-2">目标处理：{risk.due_date.slice(0, 10)}</p>}
            </div>
            {canManage && <div className="flex gap-1"><Button variant="ghost" size="sm" onClick={() => setEditingRisk(toFormRisk(risk))}><Pencil className="w-3.5 h-3.5" /></Button><Button variant="ghost" size="sm" className="text-danger" onClick={() => setDeleteTarget(risk)}><Trash2 className="w-3.5 h-3.5" /></Button></div>}
          </div>
        </Card>
      })}</div>}

      <Dialog open={editingRisk !== null} onOpenChange={(open) => { if (!open) setEditingRisk(null) }}><DialogContent><DialogHeader><DialogTitle>{editingRisk?.id ? '编辑风险' : '新增风险'}</DialogTitle><DialogDescription>记录风险、等级和缓解方案，方便团队生产复盘。</DialogDescription></DialogHeader>{editingRisk && <div className="space-y-4"><Input label="风险标题" value={editingRisk.title ?? ''} onChange={(e) => setEditingRisk({ ...editingRisk, title: e.target.value })} /><Textarea label="风险描述" rows={3} value={editingRisk.description ?? ''} onChange={(e) => setEditingRisk({ ...editingRisk, description: e.target.value })} /><Textarea label="缓解方案" rows={3} value={editingRisk.mitigation ?? ''} onChange={(e) => setEditingRisk({ ...editingRisk, mitigation: e.target.value })} /><Input label="目标处理日期" type="date" value={editingRisk.due_date ?? ''} onChange={(e) => setEditingRisk({ ...editingRisk, due_date: e.target.value })} /><div className="grid grid-cols-3 gap-3"><label className="text-[12px] text-text-secondary">类型<select value={editingRisk.category ?? 'tech'} onChange={(e) => setEditingRisk({ ...editingRisk, category: e.target.value as ProjectRiskCategory })} className="mt-1.5 w-full h-10 px-3 bg-surface-void border border-white/[0.06] rounded-lg text-[14px] text-text-primary outline-none"><option value="tech">技术</option><option value="schedule">进度</option><option value="art">美术</option><option value="team">团队</option><option value="scope">范围</option><option value="market">市场</option><option value="other">其他</option></select></label><label className="text-[12px] text-text-secondary">等级<select value={editingRisk.level ?? 'medium'} onChange={(e) => setEditingRisk({ ...editingRisk, level: e.target.value as ProjectRiskLevel })} className="mt-1.5 w-full h-10 px-3 bg-surface-void border border-white/[0.06] rounded-lg text-[14px] text-text-primary outline-none"><option value="low">低</option><option value="medium">中</option><option value="high">高</option><option value="critical">严重</option></select></label><label className="text-[12px] text-text-secondary">状态<select value={editingRisk.status ?? 'open'} onChange={(e) => setEditingRisk({ ...editingRisk, status: e.target.value as ProjectRiskStatus })} className="mt-1.5 w-full h-10 px-3 bg-surface-void border border-white/[0.06] rounded-lg text-[14px] text-text-primary outline-none"><option value="open">待处理</option><option value="mitigating">缓解中</option><option value="resolved">已解决</option></select></label></div></div>}<DialogFooter><Button variant="ghost" onClick={() => setEditingRisk(null)}>取消</Button><Button loading={saveMutation.isPending} onClick={saveRisk}>保存</Button></DialogFooter></DialogContent></Dialog>
      <Dialog open={deleteTarget !== null} onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}><DialogContent><DialogHeader><DialogTitle>删除风险</DialogTitle><DialogDescription>确定要删除风险“{deleteTarget?.title}”吗？</DialogDescription></DialogHeader><DialogFooter><Button variant="ghost" onClick={() => setDeleteTarget(null)}>取消</Button><Button variant="danger" loading={deleteMutation.isPending} onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}>删除</Button></DialogFooter></DialogContent></Dialog>
    </div>
  )
}

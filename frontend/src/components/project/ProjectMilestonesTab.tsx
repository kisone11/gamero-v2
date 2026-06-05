import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarDays, Edit, Plus, Trash2 } from 'lucide-react'
import { projectApi } from '@/api/project'
import { Badge, Button, Card, Input, Skeleton, Textarea } from '@/components/ui'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { toast } from '@/stores/toastStore'
import type { ProjectMilestone, ProjectMilestoneReq, ProjectMilestoneStatus } from '@/types/api'

const statusLabels: Record<ProjectMilestoneStatus, string> = {
  planned: '计划中',
  active: '进行中',
  done: '已完成',
}

const statusVariants: Record<ProjectMilestoneStatus, 'default' | 'warning' | 'success'> = {
  planned: 'default',
  active: 'warning',
  done: 'success',
}

const emptyMilestone: ProjectMilestoneReq = {
  title: '',
  description: '',
  status: 'planned',
  due_date: '',
}

function toFormMilestone(milestone: ProjectMilestone): ProjectMilestoneReq & { id: number } {
  return {
    id: milestone.id,
    title: milestone.title,
    description: milestone.description ?? '',
    status: milestone.status,
    due_date: milestone.due_date ? milestone.due_date.slice(0, 10) : '',
  }
}

function isOverdue(milestone: ProjectMilestone) {
  if (!milestone.due_date || milestone.status === 'done') return false
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return new Date(milestone.due_date) < today
}

function getErrorMessage(error: unknown) {
  if (!error) return '请稍后重试'
  if (typeof error === 'string') {
    return error.includes('404') || error.includes('not found')
      ? '后端服务未加载里程碑路由，请重启后端服务'
      : error
  }
  if (typeof error === 'object') {
    const data = error as {
      code?: number
      message?: string
      msg?: string
      response?: { status?: number; data?: { message?: string; msg?: string } | string }
    }
    if (data.response?.status === 404 || data.code === 404) {
      return '后端服务未加载里程碑路由，请重启后端服务'
    }
    const responseData = data.response?.data
    if (typeof responseData === 'string') {
      return responseData.includes('404') || responseData.includes('not found')
        ? '后端服务未加载里程碑路由，请重启后端服务'
        : responseData
    }
    return responseData?.message || responseData?.msg || data.message || data.msg || '请稍后重试'
  }
  return '请稍后重试'
}

export function ProjectMilestonesTab({ projectId, isOwner }: { projectId: number; isOwner: boolean }) {
  const queryClient = useQueryClient()
  const queryKey = ['project-milestones', projectId]
  const [editingMilestone, setEditingMilestone] = useState<(ProjectMilestoneReq & { id?: number }) | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ProjectMilestone | null>(null)

  const milestonesQuery = useQuery({ queryKey, queryFn: () => projectApi.listMilestones(projectId) })
  const milestones = milestonesQuery.data ?? []
  const milestoneError = getErrorMessage(milestonesQuery.error)

  const saveMutation = useMutation({
    mutationFn: (milestone: ProjectMilestoneReq & { id?: number }) => {
      const payload: ProjectMilestoneReq = {
        title: milestone.title?.trim(),
        description: milestone.description?.trim(),
        status: milestone.status,
        due_date: milestone.due_date || '',
      }
      return milestone.id
        ? projectApi.updateMilestone(projectId, milestone.id, payload)
        : projectApi.createMilestone(projectId, payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey })
      setEditingMilestone(null)
      toast.success('里程碑已保存')
    },
    onError: (e: any) => toast.error(e?.message || '里程碑保存失败'),
  })

  const statusMutation = useMutation({
    mutationFn: ({ milestoneId, status }: { milestoneId: number; status: ProjectMilestoneStatus }) =>
      projectApi.updateMilestone(projectId, milestoneId, { status }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey }),
    onError: () => toast.error('状态更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (milestoneId: number) => projectApi.deleteMilestone(projectId, milestoneId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey })
      setDeleteTarget(null)
      toast.success('里程碑已删除')
    },
    onError: () => toast.error('删除失败'),
  })

  const saveMilestone = () => {
    if (!editingMilestone?.title?.trim()) {
      toast.error('请输入里程碑标题')
      return
    }
    saveMutation.mutate(editingMilestone)
  }

  return (
    <div className="py-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-h3 text-text-primary">项目里程碑</h3>
          <p className="text-[13px] text-text-muted mt-1">规划版本目标、制作节点和上线排期</p>
        </div>
        {isOwner && <Button size="sm" onClick={() => setEditingMilestone({ ...emptyMilestone })}><Plus className="w-4 h-4" />新建里程碑</Button>}
      </div>

      {milestonesQuery.isLoading ? (
        <div className="space-y-3">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-28 rounded-xl" />)}</div>
      ) : milestonesQuery.isError ? (
        <Card hover={false}><p className="text-[13px] text-danger text-center py-8">里程碑加载失败：{milestoneError}</p></Card>
      ) : milestones.length === 0 ? (
        <Card hover={false} className="py-14 text-center">
          <CalendarDays className="w-7 h-7 text-text-muted mx-auto mb-3" />
          <p className="text-[14px] font-semibold text-text-secondary mb-1">暂无里程碑</p>
          <p className="text-[12px] text-text-muted">项目负责人可以添加制作计划和版本目标</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {milestones.map((milestone) => {
            const overdue = isOverdue(milestone)
            return (
              <Card key={milestone.id} hover={false} padding="lg" className={cn(milestone.status === 'done' && 'opacity-75', overdue && 'border-danger/20 bg-danger/[0.02]')}>
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2 mb-2">
                      <Badge variant={statusVariants[milestone.status]} size="sm">{statusLabels[milestone.status]}</Badge>
                      {overdue && <Badge variant="danger" size="sm">已逾期</Badge>}
                      {milestone.due_date && <span className="text-[11px] text-text-muted">目标 {milestone.due_date.slice(0, 10)}</span>}
                    </div>
                    <h4 className="text-[15px] font-semibold text-text-primary line-clamp-2">{milestone.title}</h4>
                    {milestone.description && <p className="text-[13px] text-text-secondary mt-1 whitespace-pre-wrap line-clamp-3">{milestone.description}</p>}
                    {milestone.completed_at && <p className="text-[11px] text-success mt-2">完成于 {milestone.completed_at.slice(0, 10)}</p>}
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <select
                      value={milestone.status}
                      onChange={(e) => statusMutation.mutate({ milestoneId: milestone.id, status: e.target.value as ProjectMilestoneStatus })}
                      className="h-8 px-2 rounded-lg bg-surface-void border border-white/[0.06] text-[12px] text-text-secondary outline-none"
                    >
                      <option value="planned">计划中</option>
                      <option value="active">进行中</option>
                      <option value="done">已完成</option>
                    </select>
                    {isOwner && <Button variant="ghost" size="sm" onClick={() => setEditingMilestone(toFormMilestone(milestone))}><Edit className="w-3.5 h-3.5" /></Button>}
                    {isOwner && <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(milestone)} className="text-danger"><Trash2 className="w-3.5 h-3.5" /></Button>}
                  </div>
                </div>
              </Card>
            )
          })}
        </div>
      )}

      <Dialog open={editingMilestone !== null} onOpenChange={(open) => { if (!open) setEditingMilestone(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>{editingMilestone?.id ? '编辑里程碑' : '新建里程碑'}</DialogTitle><DialogDescription>维护项目关键节点和目标日期。</DialogDescription></DialogHeader>
          {editingMilestone && <div className="space-y-4">
            <Input label="里程碑标题" value={editingMilestone.title ?? ''} onChange={(e) => setEditingMilestone({ ...editingMilestone, title: e.target.value })} />
            <Textarea label="说明" rows={3} value={editingMilestone.description ?? ''} onChange={(e) => setEditingMilestone({ ...editingMilestone, description: e.target.value })} />
            <Input label="目标日期" type="date" value={editingMilestone.due_date ?? ''} onChange={(e) => setEditingMilestone({ ...editingMilestone, due_date: e.target.value })} />
            <label className="text-[12px] text-text-secondary">状态<select value={editingMilestone.status ?? 'planned'} onChange={(e) => setEditingMilestone({ ...editingMilestone, status: e.target.value as ProjectMilestoneStatus })} className="mt-1.5 w-full h-10 px-3 bg-surface-void border border-white/[0.06] rounded-lg text-[14px] text-text-primary outline-none"><option value="planned">计划中</option><option value="active">进行中</option><option value="done">已完成</option></select></label>
          </div>}
          <DialogFooter><Button variant="ghost" onClick={() => setEditingMilestone(null)}>取消</Button><Button loading={saveMutation.isPending} onClick={saveMilestone}>保存</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteTarget !== null} onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}>
        <DialogContent><DialogHeader><DialogTitle>删除里程碑</DialogTitle><DialogDescription>确定要删除里程碑“{deleteTarget?.title}”吗？</DialogDescription></DialogHeader><DialogFooter><Button variant="ghost" onClick={() => setDeleteTarget(null)}>取消</Button><Button variant="danger" loading={deleteMutation.isPending} onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}>删除</Button></DialogFooter></DialogContent>
      </Dialog>
    </div>
  )
}

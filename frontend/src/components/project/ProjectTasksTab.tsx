import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle, Clock, Edit, Plus, Trash2 } from 'lucide-react'
import { projectApi } from '@/api/project'
import { Button, Badge, Card, Input, Skeleton, Textarea } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { toast } from '@/stores/toastStore'
import type { ProjectTask, ProjectTaskPriority, ProjectTaskReq, ProjectTaskStatus } from '@/types/api'

const COLUMNS: { status: ProjectTaskStatus; label: string; icon: React.ReactNode }[] = [
  { status: 'todo', label: '待处理', icon: <Clock className="w-4 h-4" /> },
  { status: 'doing', label: '进行中', icon: <Clock className="w-4 h-4 text-amber" /> },
  { status: 'done', label: '已完成', icon: <CheckCircle className="w-4 h-4 text-success" /> },
]

const priorityLabels: Record<ProjectTaskPriority, string> = { low: '低', medium: '中', high: '高' }
const priorityVariant: Record<ProjectTaskPriority, 'default' | 'warning' | 'danger'> = { low: 'default', medium: 'warning', high: 'danger' }

const emptyTask: ProjectTaskReq = { title: '', description: '', status: 'todo', priority: 'medium', due_date: '' }

function toFormTask(task: ProjectTask): ProjectTaskReq & { id: number } {
  return {
    id: task.id,
    title: task.title,
    description: task.description ?? '',
    status: task.status,
    priority: task.priority,
    due_date: task.due_date ? task.due_date.slice(0, 10) : '',
  }
}

export function ProjectTasksTab({ projectId, isOwner }: { projectId: number; isOwner: boolean }) {
  const queryClient = useQueryClient()
  const queryKey = ['project-tasks', projectId]
  const [editingTask, setEditingTask] = useState<(ProjectTaskReq & { id?: number }) | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ProjectTask | null>(null)

  const tasksQuery = useQuery({ queryKey, queryFn: () => projectApi.listTasks(projectId) })
  const tasks = tasksQuery.data ?? []
  const grouped = useMemo(() => COLUMNS.map((column) => ({ ...column, tasks: tasks.filter((task) => task.status === column.status) })), [tasks])

  const saveMutation = useMutation({
    mutationFn: (task: ProjectTaskReq & { id?: number }) => {
      const payload: ProjectTaskReq = {
        title: task.title?.trim(),
        description: task.description?.trim(),
        status: task.status,
        priority: task.priority,
        due_date: task.due_date || '',
      }
      return task.id ? projectApi.updateTask(projectId, task.id, payload) : projectApi.createTask(projectId, payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey })
      setEditingTask(null)
      toast.success('任务已保存')
    },
    onError: (e: any) => toast.error(e?.message || '任务保存失败'),
  })

  const statusMutation = useMutation({
    mutationFn: ({ taskId, status }: { taskId: number; status: ProjectTaskStatus }) => projectApi.updateTask(projectId, taskId, { status }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey }),
    onError: () => toast.error('状态更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (taskId: number) => projectApi.deleteTask(projectId, taskId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey })
      setDeleteTarget(null)
      toast.success('任务已删除')
    },
    onError: () => toast.error('删除失败'),
  })

  const saveTask = () => {
    if (!editingTask?.title?.trim()) {
      toast.error('请输入任务标题')
      return
    }
    saveMutation.mutate(editingTask)
  }

  return (
    <div className="py-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-h3 text-text-primary">项目任务看板</h3>
          <p className="text-[13px] text-text-muted mt-1">任务数据已接入后端，项目成员可协同查看与流转</p>
        </div>
        {isOwner && <Button size="sm" onClick={() => setEditingTask({ ...emptyTask })}><Plus className="w-4 h-4" />新建任务</Button>}
      </div>

      {tasksQuery.isLoading ? (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-40 rounded-xl" />)}</div>
      ) : tasksQuery.isError ? (
        <Card hover={false}><p className="text-[13px] text-danger text-center py-8">任务加载失败，请确认你是项目成员</p></Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          {grouped.map((column) => (
            <div key={column.status} className="space-y-3">
              <div className="flex items-center justify-between px-1">
                <span className="flex items-center gap-2 text-[13px] font-semibold text-text-secondary">{column.icon}{column.label}</span>
                <span className="text-[11px] text-text-muted font-mono">{column.tasks.length}</span>
              </div>
              <div className="space-y-3 min-h-32">
                {column.tasks.length === 0 ? (
                  <div className="h-24 rounded-xl border border-dashed border-white/[0.06] flex items-center justify-center text-[13px] text-text-muted">暂无任务</div>
                ) : column.tasks.map((task) => (
                  <Card key={task.id} padding="md" hover={false} className={cn(task.status === 'done' && 'opacity-75')}>
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <h4 className="text-[14px] font-semibold text-text-primary line-clamp-2">{task.title}</h4>
                        {task.description && <p className="text-[12px] text-text-muted mt-1 line-clamp-2">{task.description}</p>}
                      </div>
                      <Badge variant={priorityVariant[task.priority]} size="sm">{priorityLabels[task.priority]}</Badge>
                    </div>
                    <div className="flex flex-wrap gap-2 mt-3 text-[11px] text-text-muted">
                      {task.assignee_nickname && <span>@{task.assignee_nickname}</span>}
                      {task.due_date && <span>截止 {task.due_date.slice(0, 10)}</span>}
                    </div>
                    <div className="flex items-center gap-2 mt-4">
                      <select value={task.status} onChange={(e) => statusMutation.mutate({ taskId: task.id, status: e.target.value as ProjectTaskStatus })} className="h-8 px-2 rounded-lg bg-surface-void border border-white/[0.06] text-[12px] text-text-secondary outline-none">
                        {COLUMNS.map((item) => <option key={item.status} value={item.status}>{item.label}</option>)}
                      </select>
                      {isOwner && <Button variant="ghost" size="sm" onClick={() => setEditingTask(toFormTask(task))}><Edit className="w-3.5 h-3.5" /></Button>}
                      {isOwner && <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(task)} className="text-danger"><Trash2 className="w-3.5 h-3.5" /></Button>}
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      <Dialog open={editingTask !== null} onOpenChange={(open) => { if (!open) setEditingTask(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>{editingTask?.id ? '编辑任务' : '新建任务'}</DialogTitle><DialogDescription>维护项目协作任务和截止日期。</DialogDescription></DialogHeader>
          {editingTask && <div className="space-y-4">
            <Input label="任务标题" value={editingTask.title ?? ''} onChange={(e) => setEditingTask({ ...editingTask, title: e.target.value })} />
            <Textarea label="任务说明" rows={3} value={editingTask.description ?? ''} onChange={(e) => setEditingTask({ ...editingTask, description: e.target.value })} />
            <Input label="截止日期" type="date" value={editingTask.due_date ?? ''} onChange={(e) => setEditingTask({ ...editingTask, due_date: e.target.value })} />
            <div className="grid grid-cols-2 gap-3">
              <label className="text-[12px] text-text-secondary">状态<select value={editingTask.status ?? 'todo'} onChange={(e) => setEditingTask({ ...editingTask, status: e.target.value as ProjectTaskStatus })} className="mt-1.5 w-full h-10 px-3 bg-surface-void border border-white/[0.06] rounded-lg text-[14px] text-text-primary outline-none">{COLUMNS.map((item) => <option key={item.status} value={item.status}>{item.label}</option>)}</select></label>
              <label className="text-[12px] text-text-secondary">优先级<select value={editingTask.priority ?? 'medium'} onChange={(e) => setEditingTask({ ...editingTask, priority: e.target.value as ProjectTaskPriority })} className="mt-1.5 w-full h-10 px-3 bg-surface-void border border-white/[0.06] rounded-lg text-[14px] text-text-primary outline-none"><option value="low">低</option><option value="medium">中</option><option value="high">高</option></select></label>
            </div>
          </div>}
          <DialogFooter><Button variant="ghost" onClick={() => setEditingTask(null)}>取消</Button><Button loading={saveMutation.isPending} onClick={saveTask}>保存</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteTarget !== null} onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}>
        <DialogContent><DialogHeader><DialogTitle>删除任务</DialogTitle><DialogDescription>确定要删除任务“{deleteTarget?.title}”吗？</DialogDescription></DialogHeader><DialogFooter><Button variant="ghost" onClick={() => setDeleteTarget(null)}>取消</Button><Button variant="danger" loading={deleteMutation.isPending} onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}>删除</Button></DialogFooter></DialogContent>
      </Dialog>
    </div>
  )
}

import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle, ClipboardCheck, ExternalLink, Pencil, Plus, ShieldAlert, Trash2 } from 'lucide-react'
import { projectApi } from '@/api/project'
import { Badge, Button, Card, Input, Skeleton, Textarea } from '@/components/ui'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'
import type { ProjectQAItem, ProjectQACategory, ProjectQAItemReq, ProjectQAStatus } from '@/types/api'

const categoryLabels: Record<ProjectQACategory, string> = { gameplay: '玩法', art: '美术', audio: '音频', performance: '性能', bug: 'Bug', store: '商店页', compliance: '合规', other: '其他' }
const statusLabels: Record<ProjectQAStatus, string> = { pending: '待验收', passed: '通过', failed: '未通过', blocked: '阻塞' }
const statusVariant: Record<ProjectQAStatus, 'default' | 'success' | 'danger' | 'warning'> = { pending: 'default', passed: 'success', failed: 'danger', blocked: 'warning' }
const emptyItem: ProjectQAItemReq = { title: '', description: '', evidence_url: '', note: '', category: 'gameplay', status: 'pending', is_required: true }

function toFormItem(item: ProjectQAItem): ProjectQAItemReq & { id: number } {
  return { id: item.id, title: item.title, description: item.description ?? '', evidence_url: item.evidence_url ?? '', note: item.note ?? '', category: item.category, status: item.status, is_required: item.is_required }
}

export function ProjectQAGateTab({ projectId, isOwner, currentUserId }: { projectId: number; isOwner: boolean; currentUserId?: number }) {
  const queryClient = useQueryClient()
  const queryKey = ['project-qa-items', projectId]
  const [editingItem, setEditingItem] = useState<(ProjectQAItemReq & { id?: number }) | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ProjectQAItem | null>(null)

  const itemsQuery = useQuery({ queryKey, queryFn: () => projectApi.listQAItems(projectId) })
  const items = itemsQuery.data ?? []
  const summary = useMemo(() => {
    const required = items.filter(i => i.is_required)
    const blockers = required.filter(i => i.status === 'blocked' || i.status === 'failed').length
    const passed = required.filter(i => i.status === 'passed').length
    return { total: required.length, passed, blockers, ready: required.length > 0 && blockers === 0 && passed === required.length }
  }, [items])

  const saveMutation = useMutation({
    mutationFn: (item: ProjectQAItemReq & { id?: number }) => {
      const payload: ProjectQAItemReq = { title: item.title?.trim(), description: item.description?.trim(), evidence_url: item.evidence_url?.trim(), note: item.note?.trim(), category: item.category, status: item.status, is_required: !!item.is_required }
      return item.id ? projectApi.updateQAItem(projectId, item.id, payload) : projectApi.createQAItem(projectId, payload)
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey }); setEditingItem(null); toast.success('验收项已保存') },
    onError: (e: any) => toast.error(e?.message || '验收项保存失败'),
  })
  const deleteMutation = useMutation({ mutationFn: (itemId: number) => projectApi.deleteQAItem(projectId, itemId), onSuccess: () => { queryClient.invalidateQueries({ queryKey }); setDeleteTarget(null); toast.success('验收项已删除') }, onError: () => toast.error('删除失败') })

  const saveItem = () => {
    if (!editingItem?.title?.trim()) { toast.error('请输入验收项标题'); return }
    saveMutation.mutate(editingItem)
  }

  return <div className="py-6 space-y-4">
    <div className="flex items-center justify-between"><div><h3 className="text-h3 text-text-primary">上线验收清单</h3><p className="text-[13px] text-text-muted mt-1">发布前检查玩法、美术、音频、性能、Bug、商店页和合规项</p></div><Button size="sm" onClick={() => setEditingItem({ ...emptyItem })}><Plus className="w-4 h-4" />新增验收项</Button></div>
    <Card hover={false} padding="lg" className={summary.ready ? 'border-success/20 bg-success/[0.03]' : summary.blockers > 0 ? 'border-danger/20 bg-danger/[0.03]' : ''}><div className="flex items-center justify-between gap-4"><div className="flex items-center gap-3">{summary.ready ? <CheckCircle className="w-6 h-6 text-success" /> : summary.blockers > 0 ? <ShieldAlert className="w-6 h-6 text-danger" /> : <ClipboardCheck className="w-6 h-6 text-amber" />}<div><p className="text-[15px] font-semibold text-text-primary">{summary.ready ? '可以发布' : summary.blockers > 0 ? '存在阻塞项' : '等待验收'}</p><p className="text-[12px] text-text-muted">必选项通过 {summary.passed}/{summary.total}，阻塞 {summary.blockers}</p></div></div><Badge variant={summary.ready ? 'success' : summary.blockers > 0 ? 'danger' : 'warning'}>{summary.ready ? 'Ready' : summary.blockers > 0 ? 'Blocked' : 'Pending'}</Badge></div></Card>
    {itemsQuery.isLoading ? <div className="space-y-3">{Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-24 rounded-xl" />)}</div>
    : itemsQuery.isError ? <Card hover={false}><p className="text-[13px] text-danger text-center py-8">验收清单加载失败，请确认你是项目成员</p></Card>
    : items.length === 0 ? <Card hover={false} className="py-14 text-center"><ClipboardCheck className="w-7 h-7 text-text-muted mx-auto mb-3" /><p className="text-[14px] font-semibold text-text-secondary mb-1">暂无验收项</p><p className="text-[12px] text-text-muted">可以先添加发布前必须通过的 QA 项</p></Card>
    : <div className="space-y-3">{items.map((item) => { const canManage = isOwner || item.creator_id === currentUserId; return <Card key={item.id} hover={false} padding="lg"><div className="flex items-start justify-between gap-4"><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2 mb-2"><Badge variant={statusVariant[item.status]}>{statusLabels[item.status]}</Badge><Badge variant="info">{categoryLabels[item.category]}</Badge>{item.is_required && <Badge variant="amber">必选</Badge>}</div><h4 className="text-[15px] font-semibold text-text-primary">{item.title}</h4>{item.description && <p className="text-[12px] text-text-secondary mt-1">{item.description}</p>}{item.note && <p className="text-[12px] text-amber mt-2">备注：{item.note}</p>}{item.evidence_url && <button type="button" onClick={() => window.open(item.evidence_url, '_blank', 'noopener,noreferrer')} className="mt-2 inline-flex items-center gap-1.5 text-[12px] text-amber hover:underline">查看证据<ExternalLink className="w-3.5 h-3.5" /></button>}</div>{canManage && <div className="flex gap-1"><Button variant="ghost" size="sm" onClick={() => setEditingItem(toFormItem(item))}><Pencil className="w-3.5 h-3.5" /></Button><Button variant="ghost" size="sm" className="text-danger" onClick={() => setDeleteTarget(item)}><Trash2 className="w-3.5 h-3.5" /></Button></div>}</div></Card> })}</div>}
    <Dialog open={editingItem !== null} onOpenChange={(open) => { if (!open) setEditingItem(null) }}><DialogContent><DialogHeader><DialogTitle>{editingItem?.id ? '编辑验收项' : '新增验收项'}</DialogTitle><DialogDescription>必选项未通过或阻塞时，项目不可视为发布就绪。</DialogDescription></DialogHeader>{editingItem && <div className="space-y-4"><Input label="验收项标题" value={editingItem.title ?? ''} onChange={(e) => setEditingItem({ ...editingItem, title: e.target.value })} /><Textarea label="验收说明" rows={3} value={editingItem.description ?? ''} onChange={(e) => setEditingItem({ ...editingItem, description: e.target.value })} /><Input label="证据链接" placeholder="https://..." value={editingItem.evidence_url ?? ''} onChange={(e) => setEditingItem({ ...editingItem, evidence_url: e.target.value })} /><Textarea label="备注" rows={2} value={editingItem.note ?? ''} onChange={(e) => setEditingItem({ ...editingItem, note: e.target.value })} /><div className="grid grid-cols-2 gap-3"><label className="text-[12px] text-text-secondary">分类<select value={editingItem.category ?? 'gameplay'} onChange={(e) => setEditingItem({ ...editingItem, category: e.target.value as ProjectQACategory })} className="mt-1.5 w-full h-10 px-3 bg-surface-void border border-white/[0.06] rounded-lg text-[14px] text-text-primary outline-none"><option value="gameplay">玩法</option><option value="art">美术</option><option value="audio">音频</option><option value="performance">性能</option><option value="bug">Bug</option><option value="store">商店页</option><option value="compliance">合规</option><option value="other">其他</option></select></label><label className="text-[12px] text-text-secondary">状态<select value={editingItem.status ?? 'pending'} onChange={(e) => setEditingItem({ ...editingItem, status: e.target.value as ProjectQAStatus })} className="mt-1.5 w-full h-10 px-3 bg-surface-void border border-white/[0.06] rounded-lg text-[14px] text-text-primary outline-none"><option value="pending">待验收</option><option value="passed">通过</option><option value="failed">未通过</option><option value="blocked">阻塞</option></select></label></div><label className="flex items-center gap-2 text-[13px] text-text-secondary"><input type="checkbox" checked={!!editingItem.is_required} onChange={(e) => setEditingItem({ ...editingItem, is_required: e.target.checked })} />发布必选项</label></div>}<DialogFooter><Button variant="ghost" onClick={() => setEditingItem(null)}>取消</Button><Button loading={saveMutation.isPending} onClick={saveItem}>保存</Button></DialogFooter></DialogContent></Dialog>
    <Dialog open={deleteTarget !== null} onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}><DialogContent><DialogHeader><DialogTitle>删除验收项</DialogTitle><DialogDescription>确定要删除验收项“{deleteTarget?.title}”吗？</DialogDescription></DialogHeader><DialogFooter><Button variant="ghost" onClick={() => setDeleteTarget(null)}>取消</Button><Button variant="danger" loading={deleteMutation.isPending} onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}>删除</Button></DialogFooter></DialogContent></Dialog>
  </div>
}

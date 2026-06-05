import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Archive, Code2, ExternalLink, FileText, Image, Link2, Package, Pencil, Pin, Plus, Trash2 } from 'lucide-react'
import { projectApi } from '@/api/project'
import { Badge, Button, Card, Input, Skeleton, Textarea } from '@/components/ui'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'
import type { ProjectResource, ProjectResourceCategory, ProjectResourceReq } from '@/types/api'

const categoryMeta: Record<ProjectResourceCategory, { label: string; variant: 'default' | 'primary' | 'success' | 'warning' | 'info' | 'coral'; icon: React.ReactNode }> = {
  doc: { label: '文档', variant: 'primary', icon: <FileText className="w-3.5 h-3.5" /> },
  code: { label: '代码', variant: 'success', icon: <Code2 className="w-3.5 h-3.5" /> },
  build: { label: '试玩包', variant: 'warning', icon: <Package className="w-3.5 h-3.5" /> },
  asset: { label: '素材', variant: 'coral', icon: <Image className="w-3.5 h-3.5" /> },
  reference: { label: '参考', variant: 'info', icon: <Link2 className="w-3.5 h-3.5" /> },
  other: { label: '其他', variant: 'default', icon: <Archive className="w-3.5 h-3.5" /> },
}

const emptyResource: ProjectResourceReq = { title: '', url: '', category: 'doc', description: '', is_pinned: false }

function toFormResource(resource: ProjectResource): ProjectResourceReq & { id: number } {
  return {
    id: resource.id,
    title: resource.title,
    url: resource.url,
    category: resource.category,
    description: resource.description ?? '',
    is_pinned: resource.is_pinned,
  }
}

function getErrorMessage(error: unknown) {
  if (!error || typeof error !== 'object') return '请稍后重试'
  const data = error as { message?: string; msg?: string; response?: { data?: { message?: string; msg?: string } } }
  return data.response?.data?.message || data.response?.data?.msg || data.message || data.msg || '请稍后重试'
}

export function ProjectResourcesTab({ projectId, isOwner, currentUserId }: { projectId: number; isOwner: boolean; currentUserId?: number }) {
  const queryClient = useQueryClient()
  const queryKey = ['project-resources', projectId]
  const [editingResource, setEditingResource] = useState<(ProjectResourceReq & { id?: number }) | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ProjectResource | null>(null)

  const resourcesQuery = useQuery({ queryKey, queryFn: () => projectApi.listResources(projectId) })
  const resources = resourcesQuery.data ?? []

  const saveMutation = useMutation({
    mutationFn: (resource: ProjectResourceReq & { id?: number }) => {
      const payload: ProjectResourceReq = {
        title: resource.title?.trim(),
        url: resource.url?.trim(),
        category: resource.category,
        description: resource.description?.trim(),
        is_pinned: !!resource.is_pinned,
      }
      return resource.id ? projectApi.updateResource(projectId, resource.id, payload) : projectApi.createResource(projectId, payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey })
      setEditingResource(null)
      toast.success('资料已保存')
    },
    onError: (e: any) => toast.error(e?.message || '资料保存失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (resourceId: number) => projectApi.deleteResource(projectId, resourceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey })
      setDeleteTarget(null)
      toast.success('资料已删除')
    },
    onError: () => toast.error('删除失败'),
  })

  const saveResource = () => {
    if (!editingResource?.title?.trim()) {
      toast.error('请输入资料标题')
      return
    }
    if (!editingResource?.url?.trim()) {
      toast.error('请输入资料链接')
      return
    }
    saveMutation.mutate(editingResource)
  }

  return (
    <div className="py-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-h3 text-text-primary">项目资料库</h3>
          <p className="text-[13px] text-text-muted mt-1">集中管理设计文档、代码仓库、试玩包、素材和参考资料链接</p>
        </div>
        <Button size="sm" onClick={() => setEditingResource({ ...emptyResource })}><Plus className="w-4 h-4" />新增资料</Button>
      </div>

      {resourcesQuery.isLoading ? (
        <div className="space-y-3">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-24 rounded-xl" />)}</div>
      ) : resourcesQuery.isError ? (
        <Card hover={false}><p className="text-[13px] text-danger text-center py-8">资料加载失败：{getErrorMessage(resourcesQuery.error)}</p></Card>
      ) : resources.length === 0 ? (
        <Card hover={false} className="py-14 text-center">
          <Archive className="w-7 h-7 text-text-muted mx-auto mb-3" />
          <p className="text-[14px] font-semibold text-text-secondary mb-1">暂无项目资料</p>
          <p className="text-[12px] text-text-muted">成员可以添加文档、代码仓库、试玩包和素材链接</p>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {resources.map((resource) => {
            const meta = categoryMeta[resource.category] ?? categoryMeta.other
            const canManage = isOwner || resource.creator_id === currentUserId
            return (
              <Card key={resource.id} hover={false} padding="lg">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <Badge variant={meta.variant} size="sm" className="gap-1">{meta.icon}{meta.label}</Badge>
                      {resource.is_pinned && <Badge variant="amber" size="sm" className="gap-1"><Pin className="w-3 h-3" />置顶</Badge>}
                    </div>
                    <h4 className="text-[15px] font-semibold text-text-primary line-clamp-1">{resource.title}</h4>
                    {resource.description && <p className="text-[12px] text-text-muted mt-1 line-clamp-2">{resource.description}</p>}
                    <button type="button" onClick={() => window.open(resource.url, '_blank', 'noopener,noreferrer')} className="mt-3 inline-flex items-center gap-1.5 text-[12px] text-amber hover:underline">
                      打开链接<ExternalLink className="w-3.5 h-3.5" />
                    </button>
                  </div>
                  {canManage && (
                    <div className="flex items-center gap-1 shrink-0">
                      <Button variant="ghost" size="sm" onClick={() => setEditingResource(toFormResource(resource))}><Pencil className="w-3.5 h-3.5" /></Button>
                      <Button variant="ghost" size="sm" className="text-danger" onClick={() => setDeleteTarget(resource)}><Trash2 className="w-3.5 h-3.5" /></Button>
                    </div>
                  )}
                </div>
              </Card>
            )
          })}
        </div>
      )}

      <Dialog open={editingResource !== null} onOpenChange={(open) => { if (!open) setEditingResource(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>{editingResource?.id ? '编辑资料' : '新增资料'}</DialogTitle><DialogDescription>链接会对项目成员可见，请勿填写敏感账号或密钥。</DialogDescription></DialogHeader>
          {editingResource && <div className="space-y-4">
            <Input label="资料标题" value={editingResource.title ?? ''} onChange={(e) => setEditingResource({ ...editingResource, title: e.target.value })} />
            <Input label="链接地址" placeholder="https://..." value={editingResource.url ?? ''} onChange={(e) => setEditingResource({ ...editingResource, url: e.target.value })} />
            <label className="text-[12px] text-text-secondary">资料类型<select value={editingResource.category ?? 'doc'} onChange={(e) => setEditingResource({ ...editingResource, category: e.target.value as ProjectResourceCategory })} className="mt-1.5 w-full h-10 px-3 bg-surface-void border border-white/[0.06] rounded-lg text-[14px] text-text-primary outline-none"><option value="doc">文档</option><option value="code">代码</option><option value="build">试玩包</option><option value="asset">素材</option><option value="reference">参考</option><option value="other">其他</option></select></label>
            <Textarea label="说明" rows={3} value={editingResource.description ?? ''} onChange={(e) => setEditingResource({ ...editingResource, description: e.target.value })} />
            <label className="flex items-center gap-2 text-[13px] text-text-secondary"><input type="checkbox" checked={!!editingResource.is_pinned} onChange={(e) => setEditingResource({ ...editingResource, is_pinned: e.target.checked })} />置顶资料</label>
          </div>}
          <DialogFooter><Button variant="ghost" onClick={() => setEditingResource(null)}>取消</Button><Button loading={saveMutation.isPending} onClick={saveResource}>保存</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteTarget !== null} onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}>
        <DialogContent><DialogHeader><DialogTitle>删除资料</DialogTitle><DialogDescription>确定要删除资料“{deleteTarget?.title}”吗？</DialogDescription></DialogHeader><DialogFooter><Button variant="ghost" onClick={() => setDeleteTarget(null)}>取消</Button><Button variant="danger" loading={deleteMutation.isPending} onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}>删除</Button></DialogFooter></DialogContent>
      </Dialog>
    </div>
  )
}

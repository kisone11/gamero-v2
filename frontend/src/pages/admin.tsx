import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Pencil, Trash2, AlertTriangle, Hash,
} from 'lucide-react'
import { communityApi } from '@/api/community'
import { adminApi } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Tabs } from '@/components/ui/tabs'
import { Input } from '@/components/ui/input'
import { EmptyState } from '@/components/ui/empty-state'
import { toast } from '@/stores/toastStore'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { OverviewTab, ReportsTab, UsersTab, ContentTab, SensitiveWordsTab, TopicSkeleton } from '@/components/admin'

// ============================================================
// TopicsTab
// ============================================================

function TopicsTab() {
  const queryClient = useQueryClient()
  const [showCreate, setShowCreate] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editName, setEditName] = useState('')
  const [editDesc, setEditDesc] = useState('')
  const [createName, setCreateName] = useState('')
  const [createDesc, setCreateDesc] = useState('')
  const [deleteTopicId, setDeleteTopicId] = useState<number | null>(null)

  const { data: topics, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'topics'],
    queryFn: () => communityApi.listTopics(),
  })

  const createMut = useMutation({
    mutationFn: () => adminApi.createTopic({ name: createName, description: createDesc }),
    onSuccess: () => { setShowCreate(false); setCreateName(''); setCreateDesc(''); queryClient.invalidateQueries({ queryKey: ['admin', 'topics'] }); toast.success('话题已创建') },
    onError: () => toast.error('创建失败'),
  })

  const updateMut = useMutation({
    mutationFn: () => adminApi.updateTopic(editingId!, { name: editName, description: editDesc }),
    onSuccess: () => { setEditingId(null); queryClient.invalidateQueries({ queryKey: ['admin', 'topics'] }); toast.success('已更新') },
    onError: () => toast.error('更新失败'),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => adminApi.deleteTopic(id),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['admin', 'topics'] }); toast.success('已删除') },
    onError: () => toast.error('删除失败'),
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-[15px] font-semibold text-text-primary">话题管理</h3>
        <Button size="sm" onClick={() => setShowCreate(true)}><Plus className="w-3.5 h-3.5" /> 创建话题</Button>
      </div>

      {isLoading && <div className="space-y-3">{Array.from({ length: 3 }).map((_, i) => <TopicSkeleton key={i} />)}</div>}
      {isError && <EmptyState icon={<AlertTriangle className="w-7 h-7 text-text-muted" />} title="加载失败" action={{ label: '重试', onClick: () => refetch() }} />}
      {!isLoading && !isError && (!topics || topics.length === 0) && <EmptyState icon={<Hash className="w-7 h-7 text-text-muted" />} title="暂无话题" />}

      {topics && topics.length > 0 && (
        <div className="space-y-3">
          {topics.map((t: any) => (
            <Card key={t.id} padding="md">
              {editingId === t.id ? (
                <div className="flex items-center gap-3">
                  <Input value={editName} onChange={e => setEditName(e.target.value)} placeholder="话题名称" />
                  <Input value={editDesc} onChange={e => setEditDesc(e.target.value)} placeholder="描述" />
                  <Button size="sm" onClick={() => updateMut.mutate()} loading={updateMut.isPending}>保存</Button>
                  <Button size="sm" variant="ghost" onClick={() => setEditingId(null)}>取消</Button>
                </div>
              ) : (
                <div className="flex items-center gap-3">
                  <div className="flex-1">
                    <p className="text-[14px] font-medium text-text-primary">{t.name}</p>
                    <p className="text-[12px] text-text-muted">{t.description || '无描述'} · {t.post_count ?? 0} 帖</p>
                  </div>
                  <Button size="sm" variant="ghost" onClick={() => { setEditingId(t.id); setEditName(t.name); setEditDesc(t.description || '') }}><Pencil className="w-3.5 h-3.5" /></Button>
                  <Button size="sm" variant="ghost" onClick={() => setDeleteTopicId(t.id)}><Trash2 className="w-3.5 h-3.5 text-danger" /></Button>
                </div>
              )}
            </Card>
          ))}
        </div>
      )}

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader><DialogTitle>创建话题</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <Input label="名称" value={createName} onChange={e => setCreateName(e.target.value)} />
            <Input label="描述" value={createDesc} onChange={e => setCreateDesc(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setShowCreate(false)}>取消</Button>
            <Button loading={createMut.isPending} onClick={() => createMut.mutate()}>创建</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete topic confirmation dialog */}
      <Dialog open={deleteTopicId !== null} onOpenChange={(o) => { if (!o) setDeleteTopicId(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>删除话题</DialogTitle><DialogDescription>确定删除此话题？此操作不可撤销。</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteTopicId(null)}>取消</Button>
            <Button variant="danger" loading={deleteMut.isPending} onClick={() => {
              if (deleteTopicId !== null) {
                deleteMut.mutate(deleteTopicId)
                setDeleteTopicId(null)
              }
            }}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// ============================================================
// AdminPage
// ============================================================

export default function AdminPage() {
  const [tab, setTab] = useState('overview')

  return (
    <div className="max-w-6xl mx-auto px-6 py-8">
      <h1 className="text-[22px] font-bold text-text-primary mb-6">管理后台</h1>
      <Tabs
        tabs={[
          { value: 'overview', label: '概览' },
          { value: 'reports', label: '举报管理' },
          { value: 'users', label: '用户管理' },
          { value: 'content', label: '内容管理' },
          { value: 'topics', label: '话题管理' },
          { value: 'sensitive', label: '敏感词' },
        ]}
        value={tab}
        onChange={setTab}
      />
      <div className="mt-6">
        {tab === 'overview' && <OverviewTab />}
        {tab === 'reports' && <ReportsTab />}
        {tab === 'users' && <UsersTab />}
        {tab === 'content' && <ContentTab />}
        {tab === 'topics' && <TopicsTab />}
        {tab === 'sensitive' && <SensitiveWordsTab />}
      </div>
    </div>
  )
}

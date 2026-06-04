import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Pencil, Trash2, AlertTriangle, Hash, ShieldCheck,
} from 'lucide-react'
import { communityApi } from '@/api/community'
import { adminApi } from '@/api/admin'
import { announcementApi, type AnnouncementReq } from '@/api/announcement'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Tabs } from '@/components/ui/tabs'
import { Pagination } from '@/components/ui/pagination'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { EmptyState } from '@/components/ui/empty-state'
import { toast } from '@/stores/toastStore'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { OverviewTab, ReportsTab, UsersTab, ContentTab, SensitiveWordsTab, TopicSkeleton } from '@/components/admin'
import type { Announcement, AuditLog } from '@/types/api'

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

function AuditLogsTab() {
  const [page, setPage] = useState(1)
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'audit-logs', page],
    queryFn: () => adminApi.getAuditLogs({ page, page_size: 20 }),
  })
  const logs = data?.list ?? []

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-[15px] font-semibold text-text-primary">审计日志</h3>
          <p className="text-[12px] text-text-muted mt-1">记录封禁、删帖、处理举报等后台敏感操作。</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => refetch()}>刷新</Button>
      </div>

      {isLoading && <div className="space-y-3">{Array.from({ length: 5 }).map((_, i) => <TopicSkeleton key={i} />)}</div>}
      {isError && <EmptyState icon={<AlertTriangle className="w-7 h-7 text-text-muted" />} title="加载失败" action={{ label: '重试', onClick: () => refetch() }} />}
      {!isLoading && !isError && logs.length === 0 && <EmptyState icon={<ShieldCheck className="w-7 h-7 text-text-muted" />} title="暂无审计日志" />}

      {logs.length > 0 && (
        <>
          <div className="space-y-3">
            {logs.map((log: AuditLog) => (
              <Card key={log.id} padding="md" hover={false}>
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="rounded-full bg-amber/10 text-amber px-2 py-0.5 text-[11px] font-semibold">{log.action}</span>
                      <span className="text-[11px] text-text-muted font-mono">管理员 #{log.admin_id}</span>
                    </div>
                    <p className="text-[13px] text-text-secondary">
                      目标：{log.target_type} #{log.target_id}
                    </p>
                    {log.note && <p className="text-[12px] text-text-muted mt-1 line-clamp-2">{log.note}</p>}
                  </div>
                  <span className="text-[11px] text-text-muted font-mono shrink-0">{new Date(log.created_at).toLocaleString('zh-CN')}</span>
                </div>
              </Card>
            ))}
          </div>
          <Pagination page={page} pages={data?.pages ?? 1} onChange={setPage} />
        </>
      )}
    </div>
  )
}

const emptyAnnouncement: AnnouncementReq = { title: '', content: '', level: 'info', is_active: true, is_pinned: false, expire_days: 0 }

function AnnouncementsTab() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [editing, setEditing] = useState<(AnnouncementReq & { id?: number }) | null>(null)
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const query = useQuery({
    queryKey: ['admin', 'announcements', page],
    queryFn: () => announcementApi.adminList(page, 20),
  })
  const items = query.data?.list ?? []

  const saveMut = useMutation({
    mutationFn: (payload: AnnouncementReq & { id?: number }) => payload.id
      ? announcementApi.adminUpdate(payload.id, payload)
      : announcementApi.adminCreate(payload),
    onSuccess: () => {
      setEditing(null)
      queryClient.invalidateQueries({ queryKey: ['admin', 'announcements'] })
      queryClient.invalidateQueries({ queryKey: ['public-announcements'] })
      toast.success('公告已保存')
    },
    onError: (e: any) => toast.error(e?.message || '保存失败'),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => announcementApi.adminDelete(id),
    onSuccess: () => {
      setDeleteId(null)
      queryClient.invalidateQueries({ queryKey: ['admin', 'announcements'] })
      queryClient.invalidateQueries({ queryKey: ['public-announcements'] })
      toast.success('公告已删除')
    },
    onError: (e: any) => toast.error(e?.message || '删除失败'),
  })

  const toggleActive = (item: Announcement) => {
    saveMut.mutate({
      id: item.id,
      title: item.title,
      content: item.content,
      level: item.level,
      is_active: !item.is_active,
      is_pinned: item.is_pinned,
      expire_days: 0,
    })
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-[15px] font-semibold text-text-primary">站内公告</h3>
          <p className="text-[12px] text-text-muted mt-1">发布运营公告、维护通知和活动提醒，会展示在动态页右侧。</p>
        </div>
        <Button size="sm" onClick={() => setEditing({ ...emptyAnnouncement })}><Plus className="w-3.5 h-3.5" />发布公告</Button>
      </div>

      {query.isLoading && <div className="space-y-3">{Array.from({ length: 4 }).map((_, i) => <TopicSkeleton key={i} />)}</div>}
      {query.isError && <EmptyState icon={<AlertTriangle className="w-7 h-7 text-text-muted" />} title="加载失败" action={{ label: '重试', onClick: () => query.refetch() }} />}
      {!query.isLoading && !query.isError && items.length === 0 && <EmptyState icon={<Hash className="w-7 h-7 text-text-muted" />} title="暂无公告" />}

      {items.length > 0 && (
        <>
          <div className="space-y-3">
            {items.map((item) => (
              <Card key={item.id} padding="md" hover={false}>
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      {item.is_pinned && <span className="rounded-full bg-amber/10 text-amber px-2 py-0.5 text-[11px] font-semibold">置顶</span>}
                      <span className="rounded-full bg-white/[0.06] text-text-secondary px-2 py-0.5 text-[11px] font-mono">{item.level}</span>
                      <span className={item.is_active ? 'text-[11px] text-success' : 'text-[11px] text-text-muted'}>{item.is_active ? '展示中' : '已下线'}</span>
                    </div>
                    <p className="text-[14px] font-semibold text-text-primary">{item.title}</p>
                    <p className="text-[12px] text-text-muted mt-1 line-clamp-2">{item.content}</p>
                    <p className="text-[11px] text-text-muted mt-2 font-mono">发布：{new Date(item.published_at).toLocaleString('zh-CN')}</p>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <Button variant="outline" size="sm" onClick={() => toggleActive(item)}>{item.is_active ? '下线' : '上线'}</Button>
                    <Button variant="ghost" size="sm" onClick={() => setEditing({ id: item.id, title: item.title, content: item.content, level: item.level, is_active: item.is_active, is_pinned: item.is_pinned, expire_days: 0 })}><Pencil className="w-3.5 h-3.5" /></Button>
                    <Button variant="ghost" size="sm" onClick={() => setDeleteId(item.id)}><Trash2 className="w-3.5 h-3.5 text-danger" /></Button>
                  </div>
                </div>
              </Card>
            ))}
          </div>
          <Pagination page={page} pages={query.data?.pages ?? 1} onChange={setPage} />
        </>
      )}

      <Dialog open={editing !== null} onOpenChange={(open) => { if (!open) setEditing(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>{editing?.id ? '编辑公告' : '发布公告'}</DialogTitle><DialogDescription>公告会展示在动态页右侧，适合维护通知、活动提示和平台规则更新。</DialogDescription></DialogHeader>
          {editing && (
            <div className="space-y-4">
              <Input label="标题" value={editing.title} onChange={(e) => setEditing({ ...editing, title: e.target.value })} />
              <Textarea label="内容" value={editing.content} onChange={(e) => setEditing({ ...editing, content: e.target.value })} rows={5} />
              <div className="grid grid-cols-2 gap-3">
                <select value={editing.level} onChange={(e) => setEditing({ ...editing, level: e.target.value as AnnouncementReq['level'] })} className="h-10 px-3 bg-surface-void rounded-lg text-[13px] text-text-primary border border-white/[0.06] outline-none">
                  <option value="info">普通</option>
                  <option value="success">成功</option>
                  <option value="warning">警告</option>
                  <option value="danger">重要</option>
                </select>
                <Input type="number" label="有效天数（0为长期）" value={editing.expire_days ?? 0} onChange={(e) => setEditing({ ...editing, expire_days: Number(e.target.value) })} />
              </div>
              <div className="flex items-center gap-4 text-[13px] text-text-secondary">
                <label className="flex items-center gap-2"><input type="checkbox" checked={editing.is_active !== false} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} />展示</label>
                <label className="flex items-center gap-2"><input type="checkbox" checked={!!editing.is_pinned} onChange={(e) => setEditing({ ...editing, is_pinned: e.target.checked })} />置顶</label>
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="ghost" onClick={() => setEditing(null)}>取消</Button>
            <Button loading={saveMut.isPending} onClick={() => editing && saveMut.mutate(editing)}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteId !== null} onOpenChange={(open) => { if (!open) setDeleteId(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>删除公告</DialogTitle><DialogDescription>确定删除这条公告？此操作不可撤销。</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteId(null)}>取消</Button>
            <Button variant="danger" loading={deleteMut.isPending} onClick={() => deleteId && deleteMut.mutate(deleteId)}>确认删除</Button>
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
          { value: 'announcements', label: '公告' },
          { value: 'audit', label: '审计日志' },
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
        {tab === 'announcements' && <AnnouncementsTab />}
        {tab === 'audit' && <AuditLogsTab />}
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCheck, Trash2, Bell, Zap, UserPlus, CheckCircle, XCircle, AtSign, Heart, MessageSquare, ThumbsUp, Send, Ban, ClipboardCheck, RefreshCw, X, Eraser } from 'lucide-react'
import { notificationApi } from '@/api/notification'
import { Button } from '@/components/ui/button'
import { Pagination } from '@/components/ui/pagination'
import { Tabs } from '@/components/ui/tabs'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { toast } from '@/stores/toastStore'
import { timeAgo } from '@/lib/time'
import type { Notification, NotificationType } from '@/types/api'

// ============================================================
// Constants
// ============================================================

const NOTIF_ICON_MAP: Record<string, React.ComponentType<{ className?: string }>> = {
  recruitment_applied: UserPlus,
  application_approved: CheckCircle,
  application_rejected: XCircle,
  mention: AtSign,
  new_follower: UserPlus,
  project_status_changed: RefreshCw,
  post_liked: Heart,
  post_commented: MessageSquare,
  log_liked: ThumbsUp,
  log_commented: MessageSquare,
  system: Bell,
  talent_invited: Send,
  invite_accepted: CheckCheck,
  invite_declined: X,
  project_banned: Ban,
  project_unbanned: CheckCircle,
  report_handled: ClipboardCheck,
}

const CATEGORY_TABS = [
  { value: 'all', label: '全部' },
  { value: 'social', label: '社交' },
  { value: 'project', label: '项目' },
  { value: 'system', label: '系统' },
] as const

type CategoryTab = (typeof CATEGORY_TABS)[number]['value']

// ============================================================
// Helpers
// ============================================================

function getNotificationUrl(type: string, metadata: any): string | null {
  const m = metadata || {}
  switch (type) {
    case 'post_liked': case 'post_commented': case 'mention':
      return m.post_id ? `/post/${m.post_id}` : null
    case 'log_liked': case 'log_commented':
      return m.log_id ? `/devlog/${m.log_id}` : null
    case 'new_follower':
      return m.follower_username ? `/u/${m.follower_username}` : m.follower_id ? `/u/${m.follower_id}` : null
    case 'project_status_changed': case 'project_banned': case 'project_unbanned':
      return m.project_slug ? `/p/${m.project_slug}` : m.project_id ? `/p/${m.project_id}` : null
    case 'recruitment_applied': case 'application_approved': case 'application_rejected':
      return m.recruitment_id ? `/recruit/${m.recruitment_id}` : null
    case 'talent_invited': case 'invite_accepted': case 'invite_declined':
      return m.invitation_id ? `/me/invitations` : null
    case 'report_handled':
      return '/me/reports'
    default:
      return null
  }
}

function NotificationSkeleton() {
  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
      <div className="flex items-start gap-4">
        <div className="w-8 h-8 rounded-lg bg-white/[0.04] animate-pulse shrink-0" />
        <div className="flex-1 space-y-2">
          <div className="h-4 w-3/4 bg-white/[0.04] rounded animate-pulse" />
          <div className="h-3 w-full bg-white/[0.04] rounded animate-pulse" />
          <div className="h-3 w-20 bg-white/[0.04] rounded animate-pulse" />
        </div>
      </div>
    </div>
  )
}

// ============================================================
// NotificationsPage
// ============================================================

export default function NotificationsPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [category, setCategory] = useState<CategoryTab>('all')
  const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null)
  const [clearReadOpen, setClearReadOpen] = useState(false)

  // ---- Queries ----

  const listQuery = useQuery({
    queryKey: ['notifications', category, page],
    queryFn: () => {
      if (category === 'all') return notificationApi.list(page, 20)
      return notificationApi.getByType(category, page, 20)
    },
  })

  const unreadCountQuery = useQuery({
    queryKey: ['notifications-unread-count'],
    queryFn: () => notificationApi.getUnreadCount(),
    refetchInterval: 30000,
  })

  const categoriesQuery = useQuery({
    queryKey: ['notifications-categories'],
    queryFn: () => notificationApi.getCategories(),
  })

  const notifications = listQuery.data?.list ?? []
  const pages = listQuery.data?.pages ?? 1
  const unreadCount = (unreadCountQuery.data as any)?.unread_count ?? 0

  // Build tab counts from categories
  const categoryCounts = (categoriesQuery.data ?? []).reduce(
    (acc, cat) => {
      acc[cat.type_group as CategoryTab] = cat.unread_count
      acc.all += cat.unread_count
      return acc
    },
    { all: 0, social: 0, project: 0, system: 0 } as Record<CategoryTab, number>,
  )

  // ---- Mutations ----

  const markReadMut = useMutation({
    mutationFn: (id: number) => notificationApi.markRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-unread-count'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-categories'] })
    },
  })

  const markAllReadMut = useMutation({
    mutationFn: () => notificationApi.markAllRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-unread-count'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-categories'] })
      toast.success('已全部标记为已读')
    },
    onError: (err: any) => toast.error(err?.message || '操作失败'),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => notificationApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-unread-count'] })
      toast.success('已删除')
    },
    onError: (err: any) => toast.error(err?.message || '删除失败'),
  })

  const deleteReadMut = useMutation({
    mutationFn: () => notificationApi.deleteRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-unread-count'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-categories'] })
      toast.success('已清除所有已读通知')
      setClearReadOpen(false)
    },
    onError: (err: any) => toast.error(err?.message || '操作失败'),
  })

  // ---- Handlers ----

  const handleClick = (notif: Notification) => {
    if (!notif.is_read) {
      markReadMut.mutate(notif.id)
    }
    const url = getNotificationUrl(notif.type, notif.metadata)
    if (url) navigate(url)
  }

  const handleCategoryChange = (value: string) => {
    setCategory(value as CategoryTab)
    setPage(1)
  }

  return (
    <div className="max-w-4xl mx-auto px-6 py-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <h1 className="text-[22px] font-bold text-text-primary">通知</h1>
          {unreadCount > 0 && (
            <span className="px-2 py-0.5 text-[11px] font-mono font-semibold bg-amber/10 text-amber rounded-full">
              {unreadCount} 条未读
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            loading={deleteReadMut.isPending}
            onClick={() => setClearReadOpen(true)}
          >
            <Eraser className="h-4 w-4" />
            清除已读
          </Button>
          {unreadCount > 0 && (
            <Button
              variant="outline"
              size="sm"
              loading={markAllReadMut.isPending}
              onClick={() => markAllReadMut.mutate()}
            >
              <CheckCheck className="h-4 w-4" />
              全部已读
            </Button>
          )}
        </div>
      </div>

      {/* Category Tabs */}
      <Tabs
        tabs={CATEGORY_TABS.map((tab) => ({
          value: tab.value,
          label: tab.label,
          count: categoryCounts[tab.value],
        }))}
        value={category}
        onChange={handleCategoryChange}
        className="mb-4"
      />

      {/* Loading */}
      {listQuery.isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <NotificationSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Error */}
      {listQuery.isError && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Zap className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
          <p className="text-[13px] text-text-muted mb-6">无法加载通知，请重试</p>
          <Button variant="secondary" size="sm" onClick={() => listQuery.refetch()}>重新加载</Button>
        </div>
      )}

      {/* Empty */}
      {!listQuery.isLoading && !listQuery.isError && notifications.length === 0 && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Bell className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无通知</p>
          <p className="text-[13px] text-text-muted">当有人与你互动时，通知将出现在这里</p>
        </div>
      )}

      {/* List */}
      {!listQuery.isLoading && !listQuery.isError && notifications.length > 0 && (
        <>
          <div className="space-y-2">
            {notifications.map((notif) => (
              <div
                key={notif.id}
                className={cn(
                  'bg-surface-card border border-white/[0.04] rounded-xl p-5',
                  'flex items-start gap-4 transition-all duration-200 cursor-pointer',
                  'hover:border-amber/12 hover:shadow-[0_4px_24px_rgba(0,0,0,0.25)] hover:-translate-y-[1px]',
                  !notif.is_read && 'border-l-2 border-amber',
                )}
                onClick={() => handleClick(notif)}
              >
                <div className="w-10 h-10 rounded-lg bg-amber/10 flex items-center justify-center shrink-0 ring-1 ring-amber/20">
                  {(() => {
                    const IconComp = NOTIF_ICON_MAP[notif.type]
                    if (!IconComp) return <Bell className="w-5 h-5 text-amber" />
                    return <IconComp className="w-5 h-5 text-amber" />
                  })()}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 mb-0.5">
                    <span
                      className={cn(
                        'text-[14px]',
                        !notif.is_read ? 'font-semibold text-text-primary' : 'text-text-secondary',
                      )}
                    >
                      {notif.title}
                    </span>
                    {!notif.is_read && (
                      <span className="w-2 h-2 rounded-full bg-amber shrink-0" />
                    )}
                  </div>
                  {notif.sender_id != null && notif.sender_id > 0 && (
                    <p className="text-[11px] text-text-muted mt-0.5">
                      发送者: {notif.sender_id}
                    </p>
                  )}
                  <p className="text-[13px] text-text-secondary line-clamp-2">
                    {notif.content}
                  </p>
                  <span className="text-[11px] text-text-muted font-mono mt-1 inline-block">
                    {timeAgo(notif.created_at)}
                  </span>
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    setDeleteConfirmId(notif.id)
                  }}
                  className="shrink-0 p-1 text-text-muted hover:text-danger transition-colors"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />

          {/* Delete confirmation dialog */}
          <Dialog open={deleteConfirmId !== null} onOpenChange={(o) => { if (!o) setDeleteConfirmId(null) }}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>确认删除</DialogTitle>
              </DialogHeader>
              <p className="text-[14px] text-text-secondary">确定要删除这条通知吗？此操作不可撤销。</p>
              <DialogFooter>
                <Button variant="ghost" onClick={() => setDeleteConfirmId(null)}>取消</Button>
                <Button variant="danger" loading={deleteMut.isPending} onClick={() => {
                  if (deleteConfirmId !== null) {
                    deleteMut.mutate(deleteConfirmId)
                    setDeleteConfirmId(null)
                  }
                }}>删除</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          {/* Clear read confirmation dialog */}
          <Dialog open={clearReadOpen} onOpenChange={setClearReadOpen}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>确认清除</DialogTitle>
              </DialogHeader>
              <p className="text-[14px] text-text-secondary">确定要删除所有已读通知吗？此操作不可撤销。</p>
              <DialogFooter>
                <Button variant="ghost" onClick={() => setClearReadOpen(false)}>取消</Button>
                <Button variant="danger" loading={deleteReadMut.isPending} onClick={() => deleteReadMut.mutate()}>清除</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </>
      )}
    </div>
  )
}

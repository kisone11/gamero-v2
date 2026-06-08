import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Heart, Bookmark, Share2, Edit, Trash2, ArrowLeft, MessageSquare,
  Eye, Download,
} from 'lucide-react'
import { logApi } from '@/api/log'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { timeAgo } from '@/lib/time'
import { Markdown } from '@/components/ui/markdown'
import { toast } from '@/stores/toastStore'
import { ReportDialog } from '@/components/shared/report-dialog'
import { openLightbox } from '@/components/shared/ImageGallery'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import type { DevLogDetail, LogComment } from '@/types/api'
import type { DevLogType } from '@/types/enums'
import { type CommentItemData } from '@/components/shared/CommentItem'
import { CommentSection, type CommentSectionConfig } from '@/components/shared/CommentSection'
import { DevlogSidebar, DevlogSkeleton, ScrollTextIcon, Send, ExternalLinkIcon } from '@/components/devlog'

const logTypeConfig: Record<DevLogType, { label: string }> = {
  log: { label: '开发日志' },
  release: { label: '版本发布' },
}

// ============================================================
// Adapter: LogComment -> CommentItemData
// ============================================================

function toCommentData(c: LogComment, currentUserId?: number): CommentItemData {
  return {
    id: c.id,
    content: c.content,
    authorName: c.author?.nickname || c.user_nickname || c.user_username || '用户',
    authorUsername: c.author?.username || c.user_username || '',
    authorAvatar: c.author?.avatar_url,
    likeCount: c.like_count,
    isLiked: c.is_liked,
    isDeleted: c.is_deleted,
    createdAt: c.created_at,
    isAuthor: currentUserId != null && (c.user_id === currentUserId || c.author?.id === currentUserId),
    parentAuthorName: c.parent_nickname ?? undefined,
    replies: c.replies?.map(r => toCommentData(r, currentUserId)),
  }
}

// ============================================================
// DevLogDetailPage
// ============================================================

export default function DevLogDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const user = useAuthStore((s) => s.user)

  const logId = Number(id)

  const logQuery = useQuery({
    queryKey: ['devlog', logId],
    queryFn: () => logApi.get(logId),
    enabled: !!logId && !isNaN(logId),
  })

  const log = logQuery.data
  const author = log?.author

  const [deleteLogDialogOpen, setDeleteLogDialogOpen] = useState(false)
  const [publishDialogOpen, setPublishDialogOpen] = useState(false)

  // ── Mutations ──

  const likeMutation = useMutation({
    mutationFn: () =>
      log?.is_liked ? logApi.unlike(logId) : logApi.like(logId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devlog', logId] })
      toast.success(log?.is_liked ? '已取消点赞' : '已点赞')
    },
    onError: () => toast.error('操作失败，请重试'),
  })

  const collectMutation = useMutation({
    mutationFn: () =>
      log?.is_collected ? logApi.uncollect(logId) : logApi.collect(logId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devlog', logId] })
      toast.success(log?.is_collected ? '已取消收藏' : '已收藏')
    },
    onError: () => toast.error('操作失败，请重试'),
  })

  const publishMutation = useMutation({
    mutationFn: () => logApi.publish(logId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devlog', logId] })
      toast.success('日志已发布')
    },
    onError: () => toast.error('发布失败'),
  })

  const deleteLogMutation = useMutation({
    mutationFn: () => logApi.delete(logId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devlogs'] })
      queryClient.invalidateQueries({ queryKey: ['devlog', logId] })
      if (log?.project_id) {
        queryClient.invalidateQueries({ queryKey: ['project-logs', log.project_id] })
      }
      toast.success('日志已删除')
      if (log?.project_slug) {
        navigate(`/p/${log.project_slug}`)
      } else {
        navigate('/')
      }
    },
    onError: () => toast.error('删除失败'),
  })

  const handleShare = () => {
    if (navigator.share) {
      navigator.share({ title: log?.title, url: window.location.href }).catch(() => { })
    } else {
      navigator.clipboard.writeText(window.location.href).then(
        () => toast.success('链接已复制'),
        () => toast.error('复制失败'),
      )
    }
  }

  const handleEdit = () => {
    navigate(`/devlog/${logId}/edit`)
  }

  const handleDelete = () => {
    setDeleteLogDialogOpen(true)
  }

  const handlePublish = () => {
    setPublishDialogOpen(true)
  }

  // ── Loading ──
  if (logQuery.isLoading) {
    return <DevlogSkeleton />
  }

  // ── Error ──
  if (logQuery.isError) {
    return (
      <div className="max-w-[720px] mx-auto px-6 py-8 min-h-[60vh] flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <MessageSquare className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">
            加载失败
          </p>
          <p className="text-[13px] text-text-muted mb-6">
            无法加载开发日志信息，请检查网络后重试
          </p>
          <Button variant="secondary" size="sm" onClick={() => logQuery.refetch()}>
            重新加载
          </Button>
        </div>
      </div>
    )
  }

  // ── Not found ──
  if (!log) {
    return (
      <div className="max-w-[720px] mx-auto px-6 py-8 min-h-[60vh] flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <ScrollTextIcon className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">
            日志不存在
          </p>
          <p className="text-[13px] text-text-muted mb-6">
            该开发日志可能已被删除或链接地址有误
          </p>
          <Button variant="secondary" size="sm" onClick={() => navigate('/')}>
            返回首页
          </Button>
        </div>
      </div>
    )
  }

  // ── Render ──
  const isOwner = user && author && author.id === user.id
  const logType = logTypeConfig[log.log_type]
  const isDraft = log.status === 'draft'
  const PAGE_SIZE = 20
  const commentConfig: CommentSectionConfig = {
    queryKey: ['log-comments', logId],
    fetchComments: (page, sort) => logApi.listComments(logId, page, PAGE_SIZE, sort),
    createComment: (content, replyToId) => logApi.createComment(logId, { content, reply_to_id: replyToId ?? 0 }),
    likeComment: (id) => logApi.likeComment(logId, id),
    unlikeComment: (id) => logApi.unlikeComment(logId, id),
    deleteComment: (id) => logApi.deleteComment(logId, id),
    normalizeComment: (c: any, uid?: number) => toCommentData(c as LogComment, uid),
  }

  return (
    <>
      <div className="max-w-[1060px] mx-auto px-6 py-8">
        <div className="flex gap-8">
          {/* ── Main content ── */}
          <main className="flex-1 min-w-0 max-w-[720px]">
            {/* Log card */}
            <div className="bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden">
              {/* Header */}
              <div className="p-5 border-b border-white/[0.04]">
                {/* Back and actions */}
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <Link
                      to="/devlogs"
                      className="flex items-center gap-1.5 text-[13px] text-text-muted hover:text-amber transition-colors"
                    >
                      <ArrowLeft className="w-4 h-4" />
                      返回开发日志
                    </Link>
                    {log.project_slug && (
                      <Link
                        to={`/p/${log.project_slug}`}
                        className="flex items-center gap-1.5 text-[13px] text-text-muted hover:text-amber transition-colors"
                      >
                        返回项目
                      </Link>
                    )}
                  </div>

                  <div className="flex items-center gap-1">
                    <Button variant="ghost" size="sm" onClick={handleShare}>
                      <Share2 className="w-4 h-4" />
                      <span className="hidden sm:inline ml-1">分享</span>
                    </Button>
                    {isOwner && (
                      <>
                        <Button variant="ghost" size="sm" onClick={handleEdit}>
                          <Edit className="w-4 h-4" />
                        </Button>
                        {isDraft && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={handlePublish}
                            loading={publishMutation.isPending}
                          >
                            <Send className="w-4 h-4" />
                            <span className="hidden sm:inline ml-1">发布</span>
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={handleDelete}
                          className="text-danger hover:text-danger"
                          loading={deleteLogMutation.isPending}
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </>
                    )}
                    {user && !isOwner && (
                      <ReportDialog targetType="project_release" targetId={log.id} />
                    )}
                  </div>
                </div>

                {/* Badges row */}
                <div className="flex flex-wrap items-center gap-2 mb-3">
                  <span
                    className={cn(
                      'rounded-full px-3 py-1 text-[11px] font-medium',
                      log.log_type === 'release'
                        ? 'bg-amber/10 text-amber'
                        : 'bg-white/[0.06] text-text-secondary',
                    )}
                  >
                    {logType.label}
                  </span>
                  {log.version && (
                    <span className="rounded-full bg-black/30 border border-white/[0.04] text-text-secondary px-3 py-1 text-[11px] font-mono">
                      v{log.version}
                    </span>
                  )}
                  {isDraft && (
                    <span className="rounded-full bg-warning/10 text-warning px-3 py-1 text-[11px] font-medium">
                      草稿
                    </span>
                  )}
                  <span className="text-[12px] text-text-muted font-mono ml-auto">
                    {timeAgo(log.created_at)}
                  </span>
                </div>

                {/* Title */}
                <h1 className="text-[24px] font-bold text-text-primary leading-tight">
                  {log.title}
                </h1>
              </div>

              {/* Content */}
              <div className="p-5 border-b border-white/[0.04]">
                <Markdown content={log.content || '暂无内容'} />
              </div>

              {/* Images gallery */}
              {(log.image_urls?.length ?? 0) > 0 && (
                <div className="p-5 border-b border-white/[0.04]">

                  <div className="flex flex-wrap gap-2">
                    {log.image_urls!.map((url, i) => (
                      <button key={i} onClick={() => openLightbox(log.image_urls!, i)}
                        className="w-[calc(33.33%-0.5rem)] aspect-[4/3] bg-black/20 rounded-lg overflow-hidden cursor-pointer hover:opacity-90 transition-opacity border border-white/[0.04]">
                        <img src={url} alt={`图片 ${i + 1}`} className="w-full h-full object-cover" loading="lazy" />
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Download link */}
              {log.download_url && (
                <div className="px-5 py-4 border-b border-white/[0.04]">
                  <a
                    href={log.download_url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <Button variant="secondary" size="sm">
                      <Download className="w-4 h-4" />
                      下载
                      <ExternalLinkIcon className="w-3 h-3 ml-1" />
                    </Button>
                  </a>
                </div>
              )}

              {/* Action bar */}
              <div className="px-5 py-4 flex items-center gap-2">
                <button
                  onClick={() => likeMutation.mutate()}
                  disabled={likeMutation.isPending}
                  className={cn(
                    'flex items-center gap-2 px-4 py-2 rounded-lg text-[13px] font-medium transition-all duration-200',
                    log.is_liked
                      ? 'bg-danger/10 text-danger'
                      : 'text-text-secondary hover:bg-white/[0.06] hover:text-text-primary',
                  )}
                >
                  <Heart
                    className={cn('w-4 h-4', log.is_liked && 'fill-current')}
                  />
                  {log.like_count || '点赞'}
                </button>

                <button
                  onClick={() => collectMutation.mutate()}
                  disabled={collectMutation.isPending}
                  className={cn(
                    'flex items-center gap-2 px-4 py-2 rounded-lg text-[13px] font-medium transition-all duration-200',
                    log.is_collected
                      ? 'bg-amber/10 text-amber'
                      : 'text-text-secondary hover:bg-white/[0.06] hover:text-text-primary',
                  )}
                >
                  <Bookmark
                    className={cn('w-4 h-4', log.is_collected && 'fill-current')}
                  />
                  {log.collect_count || '收藏'}
                </button>

                <button
                  onClick={handleShare}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg text-[13px] font-medium text-text-secondary hover:bg-white/[0.06] hover:text-text-primary transition-all duration-200"
                >
                  <Share2 className="w-4 h-4" />
                  分享
                </button>

                <div className="flex items-center gap-1.5 ml-auto text-[12px] text-text-muted">
                  <Eye className="w-4 h-4" />
                  <span className="font-mono">{log.view_count}</span>
                </div>
              </div>
            </div>

            {/* ── Comments section ── */}
            <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 mt-6">
              <CommentSection config={commentConfig} currentUser={user} />
            </div>
          </main>

          {/* ── Sidebar ── */}
          <aside className="hidden lg:block w-[280px] shrink-0">
            <div className="sticky top-[88px]">
              <DevlogSidebar log={log} currentUser={user} />
            </div>
          </aside>
        </div>
      </div>

      {/* Delete log confirmation dialog */}
      <Dialog open={deleteLogDialogOpen} onOpenChange={setDeleteLogDialogOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>删除日志</DialogTitle><DialogDescription>确定要删除这篇开发日志吗？<strong className="text-danger">此操作不可撤销。</strong></DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteLogDialogOpen(false)}>取消</Button>
            <Button variant="danger" loading={deleteLogMutation.isPending} onClick={() => {
              deleteLogMutation.mutate()
              setDeleteLogDialogOpen(false)
            }}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Publish log confirmation dialog */}
      <Dialog open={publishDialogOpen} onOpenChange={setPublishDialogOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>发布日志</DialogTitle><DialogDescription>确定要发布这篇日志吗？发布后将公开可见。</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setPublishDialogOpen(false)}>取消</Button>
            <Button loading={publishMutation.isPending} onClick={() => {
              publishMutation.mutate()
              setPublishDialogOpen(false)
            }}>确认发布</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

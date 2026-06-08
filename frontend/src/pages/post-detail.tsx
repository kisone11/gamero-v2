import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Heart, Bookmark, Share2, Edit, Trash2, ArrowLeft, MessageSquare,
  Eye, Pin,
} from 'lucide-react'
import { communityApi } from '@/api/community'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { Markdown } from '@/components/ui/markdown'
import { toast } from '@/stores/toastStore'
import { ReportDialog } from '@/components/shared/report-dialog'
import { openLightbox } from '@/components/shared/ImageGallery'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { timeAgo } from '@/lib/time'
import type { PostComment } from '@/types/api'
import { type CommentItemData } from '@/components/shared/CommentItem'
import { CommentSection, type CommentSectionConfig } from '@/components/shared/CommentSection'
import { PostDetailSidebar, PostSkeleton } from '@/components/post'

// ============================================================
// Adapter: PostComment -> CommentItemData
// ============================================================

function toCommentData(c: PostComment, currentUserId?: number): CommentItemData {
  return {
    id: c.id,
    content: c.content,
    authorName: c.author.nickname || c.author.username || '用户',
    authorUsername: c.author.username || '',
    authorAvatar: c.author.avatar_url,
    likeCount: c.like_count,
    isLiked: c.is_liked,
    isDeleted: c.is_deleted,
    createdAt: c.created_at,
    isAuthor: currentUserId != null && c.author_id === currentUserId,
    replies: c.replies?.map(r => toCommentData(r, currentUserId)),
  }
}

// ============================================================
// PostDetailPage
// ============================================================

export default function PostDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const user = useAuthStore((s) => s.user)

  const postId = Number(id)
  const [deletePostOpen, setDeletePostOpen] = useState(false)

  const postQuery = useQuery({
    queryKey: ['post', postId],
    queryFn: () => communityApi.get(postId),
    enabled: !!postId && !isNaN(postId),
  })

  const post = postQuery.data
  const author = post?.author

  // ── Mutations ──

  const likeMutation = useMutation({
    mutationFn: () => {
      if (!user) throw new Error('请先登录')
      return post?.is_liked ? communityApi.unlike(postId) : communityApi.like(postId)
    },
    onMutate: () => {
      // Optimistic update
      queryClient.setQueryData(['post', postId], (old: any) =>
        old ? { ...old, is_liked: !old.is_liked, like_count: old.like_count + (old.is_liked ? -1 : 1) } : old)
    },
    onError: () => {
      queryClient.invalidateQueries({ queryKey: ['post', postId] })
      toast.error('操作失败，请重试')
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['post', postId] }),
  })

  const collectMutation = useMutation({
    mutationFn: () => {
      if (!user) throw new Error('请先登录')
      return post?.is_collected ? communityApi.uncollect(postId) : communityApi.collect(postId)
    },
    onMutate: () => {
      queryClient.setQueryData(['post', postId], (old: any) =>
        old ? { ...old, is_collected: !old.is_collected, collect_count: old.collect_count + (old.is_collected ? -1 : 1) } : old)
    },
    onError: () => {
      queryClient.invalidateQueries({ queryKey: ['post', postId] })
      toast.error('操作失败，请重试')
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['post', postId] }),
  })

  const deletePostMutation = useMutation({
    mutationFn: () => communityApi.delete(postId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['community-posts'] })
      toast.success('帖子已删除')
      navigate('/community')
    },
    onError: () => toast.error('删除失败'),
  })

  // ── Handlers ──

  const handleShare = () => {
    if (navigator.share) {
      navigator.share({ title: post?.title, url: window.location.href }).catch(() => { })
    } else {
      navigator.clipboard.writeText(window.location.href).then(
        () => toast.success('链接已复制'),
        () => toast.error('复制失败'),
      )
    }
  }

  const handleEdit = () => {
    navigate(`/post/${postId}/edit`)
  }

  const handleDelete = () => setDeletePostOpen(true)
  const confirmDeletePost = () => { deletePostMutation.mutate(); setDeletePostOpen(false) }
  const handleLike = () => {
    if (!user) {
      toast.error('请先登录')
      navigate('/login')
      return
    }
    likeMutation.mutate()
  }
  const handleCollect = () => {
    if (!user) {
      toast.error('请先登录')
      navigate('/login')
      return
    }
    collectMutation.mutate()
  }

  // ── Loading ──
  if (postQuery.isLoading) {
    return <PostSkeleton />
  }

  // ── Error ──
  if (postQuery.isError) {
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
            无法加载帖子信息，请检查网络后重试
          </p>
          <Button variant="secondary" size="sm" onClick={() => postQuery.refetch()}>
            重新加载
          </Button>
        </div>
      </div>
    )
  }

  // ── Not found ──
  if (!post) {
    return (
      <div className="max-w-[720px] mx-auto px-6 py-8 min-h-[60vh] flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <MessageSquare className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">
            帖子不存在
          </p>
          <p className="text-[13px] text-text-muted mb-6">
            该帖子可能已被删除或链接地址有误
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
  const PAGE_SIZE = 20
  const commentConfig: CommentSectionConfig = {
    queryKey: ['post-comments', postId],
    fetchComments: (page, sort) => communityApi.getComments(postId, page, PAGE_SIZE, sort),
    createComment: (content, replyToId) => communityApi.createComment(postId, { content, parent_id: replyToId ?? 0 }),
    likeComment: (id) => communityApi.likeComment(postId, id),
    unlikeComment: (id) => communityApi.unlikeComment(postId, id),
    deleteComment: (id) => communityApi.deleteComment(postId, id),
    updateComment: (id, content) => communityApi.updateComment(postId, id, content),
    normalizeComment: (c: any, uid?: number) => toCommentData(c as PostComment, uid),
  }

  return (
    <div className="max-w-[1060px] mx-auto px-6 py-8">
      <div className="flex gap-8">
        {/* ── Main content ── */}
        <main className="flex-1 min-w-0 max-w-[720px]">
          {/* Post card */}
          <div className="bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden">
            {/* Header */}
            <div className="p-5 border-b border-white/[0.04]">
              {/* Back and actions */}
              <div className="flex items-center justify-between mb-4">
                <Link
                  to="/community"
                  className="flex items-center gap-1.5 text-[13px] text-text-muted hover:text-amber transition-colors"
                >
                  <ArrowLeft className="w-4 h-4" />
                  返回社区
                </Link>
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
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={handleDelete}
                        className="text-danger hover:text-danger"
                        loading={deletePostMutation.isPending}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </>
                  )}
                  {user && !isOwner && (
                    <ReportDialog targetType="post" targetId={post.id} />
                  )}
                </div>
              </div>

              {/* Topic badge & pin */}
              <div className="flex items-center gap-2 mb-3">
                {post.topic && (
                  <span className="rounded-full bg-amber/10 text-amber px-3 py-1 text-[11px] font-medium">
                    {post.topic.name}
                  </span>
                )}
                {post.is_pinned && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-white/[0.06] text-text-secondary px-3 py-1 text-[11px] font-medium">
                    <Pin className="w-3 h-3" />
                    置顶
                  </span>
                )}
                <span className="text-[12px] text-text-muted font-mono ml-auto">
                  {timeAgo(post.created_at)}
                </span>
              </div>

              {/* Title */}
              <h1 className="text-[24px] font-bold text-text-primary leading-tight">
                {post.title}
              </h1>
            </div>

            {/* Content */}
            <div className="p-5 border-b border-white/[0.04]">
              <Markdown content={post.content} />
            </div>

            {/* Image gallery */}
            {post.image_urls && post.image_urls.length > 0 && (
              <div className="p-5 border-b border-white/[0.04]">
                <div className="flex flex-wrap gap-2">
                  {post.image_urls.map((url, i) => (
                    <button key={i} onClick={() => openLightbox(post.image_urls!, i)}
                      className="w-[calc(33.33%-0.5rem)] aspect-[4/3] bg-black/20 rounded-lg overflow-hidden cursor-pointer hover:opacity-90 transition-opacity border border-white/[0.04]">
                      <img src={url} alt="" className="w-full h-full object-cover" loading="lazy" />
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Video gallery */}
            {post.video_urls && post.video_urls.length > 0 && (
              <div className="p-5 border-b border-white/[0.04]">
                <div className="space-y-3">
                  {post.video_urls.map((url, i) => (
                    <div key={i} className="rounded-xl overflow-hidden border border-white/[0.04]">
                      <video src={url} controls className="w-full max-h-96"
                        preload="metadata" />
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Action bar */}
            <div className="px-5 py-4 flex items-center gap-2">
              <button
                onClick={handleLike}
                disabled={likeMutation.isPending}
                className={cn(
                  'flex items-center gap-2 px-4 py-2 rounded-lg text-[13px] font-medium transition-all duration-200',
                  post.is_liked
                    ? 'bg-danger/10 text-danger'
                    : 'text-text-secondary hover:bg-white/[0.06] hover:text-text-primary',
                )}
              >
                <Heart
                  className={cn('w-4 h-4', post.is_liked && 'fill-current')}
                />
                {post.like_count || '点赞'}
              </button>

              <button
                onClick={handleCollect}
                disabled={collectMutation.isPending}
                className={cn(
                  'flex items-center gap-2 px-4 py-2 rounded-lg text-[13px] font-medium transition-all duration-200',
                  post.is_collected
                    ? 'bg-amber/10 text-amber'
                    : 'text-text-secondary hover:bg-white/[0.06] hover:text-text-primary',
                )}
              >
                <Bookmark
                  className={cn('w-4 h-4', post.is_collected && 'fill-current')}
                />
                {post.collect_count || '收藏'}
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
                <span className="font-mono">{post.view_count}</span>
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
            <PostDetailSidebar author={author} currentUser={user} post={post} />
          </div>
        </aside>
      </div>

      <Dialog open={deletePostOpen} onOpenChange={setDeletePostOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>确认删除</DialogTitle><DialogDescription>确定要删除这个帖子吗？<strong className="text-danger">此操作不可撤销。</strong></DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeletePostOpen(false)}>取消</Button>
            <Button variant="danger" loading={deletePostMutation.isPending} onClick={confirmDeletePost}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

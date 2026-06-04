import { useState } from 'react'
import { Send } from 'lucide-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Pagination } from '@/components/ui/pagination'
import { CommentItem, type CommentItemData } from '@/components/shared/CommentItem'
import { toast } from '@/stores/toastStore'

type CommentSort = 'latest' | 'hot'

export interface CommentSectionConfig {
  queryKey: string
  fetchComments: (page: number, sort: CommentSort) => Promise<{ list: any[]; pages: number; page: number }>
  createComment: (content: string, replyToId?: number) => Promise<any>
  likeComment: (commentId: number) => Promise<any>
  unlikeComment: (commentId: number) => Promise<any>
  deleteComment: (commentId: number) => Promise<any>
  updateComment?: (commentId: number, content: string) => Promise<any>
  normalizeComment: (comment: any, currentUserId?: number) => CommentItemData
}

interface CommentSectionProps {
  config: CommentSectionConfig
  currentUser: { id: number } | null
}

export function CommentSection({ config, currentUser }: CommentSectionProps) {
  const queryClient = useQueryClient()
  const [sort, setSort] = useState<CommentSort>('latest')
  const [commentPage, setCommentPage] = useState(1)
  const [commentText, setCommentText] = useState('')
  const [replyTo, setReplyTo] = useState<CommentItemData | null>(null)
  const PAGE_SIZE = 20

  const commentsQuery = useQuery({
    queryKey: [config.queryKey, sort, commentPage],
    queryFn: () => config.fetchComments(commentPage, sort),
  })

  const createMutation = useMutation({
    mutationFn: async (payload: { content: string; replyToId?: number }) => {
      await config.createComment(payload.content, payload.replyToId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [config.queryKey] })
      setCommentText('')
      setReplyTo(null)
      toast.success('评论发布成功')
    },
    onError: () => toast.error('评论发布失败'),
  })

  const likeMutation = useMutation({
    mutationFn: async ({ commentId, isLiked }: { commentId: number; isLiked: boolean }) => {
      if (isLiked) await config.unlikeComment(commentId)
      else await config.likeComment(commentId)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [config.queryKey] }),
  })

  const deleteMutation = useMutation({
    mutationFn: async (commentId: number) => { await config.deleteComment(commentId) },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: [config.queryKey] }); toast.success('评论已删除') },
  })

  const editMutation = useMutation({
    mutationFn: async ({ commentId, content }: { commentId: number; content: string }) => {
      if (config.updateComment) await config.updateComment(commentId, content)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [config.queryKey] }),
  })

  const comments: CommentItemData[] = (commentsQuery.data?.list ?? []).map(
    (c: any) => config.normalizeComment(c, currentUser?.id)
  )
  const pages = commentsQuery.data?.pages ?? 1

  const handleSubmit = () => {
    if (!commentText.trim()) return
    createMutation.mutate({ content: commentText.trim(), replyToId: replyTo?.id })
  }

  const sortOptions: { value: CommentSort; label: string }[] = [
    { value: 'latest', label: '最新' },
    { value: 'hot', label: '最热' },
  ]

  return (
    <div>
      {/* Sort tabs */}
      <div className="flex items-center gap-1 mb-5">
        {sortOptions.map((opt) => (
          <button
            key={opt.value}
            onClick={() => { setSort(opt.value); setCommentPage(1) }}
            className={`px-3 py-1.5 text-small rounded-md transition-colors ${
              sort === opt.value ? 'bg-amber/10 text-amber' : 'text-text-muted hover:text-text-secondary'
            }`}
          >
            {opt.label}
          </button>
        ))}
      </div>

      {/* Comment input */}
      {currentUser && (
        <div className="mb-6">
          {replyTo && (
            <div className="flex items-center gap-2 mb-2 text-small text-text-muted">
              回复 @{replyTo.authorName}
              <button onClick={() => setReplyTo(null)} className="text-amber hover:underline">取消</button>
            </div>
          )}
          <div className="flex gap-2">
            <textarea
              value={commentText}
              onChange={(e) => setCommentText(e.target.value)}
              placeholder={replyTo ? `回复 @${replyTo.authorName}...` : '发表评论...'}
              className="flex-1 bg-surface-deep border border-white/[0.08] rounded-lg px-3 py-2 text-body text-text-primary placeholder:text-text-muted resize-none focus:outline-none focus:border-amber"
              rows={2}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleSubmit()
              }}
            />
            <Button size="sm" variant="primary" onClick={handleSubmit} loading={createMutation.isPending}>
              <Send className="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>
      )}

      {/* Comments list */}
      {commentsQuery.isLoading ? (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="flex gap-3">
              <Skeleton className="w-8 h-8 rounded-full shrink-0" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-4 w-full" />
              </div>
            </div>
          ))}
        </div>
      ) : commentsQuery.isError ? (
        <div className="text-center py-8">
          <p className="text-body text-text-muted">评论加载失败</p>
          <Button variant="secondary" size="sm" onClick={() => commentsQuery.refetch()} className="mt-2">重试</Button>
        </div>
      ) : comments.length === 0 ? (
        <div className="text-center py-8">
          <p className="text-body text-text-muted">暂无评论</p>
        </div>
      ) : (
        <div>
          {comments.map((comment) => (
            <CommentItem
              key={comment.id}
              comment={comment}
              currentUser={currentUser}
              onLike={(id, isLiked) => likeMutation.mutate({ commentId: id, isLiked })}
              onDelete={(id) => deleteMutation.mutate(id)}
              onReply={(c) => setReplyTo(c)}
              onEdit={config.updateComment ? (id, content) => editMutation.mutate({ commentId: id, content }) : undefined}
            />
          ))}
          {pages > 1 && (
            <Pagination page={commentsQuery.data?.page ?? 1} pages={pages} onChange={setCommentPage} />
          )}
        </div>
      )}
    </div>
  )
}

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Star, MessageSquare, ThumbsUp, Trash2 } from 'lucide-react'
import { reviewApi } from '@/api/review'
import { Button, Badge, Card, Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, EmptyState, Pagination } from '@/components/ui'
import { Avatar } from '@/components/ui/avatar'
import { Textarea } from '@/components/ui/textarea'
import { timeAgo, formatDate } from '@/lib/time'
import { cn } from '@/lib/utils'
import { toast } from '@/stores/toastStore'
import { StarRating } from './StarRating'
import type { ReviewItem, ReviewRatingSummary } from '@/types/api'

export function ProjectReviewsTab({
  projectId, currentUser, isOwner,
}: {
  projectId: number
  currentUser: { id: number } | null
  isOwner: boolean
}) {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [reviewModalOpen, setReviewModalOpen] = useState(false)
  const [reviewRating, setReviewRating] = useState(5)
  const [reviewContent, setReviewContent] = useState('')
  const [deleteReviewTarget, setDeleteReviewTarget] = useState<number | null>(null)
  const [replyText, setReplyText] = useState<Record<number, string>>({})
  const [replyOpen, setReplyOpen] = useState<Record<number, boolean>>({})

  const { data: summary } = useQuery({
    queryKey: ['review-summary', projectId],
    queryFn: () => reviewApi.getSummary(projectId),
  })

  const { data: reviewsData, isLoading } = useQuery({
    queryKey: ['project-reviews', projectId, page],
    queryFn: () => reviewApi.list(projectId, page, 20),
  })

  const createReviewMutation = useMutation({
    mutationFn: () =>
      reviewApi.create(projectId, { rating: reviewRating, content: reviewContent }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-reviews', projectId] })
      queryClient.invalidateQueries({ queryKey: ['review-summary', projectId] })
      setReviewModalOpen(false)
      toast.success('评测已发布')
    },
    onError: () => toast.error('发布评测失败'),
  })

  const reviewLikeMutation = useMutation({
    mutationFn: ({ reviewId, isLiked }: { reviewId: number; isLiked: boolean }) =>
      isLiked ? reviewApi.unlike(projectId, reviewId) : reviewApi.like(projectId, reviewId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-reviews', projectId] })
    },
    onError: () => toast.error('操作失败'),
  })

  const reviewDeleteMutation = useMutation({
    mutationFn: (reviewId: number) => reviewApi.delete(projectId, reviewId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-reviews', projectId] })
      queryClient.invalidateQueries({ queryKey: ['review-summary', projectId] })
      toast.success('评测已删除')
    },
    onError: () => toast.error('删除失败'),
  })

  const reviewReplyMutation = useMutation({
    mutationFn: ({ reviewId, content }: { reviewId: number; content: string }) =>
      reviewApi.reply(projectId, reviewId, content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-reviews', projectId] })
      toast.success('回复成功')
    },
    onError: () => toast.error('回复失败'),
  })

  const reviews = reviewsData?.list ?? []
  const pages = reviewsData?.pages ?? 1

  // Normalize distribution: backend sends map {1:5,2:3} or array [{rating:1,count:5}]
  const distMap: Record<number, number> = {}
  if (summary?.distribution) {
    if (Array.isArray(summary.distribution)) {
      summary.distribution.forEach((d: any) => { distMap[d.rating] = d.count })
    } else {
      Object.entries(summary.distribution).forEach(([k, v]) => { distMap[Number(k)] = v as number })
    }
  }
  const dist = [5, 4, 3, 2, 1].map(rating => {
    const count = distMap[rating] ?? 0
    return { rating, count, pct: summary?.total ? (count / summary.total) * 100 : 0 }
  })

  return (
    <div className="py-6 space-y-6">
      {/* Rating summary */}
      {summary && summary.total > 0 && (
        <Card padding="lg">
          <div className="flex items-start gap-6 md:gap-10">
            <div className="text-center shrink-0">
              <div className="text-[32px] font-bold text-amber">{summary.average.toFixed(1)}</div>
              <StarRating value={Math.round(summary.average)} readonly size="sm" />
              <p className="text-small text-text-muted font-mono mt-1">{summary.total} 条评测</p>
            </div>
            <div className="flex-1 space-y-1.5 min-w-0">
              {dist.map(d => (
                <div key={d.rating} className="flex items-center gap-2">
                  <span className="text-small text-text-secondary font-mono w-4 text-right">{d.rating}</span>
                  <Star className="h-3 w-3 text-amber fill-current shrink-0" />
                  <div className="flex-1 h-2 bg-white/[0.04] rounded-full overflow-hidden">
                    <div className="h-full bg-amber rounded-full transition-all" style={{ width: `${d.pct}%` }} />
                  </div>
                  <span className="text-small text-text-muted font-mono w-6 text-right">{d.count}</span>
                </div>
              ))}
            </div>
          </div>
        </Card>
      )}

      {/* Write review */}
      {currentUser && (
        <div className="flex justify-end">
          <Button size="sm" onClick={() => {
            setReviewRating(5)
            setReviewContent('')
            setReviewModalOpen(true)
          }}>
            <Star className="h-4 w-4" />
            写评测
          </Button>
        </div>
      )}

      {/* List */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i} padding="lg" className="space-y-3">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-lg bg-white/[0.04] animate-pulse" />
                <div className="space-y-1.5 flex-1">
                  <div className="h-3 w-24 bg-white/[0.04] animate-pulse rounded" />
                  <div className="h-3 w-full bg-white/[0.04] animate-pulse rounded" />
                </div>
              </div>
            </Card>
          ))}
        </div>
      ) : reviews.length === 0 ? (
        <EmptyState
          icon={<MessageSquare className="w-7 h-7 text-text-muted" />}
          title="暂无评测"
          description={currentUser ? '来写下第一条评测吧' : '还没有人为这个项目写评测'}
          action={currentUser ? { label: '写评测', onClick: () => {
            setReviewRating(5)
            setReviewContent('')
            setReviewModalOpen(true)
          } } : undefined}
        />
      ) : (
        <div className="space-y-3">
          {reviews.map(review => (
            <Card key={review.id} padding="lg">
              <div className="flex items-start gap-3">
                <Avatar src={review.user_avatar_url} name={review.user_nickname} size="sm" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1 flex-wrap">
                    <span className="text-h4 text-text-primary">{review.user_nickname}</span>
                    <StarRating value={review.rating} readonly size="sm" />
                    <span className="text-small text-text-muted font-mono ml-auto">{timeAgo(review.created_at)}</span>
                  </div>
                  <p className="text-body text-text-secondary whitespace-pre-wrap">{review.content}</p>
                  <div className="flex items-center gap-3 mt-3">
                    <button
                      onClick={() => reviewLikeMutation.mutate({ reviewId: review.id, isLiked: review.is_liked })}
                      className={cn(
                        'flex items-center gap-1 text-small font-mono transition-colors',
                        review.is_liked ? 'text-amber' : 'text-text-muted hover:text-text-secondary',
                      )}
                    >
                      <ThumbsUp className="h-3.5 w-3.5" />
                      {review.like_count}
                    </button>
                    {currentUser && review.user_id === currentUser.id && (
                      <button
                        onClick={() => setDeleteReviewTarget(review.id)}
                        className="flex items-center gap-1 text-small font-mono text-text-muted hover:text-danger transition-colors"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                  {isOwner && !review.reply_content && (
                    <div className="mt-3">
                      {replyOpen[review.id] ? (
                        <div className="space-y-2">
                          <Textarea
                            placeholder="写下你的回复..."
                            value={replyText[review.id] ?? ''}
                            onChange={(e) => setReplyText(prev => ({ ...prev, [review.id]: e.target.value }))}
                            rows={3}
                          />
                          <div className="flex items-center gap-2">
                            <Button size="sm" onClick={() => {
                              reviewReplyMutation.mutate({ reviewId: review.id, content: replyText[review.id] ?? '' })
                              setReplyText(prev => ({ ...prev, [review.id]: '' }))
                              setReplyOpen(prev => ({ ...prev, [review.id]: false }))
                            }}>
                              提交
                            </Button>
                            <Button variant="ghost" size="sm" onClick={() => {
                              setReplyOpen(prev => ({ ...prev, [review.id]: false }))
                              setReplyText(prev => ({ ...prev, [review.id]: '' }))
                            }}>
                              取消
                            </Button>
                          </div>
                        </div>
                      ) : (
                        <button
                          onClick={() => setReplyOpen(prev => ({ ...prev, [review.id]: true }))}
                          className="flex items-center gap-1 text-small font-mono text-text-muted hover:text-amber transition-colors"
                        >
                          <MessageSquare className="h-3.5 w-3.5" />
                          回复
                        </button>
                      )}
                    </div>
                  )}
                  {review.reply_content && (
                    <div className="mt-3 p-3 bg-white/[0.02] rounded-xl border border-white/[0.04]">
                      <p className="text-small font-semibold text-amber mb-1">开发者回复</p>
                      <p className="text-body text-text-secondary whitespace-pre-wrap">{review.reply_content}</p>
                      {review.replied_at && (
                        <p className="text-small text-text-muted font-mono mt-1">{formatDate(review.replied_at)}</p>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </Card>
          ))}
          <Pagination page={page} pages={pages} onChange={setPage} />
        </div>
      )}

      {/* Delete review confirmation dialog */}
      <Dialog open={deleteReviewTarget !== null} onOpenChange={(o) => { if (!o) setDeleteReviewTarget(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>删除评测</DialogTitle><DialogDescription>确定要删除这条评测吗？</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteReviewTarget(null)}>取消</Button>
            <Button variant="danger" onClick={() => {
              if (deleteReviewTarget !== null) {
                reviewDeleteMutation.mutate(deleteReviewTarget)
                setDeleteReviewTarget(null)
              }
            }}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Create review dialog */}
      <Dialog open={reviewModalOpen} onOpenChange={setReviewModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>写评测</DialogTitle>
            <DialogDescription>分享你对这个项目的看法</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <span className="text-small font-semibold text-text-secondary">评分</span>
              <StarRating value={reviewRating} onChange={setReviewRating} />
            </div>
            <Textarea
              placeholder="写下你的评测内容……"
              value={reviewContent}
              onChange={e => setReviewContent(e.target.value)}
              rows={5}
            />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setReviewModalOpen(false)}>取消</Button>
            <Button
              loading={createReviewMutation.isPending}
              disabled={!reviewContent.trim()}
              onClick={() => createReviewMutation.mutate()}
            >
              发布评测
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

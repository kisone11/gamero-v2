import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Button, Badge } from '@/components/ui'
import { collabApi } from '@/api/collab'
import { cn } from '@/lib/utils'
import { Avatar } from '@/components/ui/avatar'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Textarea } from '@/components/ui/textarea'
import { StarRating } from './StarRating'
import { MEMBER_ROLE_LABELS } from '@/lib/constants'
import type { MemberDetail } from '@/types/api'

export function CollabReviewDialog({
  open, onOpenChange, target, onSubmit, loading,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  target: MemberDetail | null
  onSubmit: (data: { reviewee_id: number; rating: number; comment: string; tags?: string[] }) => void
  loading: boolean
}) {
  const [rating, setRating] = useState(5)
  const [comment, setComment] = useState('')
  const [tags, setTags] = useState<string[]>([])
  const { data: tagData } = useQuery({ queryKey: ['review-tags'], queryFn: () => collabApi.getReviewTags(), enabled: open })

  useEffect(() => {
    if (open) {
      setRating(5)
      setComment('')
      setTags([])
    }
  }, [open])

  const toggleTag = (tag: string) => {
    setTags((prev) => prev.includes(tag) ? prev.filter((item) => item !== tag) : [...prev, tag])
  }

  const handleSubmit = () => {
    if (!target || !comment.trim()) return
    onSubmit({ reviewee_id: target.user_id, rating, comment, tags })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>评价队友</DialogTitle>
          {target && (
            <DialogDescription>
              为 {target.nickname || target.username} 撰写协作评价
            </DialogDescription>
          )}
        </DialogHeader>
        <div className="space-y-5">
          {target && (
            <div className="flex items-center gap-3 pb-2 border-b border-white/[0.04]">
              <Avatar src={target.avatar_url} name={target.nickname} size="md" />
              <div>
                <p className="text-h4 text-text-primary">{target.nickname || target.username}</p>
                <Badge variant="default" size="sm">{MEMBER_ROLE_LABELS[target.role]}</Badge>
              </div>
            </div>
          )}
          <div className="flex items-center gap-3">
            <span className="text-small font-semibold text-text-secondary shrink-0">评分</span>
            <StarRating value={rating} onChange={setRating} />
          </div>
          <Textarea
            placeholder="分享你与这位队友的合作体验……"
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            rows={4}
          />
          {(tagData?.tags?.length ?? 0) > 0 && (
            <div>
              <p className="text-small font-semibold text-text-secondary mb-2">协作标签</p>
              <div className="flex flex-wrap gap-2">
                {tagData!.tags.map((tag) => (
                  <button
                    key={tag}
                    type="button"
                    onClick={() => toggleTag(tag)}
                    className={cn(
                      'px-3 py-1.5 rounded-lg border text-[12px] transition-colors',
                      tags.includes(tag) ? 'border-amber/30 bg-amber/10 text-amber' : 'border-white/[0.06] text-text-muted hover:text-text-secondary',
                    )}
                  >
                    {tag}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>取消</Button>
          <Button loading={loading} disabled={!comment.trim()} onClick={handleSubmit}>提交评价</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

import { Star } from 'lucide-react'
import { Avatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { timeAgo } from '@/lib/time'
import type { CollabReview } from '@/types/api'

function renderStars(rating: number) {
  return Array.from({ length: 5 }).map((_, i) => (
    <Star key={i} className={`w-3.5 h-3.5 ${i < rating ? 'text-amber fill-amber' : 'text-text-muted'}`} />
  ))
}

export function EndorsementCard({ review }: { review: CollabReview }) {
  return (
    <Card padding="md">
      <div className="flex items-start gap-3">
        <Avatar
          src={review.reviewer_avatar_url}
          name={review.reviewer_nickname}
          size="sm"
        />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 mb-1.5">
            <span className="text-body font-medium text-text-primary">
              {review.reviewer_nickname}
            </span>
            <span className="text-body text-amber font-mono">{renderStars(review.rating)}</span>
            <span className="text-caption text-text-muted font-mono">{timeAgo(review.created_at)}</span>
          </div>
          <p className="text-body text-text-secondary leading-relaxed line-clamp-2 mb-2">{review.comment}</p>
          {review.tags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {review.tags.map((tag) => (
                <Badge key={tag} variant="default" size="sm">{tag}</Badge>
              ))}
            </div>
          )}
        </div>
      </div>
    </Card>
  )
}

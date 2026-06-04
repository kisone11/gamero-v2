import { Link } from 'react-router-dom'
import { FileText, Heart, MessageSquare, Eye } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Avatar } from '@/components/ui/avatar'
import { ImageGallery } from '@/components/shared/ImageGallery'
import { timeAgo } from '@/lib/time'
import { stripMarkdown } from '@/lib/markdown'
import type { Post } from '@/types/api'

interface PostCardProps {
  post: Post
  showAuthor?: boolean
  showPinned?: boolean
}

export function PostCard({ post, showAuthor = false, showPinned = false }: PostCardProps) {
  const author = post.author
  const topic = post.topics?.[0] ?? post.topic ?? null

  return (
    <Link
      to={`/post/${post.id}`}
      className="block bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden
                 hover:border-amber/12 hover:shadow-card-hover
                 hover:-translate-y-[1px] transition-all duration-200 group"
    >
      <div className="p-5">
        {showPinned && post.is_pinned && (
          <div className="flex items-center gap-1 mb-3">
            <Badge variant="primary" size="sm">置顶</Badge>
          </div>
        )}

        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-lg bg-purple-500/10 flex items-center justify-center shrink-0 ring-1 ring-purple-500/20">
            <FileText className="w-5 h-5 text-purple-400" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 mb-1.5">
              {showAuthor && author && (
                <>
                  <Avatar
                    src={author.avatar_url}
                    name={author.nickname ?? author.username ?? '?'}
                    size="xs"
                  />
                  <span className="text-h4 text-text-primary">
                    {author.nickname ?? author.username ?? '用户'}
                  </span>
                  <span className="text-caption text-text-muted font-mono">
                    @{author.username}
                  </span>
                  <span className="text-caption text-text-muted">· </span>
                </>
              )}
              <span className="text-caption text-text-muted font-mono">{timeAgo(post.created_at)}</span>
            </div>

            <h3 className="text-h3 text-text-primary leading-snug mb-1.5">
              {post.title}
            </h3>

            {(post.content_snippet || post.content) && (
              <p className="text-body text-text-secondary leading-relaxed line-clamp-2 mb-2.5">
                {stripMarkdown(post.content_snippet || post.content)}
              </p>
            )}

            <div className="flex items-center justify-between mt-2">
              <div className="flex items-center gap-2 flex-wrap">
                {topic && (
                  <Badge variant="default" size="sm">{topic.name}</Badge>
                )}
              </div>
              <div className="flex items-center gap-3 text-text-muted">
                <span className="flex items-center gap-1 text-caption font-mono">
                  <Heart className="w-3 h-3" />{post.like_count ?? 0}
                </span>
                <span className="flex items-center gap-1 text-caption font-mono">
                  <MessageSquare className="w-3 h-3" />{post.comment_count ?? 0}
                </span>
                <span className="flex items-center gap-1 text-caption font-mono">
                  <Eye className="w-3 h-3" />{post.view_count ?? 0}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
      {(post.image_urls?.length ?? 0) > 0 && (
        <div className="border-t border-white/[0.04]" onClick={(e) => e.preventDefault()}>
          <ImageGallery images={post.image_urls ?? []} thumbnail maxShow={3} />
        </div>
      )}
    </Link>
  )
}

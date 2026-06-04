import { useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Heart, Bookmark, MessageSquare, Eye, Users } from 'lucide-react'
import { userApi } from '@/api/user'
import { communityApi } from '@/api/community'
import { Avatar } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { CalendarIcon } from '@/components/shared/CalendarIcon'
import { timeAgo, formatDate } from '@/lib/time'
import { toast } from '@/stores/toastStore'
import type { Post, UserInfo } from '@/types/api'

function SidebarStatRow({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode
  label: string
  value: string | number
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="flex items-center gap-1.5 text-body text-text-secondary">
        {icon}
        {label}
      </span>
      <span className="text-body font-semibold text-text-primary font-mono">
        {value ?? '0'}
      </span>
    </div>
  )
}

export function PostDetailSidebar({
  author,
  currentUser,
  post,
}: {
  author: Post['author']
  currentUser: UserInfo | null
  post: Post
}) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const username = author?.username

  // Profile for follow status
  const profileQuery = useQuery({
    queryKey: ['user-profile', username],
    queryFn: () => userApi.getProfile(username!),
    enabled: !!username,
  })

  const profile = profileQuery.data

  const followMutation = useMutation({
    mutationFn: () => {
      if (!profile) throw new Error('profile not loaded')
      return profile.is_following
        ? userApi.unfollow(profile.id)
        : userApi.follow(profile.id)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-profile', username] })
      toast.success(profile?.is_following ? '已取消关注' : '已关注')
    },
    onError: () => toast.error('操作失败，请重试'),
  })

  // Related posts from same topic
  const topicId = post.topic_ids?.[0] || post.topics?.[0]?.id || post.topic?.id

  const relatedPostsQuery = useQuery({
    queryKey: ['related-posts', topicId],
    queryFn: () => communityApi.list({ topic_id: topicId!, page_size: 5 }),
    enabled: !!topicId,
  })

  const relatedPosts =
    relatedPostsQuery.data?.list?.filter((p) => p.id !== post.id) ?? []

  return (
    <div className="space-y-4">
      {/* Author card */}
      {author && (
        <Card>
          <div className="flex flex-col items-center text-center">
            <Link to={`/u/${author.username || author.id}`}>
              <Avatar
                src={author.avatar_url}
                name={author.nickname}
                size="lg"
              />
            </Link>
            <Link
              to={`/u/${author.username || author.id}`}
              className="mt-3 text-h3 text-text-primary hover:text-amber transition-colors"
            >
              {author.nickname || author.username}
            </Link>
            <p className="text-small text-text-muted font-mono mt-0.5">
              @{author.username}
            </p>
          </div>

          {/* Stats */}
          {profile && (
            <div className="flex justify-center gap-4 mt-4 pt-4 border-t border-white/[0.04]">
              <div className="text-center">
                <div className="text-body font-bold text-amber font-mono">
                  {profile.followers_count}
                </div>
                <div className="text-caption text-text-muted">关注者</div>
              </div>
              <div className="text-center">
                <div className="text-body font-bold text-amber font-mono">
                  {profile.following_count}
                </div>
                <div className="text-caption text-text-muted">关注中</div>
              </div>
              <div className="text-center">
                <div className="text-body font-bold text-amber font-mono">
                  {profile.posts?.length ?? 0}
                </div>
                <div className="text-caption text-text-muted">帖子</div>
              </div>
            </div>
          )}

          {/* Follow / Edit */}
          {currentUser && author.id !== currentUser.id && (
            <div className="mt-3">
              <Button
                variant={profile?.is_following ? 'secondary' : 'primary'}
                size="sm"
                className="w-full"
                loading={followMutation.isPending}
                onClick={() => followMutation.mutate()}
              >
                <Users className="w-3.5 h-3.5" />
                {profile?.is_following ? '已关注' : '关注'}
              </Button>
            </div>
          )}
        </Card>
      )}

      {/* Stats card */}
      <Card>
        <p className="text-meta font-semibold text-text-muted uppercase tracking-wider mb-3">
          统计
        </p>
        <div className="space-y-2.5">
          <SidebarStatRow
            icon={<Eye className="w-3.5 h-3.5" />}
            label="浏览"
            value={post.view_count}
          />
          <SidebarStatRow
            icon={<Heart className="w-3.5 h-3.5" />}
            label="点赞"
            value={post.like_count}
          />
          <SidebarStatRow
            icon={<MessageSquare className="w-3.5 h-3.5" />}
            label="评论"
            value={post.comment_count}
          />
          <SidebarStatRow
            icon={<Bookmark className="w-3.5 h-3.5" />}
            label="收藏"
            value={post.collect_count}
          />
          <SidebarStatRow
            icon={<CalendarIcon className="w-3.5 h-3.5" />}
            label="发布于"
            value={formatDate(post.created_at)}
          />
        </div>
      </Card>

      {/* Related posts */}
      {relatedPosts.length > 0 && (
        <Card>
          <p className="text-meta font-semibold text-text-muted uppercase tracking-wider mb-3">
            相关帖子
          </p>
          <div className="space-y-3">
            {relatedPosts.slice(0, 5).map((rp) => (
              <Link
                key={rp.id}
                to={`/post/${rp.id}`}
                className="flex items-start gap-3 group"
              >
                <MessageSquare className="w-4 h-4 text-text-muted mt-0.5 shrink-0" />
                <div className="min-w-0">
                  <p className="text-body font-medium text-text-primary group-hover:text-amber transition-colors line-clamp-2">
                    {rp.title}
                  </p>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="text-caption text-text-muted font-mono">
                      {timeAgo(rp.created_at)}
                    </span>
                    <span className="flex items-center gap-0.5 text-caption text-text-muted">
                      <Heart className="w-3 h-3" />
                      {rp.like_count}
                    </span>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}

import { useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Heart, Bookmark, MessageSquare, Eye, Gamepad2, Users, Edit } from 'lucide-react'
import { userApi } from '@/api/user'
import { Avatar } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { CalendarIcon } from '@/components/shared/CalendarIcon'
import { formatDate } from '@/lib/time'
import { toast } from '@/stores/toastStore'
import type { DevLogDetail, UserInfo } from '@/types/api'

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

export function DevlogSidebar({
  log,
  currentUser,
}: {
  log: DevLogDetail
  currentUser: UserInfo | null
}) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const username = log.author?.username

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

  return (
    <div className="space-y-4">
      {/* Project info card */}
      {log.project_name && (
        <Card>
          <p className="text-meta font-semibold text-text-muted uppercase tracking-wider mb-3">
            所属项目
          </p>
          <Link
            to={`/p/${log.project_slug}`}
            className="flex items-center gap-3 group"
          >
            <div className="w-11 h-11 rounded-lg bg-black/30 border border-white/[0.04] overflow-hidden shrink-0 flex items-center justify-center">
              {log.project_cover_url ? (
                <img
                  src={log.project_cover_url}
                  alt=""
                  className="w-full h-full object-cover"
                />
              ) : (
                <Gamepad2 className="w-5 h-5 text-text-muted" />
              )}
            </div>
            <div className="min-w-0">
              <p className="text-h4 text-text-primary group-hover:text-amber transition-colors truncate">
                {log.project_name}
              </p>
              <p className="text-caption text-text-muted font-mono">
                @{log.project_slug}
              </p>
            </div>
          </Link>
        </Card>
      )}

      {/* Author card */}
      {log.author && (
        <Card>
          <p className="text-meta font-semibold text-text-muted uppercase tracking-wider mb-3">
            作者
          </p>
          <div className="flex flex-col items-center text-center">
            <Link to={`/u/${log.author.username || log.author.id}`}>
              <Avatar
                src={log.author.avatar_url}
                name={log.author.nickname}
                size="lg"
              />
            </Link>
            <Link
              to={`/u/${log.author.username || log.author.id}`}
              className="mt-3 text-h3 text-text-primary hover:text-amber transition-colors"
            >
              {log.author.nickname || log.author.username}
            </Link>
            <p className="text-small text-text-muted font-mono mt-0.5">
              @{log.author.username}
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
            </div>
          )}

          {/* Follow button */}
          {currentUser && log.author.id !== currentUser.id && (
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
          {currentUser && log.author.id === currentUser.id && (
            <div className="mt-3">
              <Button
                variant="outline"
                size="sm"
                className="w-full"
                onClick={() => navigate('/settings')}
              >
                <Edit className="w-3.5 h-3.5" />
                编辑资料
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
            value={log.view_count}
          />
          <SidebarStatRow
            icon={<Heart className="w-3.5 h-3.5" />}
            label="点赞"
            value={log.like_count}
          />
          <SidebarStatRow
            icon={<MessageSquare className="w-3.5 h-3.5" />}
            label="评论"
            value={log.comment_count}
          />
          <SidebarStatRow
            icon={<Bookmark className="w-3.5 h-3.5" />}
            label="收藏"
            value={log.collect_count}
          />
          <SidebarStatRow
            icon={<CalendarIcon className="w-3.5 h-3.5" />}
            label="发布于"
            value={formatDate(log.created_at)}
          />
          {log.updated_at !== log.created_at && (
            <SidebarStatRow
              icon={<CalendarIcon className="w-3.5 h-3.5" />}
              label="更新于"
              value={formatDate(log.updated_at)}
            />
          )}
        </div>
      </Card>
    </div>
  )
}

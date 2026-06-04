import { useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  MapPin,
  ExternalLink,
  Gamepad2,
  ScrollText,
  MessageSquare,
  Star,
  Users,
  UserPlus,
  Edit3,
  ChevronRight,
} from 'lucide-react'
import { userApi } from '@/api/user'
import { projectApi } from '@/api/project'
import { communityApi } from '@/api/community'
import { useAuthStore } from '@/stores/authStore'
import { Avatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/ui/empty-state'
import { Tabs } from '@/components/ui/tabs'
import { Pagination } from '@/components/ui/pagination'
import { PostCard } from '@/components/shared/PostCard'
import { toast } from '@/stores/toastStore'
import { PORTFOLIO_TYPE_LABELS } from '@/lib/constants'
import { ProjectCard, EndorsementCard, SkillCategoryGroup, FollowListModal, ProfileSkeleton } from '@/components/profile'
import { DevLogCard } from '@/components/shared/DevLogCard'
import type { UserProfileResponse } from '@/types/api'

const PAGE_SIZE_GRID = 9
const PAGE_SIZE_LIST = 10

// ============================================================
// Profile Page
// ============================================================

export default function ProfilePage() {
  const { id: username } = useParams<{ id: string }>()
  const me = useAuthStore((s) => s.user)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [tab, setTab] = useState('overview')
  const [projectPage, setProjectPage] = useState(1)
  const [logPage, setLogPage] = useState(1)
  const [postPage, setPostPage] = useState(1)
  const [endorsePage, setEndorsePage] = useState(1)
  const [favPage, setFavPage] = useState(1)
  const [followListType, setFollowListType] = useState<'followers' | 'following' | null>(null)

  const profileQuery = useQuery({
    queryKey: ['profile', username],
    queryFn: () => userApi.getProfile(username!),
    enabled: !!username,
  })

  const profile = profileQuery.data
  const isSelf = !!(me && profile && me.id === profile.id)

  const handleTabChange = useCallback((value: string) => {
    setTab(value)
    setProjectPage(1)
    setLogPage(1)
    setPostPage(1)
    setEndorsePage(1)
    setFavPage(1)
  }, [])

  const followMutation = useMutation({
    mutationFn: async () => {
      const res = profile!.is_following
        ? await userApi.unfollow(profile!.id)
        : await userApi.follow(profile!.id)
      return res
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ['profile', username] })
      const previous = queryClient.getQueryData(['profile', username])
      queryClient.setQueryData(['profile', username], (old: any) => {
        if (!old) return old
        return {
          ...old,
          is_following: !old.is_following,
          followers_count: old.followers_count + (old.is_following ? -1 : 1),
        }
      })
      return { previous }
    },
    onError: (_err, _vars, context) => {
      queryClient.setQueryData(['profile', username], (context as any)?.previous)
      toast.error('操作失败，请重试')
    },
    onSuccess: () => {
      toast.success(profile?.is_following ? '已取消关注' : '关注成功')
      queryClient.invalidateQueries({ queryKey: ['profile', username] })
      // Also refresh follow list modal if open
      queryClient.invalidateQueries({ queryKey: ['follow-list', username] })
    },
  })

  const collectedPostsQuery = useQuery({
    queryKey: ['collected-posts', username, favPage],
    queryFn: () => userApi.getMyCollectedPosts(favPage, PAGE_SIZE_LIST),
    enabled: isSelf && tab === 'favorites',
  })

  const userId = profile?.id

  const projectsQuery = useQuery({
    queryKey: ['user-projects', userId, projectPage],
    queryFn: () => projectApi.getUserProjects(userId!, projectPage, PAGE_SIZE_GRID),
    enabled: tab === 'projects' && !!userId,
  })

  const logsQuery = useQuery({
    queryKey: ['user-logs', userId, logPage],
    queryFn: () => userApi.getUserLogs(userId!, logPage, PAGE_SIZE_LIST),
    enabled: tab === 'logs' && !!userId,
  })

  const postsQuery = useQuery({
    queryKey: ['user-posts', userId, postPage],
    queryFn: () => communityApi.getUserPosts(userId!, postPage, PAGE_SIZE_LIST),
    enabled: tab === 'posts' && !!userId,
  })

  const endorsementsQuery = useQuery({
    queryKey: ['user-endorsements', userId, endorsePage],
    queryFn: () => userApi.getUserEndorsements(userId!, endorsePage, PAGE_SIZE_LIST),
    enabled: tab === 'endorsements' && !!userId,
  })

  const tabs = [
    { value: 'overview', label: '概览' },
    { value: 'projects', label: '项目', count: projectsQuery.data?.total ?? profile?.project_count ?? 0 },
    { value: 'logs', label: '日志', count: logsQuery.data?.total ?? profile?.log_count ?? 0 },
    { value: 'posts', label: '帖子', count: postsQuery.data?.total ?? profile?.post_count ?? 0 },
    { value: 'endorsements', label: '评价', count: endorsementsQuery.data?.total ?? 0 },
    ...(isSelf ? [{ value: 'favorites' as const, label: '收藏' }] : []),
  ]

  // Loading state
  if (profileQuery.isLoading) {
    return <ProfileSkeleton />
  }

  // Error / Not Found
  if (profileQuery.isError || !profile) {
    return (
      <div className="max-w-[960px] mx-auto px-6 py-8">
        <div className="text-center py-20">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Users className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">用户不存在</p>
          <p className="text-[13px] text-text-muted mb-6">无法找到该用户，可能已被删除或链接不正确</p>
          <Button variant="secondary" size="sm" onClick={() => navigate(isSelf ? '/feed' : '/projects')}>
            返回首页
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-[960px] mx-auto px-6 py-8 space-y-6">
      {/* ============================================================ */}
      {/* Header Section */}
      {/* ============================================================ */}
      <div className="bg-surface-card border border-white/[0.04] rounded-xl p-6">
        <div className="flex flex-col sm:flex-row gap-5">
          {/* Avatar */}
          <div className="flex justify-center sm:justify-start">
            <Avatar src={profile.avatar_url} name={profile.nickname} size="xl" />
          </div>

          {/* Info */}
          <div className="flex-1 min-w-0">
            <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
              <div className="text-center sm:text-left">
                <h1 className="text-[22px] font-bold text-text-primary">{profile.nickname}</h1>
                <p className="text-[13px] text-text-muted font-mono mt-0.5">@{profile.username}</p>
              </div>

              {/* Action button */}
              <div className="flex justify-center sm:justify-end shrink-0">
                {isSelf ? (
                  <Button variant="outline" size="sm" onClick={() => navigate('/settings')}>
                    <Edit3 className="h-3.5 w-3.5" />
                    编辑资料
                  </Button>
                ) : (
                  <Button
                    variant={profile.is_following ? 'outline' : 'primary'}
                    size="sm"
                    loading={followMutation.isPending}
                    onClick={() => followMutation.mutate()}
                  >
                    <UserPlus className="h-3.5 w-3.5" />
                    {profile.is_following ? '已关注' : '关注'}
                  </Button>
                )}
              </div>
            </div>

            {/* Bio */}
            {profile.bio && (
              <p className="text-[13px] text-text-secondary leading-relaxed mt-3 text-center sm:text-left">
                {profile.bio}
              </p>
            )}

            {/* Location */}
            {profile.location && (
              <p className="flex items-center justify-center sm:justify-start gap-1 mt-2 text-[12px] text-text-muted">
                <MapPin className="w-3 h-3" />
                {profile.location}
              </p>
            )}

            {/* Badges */}
            {profile.badges && profile.badges.length > 0 && (
              <div className="flex flex-wrap justify-center sm:justify-start gap-1.5 mt-3">
                {profile.badges.map((badge) => (
                  <Badge key={badge} variant="info" size="sm">{badge}</Badge>
                ))}
              </div>
            )}

            {/* Stats */}
            <div className="flex justify-center sm:justify-start gap-6 mt-4 pt-4 border-t border-white/[0.04]">
              <div className="text-center sm:text-left">
                <p className="text-[16px] font-bold text-text-primary font-mono">
                  {profile.project_count}
                </p>
                <p className="text-[10px] text-text-muted mt-0.5">项目</p>
              </div>
              <button
                onClick={() => setFollowListType('followers')}
                className="text-center sm:text-left hover:opacity-80 transition-opacity"
              >
                <p className="text-[16px] font-bold text-text-primary font-mono">
                  {profile.followers_count}
                </p>
                <p className="text-[10px] text-text-muted mt-0.5">粉丝</p>
              </button>
              <button
                onClick={() => setFollowListType('following')}
                className="text-center sm:text-left hover:opacity-80 transition-opacity"
              >
                <p className="text-[16px] font-bold text-text-primary font-mono">
                  {profile.following_count}
                </p>
                <p className="text-[10px] text-text-muted mt-0.5">关注</p>
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Follow List Modal */}
      <FollowListModal
        username={username!}
        type={followListType || 'followers'}
        open={!!followListType}
        onClose={() => setFollowListType(null)}
      />

      {/* ============================================================ */}
      {/* Tabs */}
      {/* ============================================================ */}
      <Tabs tabs={tabs} value={tab} onChange={handleTabChange} />

      {/* ============================================================ */}
      {/* Tab Content */}
      {/* ============================================================ */}

      {/* --- Overview --- */}
      {tab === 'overview' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Left Column */}
          <div className="space-y-6">
            {/* Skills */}
            {profile.skills.length > 0 && (
              <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
                <h3 className="text-[14px] font-semibold text-text-primary mb-4">技能</h3>
                {groupSkillsByCategory(profile.skills).map((group) => (
                  <SkillCategoryGroup
                    key={group.category}
                    category={group.category}
                    skills={group.skills}
                  />
                ))}
              </div>
            )}

            {/* Portfolio */}
            {profile.portfolio.length > 0 && (
              <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
                <h3 className="text-[14px] font-semibold text-text-primary mb-4">作品集</h3>
                <div className="space-y-3">
                  {profile.portfolio.map((item) => (
                    <div key={item.id} className="flex items-start gap-3">
                      <div className="w-8 h-8 rounded-lg bg-amber/10 flex items-center justify-center shrink-0">
                        {item.image_url ? (
                          <img src={item.image_url} alt="" className="w-full h-full object-cover rounded-lg" />
                        ) : (
                          <ExternalLink className="w-4 h-4 text-amber" />
                        )}
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <span className="text-[13px] font-medium text-text-primary">{item.name}</span>
                          <Badge variant="default" size="sm">{PORTFOLIO_TYPE_LABELS[item.type] ?? item.type}</Badge>
                        </div>
                        {item.description && (
                          <p className="text-[13px] text-text-secondary line-clamp-2 mt-0.5">{item.description}</p>
                        )}
                        {item.link && (
                          <a
                            href={item.link}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 mt-1 text-[11px] font-mono text-amber hover:underline"
                          >
                            <ExternalLink className="w-3 h-3" />
                            {item.link}
                          </a>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Right Column */}
          <div className="space-y-6">
            {/* Empty state */}
            {profile.skills.length === 0 &&
              profile.portfolio.length === 0 && (
                <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
                  <EmptyState
                    icon={<Users className="w-6 h-6 text-text-muted" />}
                    title="这个用户还没有任何内容"
                    description="等待他们开始创作吧"
                  />
                </div>
              )}
          </div>
        </div>
      )}

      {/* --- Projects Tab --- */}
      {tab === 'projects' && projectsQuery.data && (
        <div>
          {projectsQuery.data.list.length > 0 ? (
            <>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {projectsQuery.data.list.map((project) => (
                  <ProjectCard key={project.id} project={project} />
                ))}
              </div>
              {projectsQuery.data.pages > 1 && (
                <Pagination
                  page={projectsQuery.data.page}
                  pages={projectsQuery.data.pages}
                  onChange={setProjectPage}
                />
              )}
            </>
          ) : (
            <EmptyState
              icon={<Gamepad2 className="w-6 h-6 text-text-muted" />}
              title="暂无项目"
              description="该用户还没有参与任何项目"
            />
          )}
        </div>
      )}
      {tab === 'projects' && projectsQuery.isLoading && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-40 w-full rounded-xl" />
          ))}
        </div>
      )}
      {tab === 'projects' && projectsQuery.isError && (
        <div className="text-center py-16">
          <p className="text-h4 text-text-secondary mb-1">加载失败</p>
          <Button variant="secondary" size="sm" onClick={() => projectsQuery.refetch()}>重试</Button>
        </div>
      )}

      {/* --- Logs Tab --- */}
      {tab === 'logs' && logsQuery.data && (
        <div>
          {logsQuery.data.list.length > 0 ? (
            <>
              <div className="space-y-3">
                {logsQuery.data.list.map((log) => (
                  <DevLogCard key={log.id} log={log} />
                ))}
              </div>
              {logsQuery.data.pages > 1 && (
                <Pagination
                  page={logsQuery.data.page}
                  pages={logsQuery.data.pages}
                  onChange={setLogPage}
                />
              )}
            </>
          ) : (
            <EmptyState
              icon={<ScrollText className="w-6 h-6 text-text-muted" />}
              title="暂无日志"
              description="该用户还没有发表开发日志"
            />
          )}
        </div>
      )}
      {tab === 'logs' && logsQuery.isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-28 w-full rounded-xl" />
          ))}
        </div>
      )}
      {tab === 'logs' && logsQuery.isError && (
        <div className="text-center py-16">
          <p className="text-h4 text-text-secondary mb-1">加载失败</p>
          <Button variant="secondary" size="sm" onClick={() => logsQuery.refetch()}>重试</Button>
        </div>
      )}

      {/* --- Posts Tab --- */}
      {tab === 'posts' && postsQuery.data && (
        <div>
          {postsQuery.data.list.length > 0 ? (
            <>
              <div className="space-y-3">
                {postsQuery.data.list.map((post) => (
                  <PostCard key={post.id} post={post} />
                ))}
              </div>
              {postsQuery.data.pages > 1 && (
                <Pagination
                  page={postsQuery.data.page}
                  pages={postsQuery.data.pages}
                  onChange={setPostPage}
                />
              )}
            </>
          ) : (
            <EmptyState
              icon={<MessageSquare className="w-6 h-6 text-text-muted" />}
              title="暂无帖子"
              description="该用户还没有发布社区帖子"
            />
          )}
        </div>
      )}
      {tab === 'posts' && postsQuery.isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-28 w-full rounded-xl" />
          ))}
        </div>
      )}
      {tab === 'posts' && postsQuery.isError && (
        <div className="text-center py-16">
          <p className="text-h4 text-text-secondary mb-1">加载失败</p>
          <Button variant="secondary" size="sm" onClick={() => postsQuery.refetch()}>重试</Button>
        </div>
      )}

      {/* --- Endorsements Tab --- */}
      {tab === 'endorsements' && endorsementsQuery.data && (
        <div>
          {endorsementsQuery.data.list.length > 0 ? (
            <>
              <div className="space-y-3">
                {endorsementsQuery.data.list.map((review) => (
                  <EndorsementCard key={review.id} review={review} />
                ))}
              </div>
              {endorsementsQuery.data.pages > 1 && (
                <Pagination
                  page={endorsementsQuery.data.page}
                  pages={endorsementsQuery.data.pages}
                  onChange={setEndorsePage}
                />
              )}
            </>
          ) : (
            <EmptyState
              icon={<Star className="w-6 h-6 text-text-muted" />}
              title="暂无评价"
              description="该用户还没有收到合作评价"
            />
          )}
        </div>
      )}
      {tab === 'endorsements' && endorsementsQuery.isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 w-full rounded-xl" />
          ))}
        </div>
      )}
      {tab === 'endorsements' && endorsementsQuery.isError && (
        <div className="text-center py-16">
          <p className="text-h4 text-text-secondary mb-1">加载失败</p>
          <Button variant="secondary" size="sm" onClick={() => endorsementsQuery.refetch()}>重试</Button>
        </div>
      )}

      {/* --- Favorites Tab (self only) --- */}
      {tab === 'favorites' && (
        <div>
          {collectedPostsQuery.isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-24 w-full rounded-xl" />
              ))}
            </div>
          ) : collectedPostsQuery.isError ? (
            <EmptyState
              icon={<MessageSquare className="w-6 h-6 text-text-muted" />}
              title="加载失败"
              action={{
                label: '重试',
                onClick: () => collectedPostsQuery.refetch(),
              }}
            />
          ) : collectedPostsQuery.data?.list.length ? (
            <>
              <div className="space-y-3">
                {collectedPostsQuery.data.list.map((post) => (
                  <PostCard key={post.id} post={post} />
                ))}
              </div>
              {collectedPostsQuery.data.pages > 1 && (
                <Pagination
                  page={collectedPostsQuery.data.page}
                  pages={collectedPostsQuery.data.pages}
                  onChange={setFavPage}
                />
              )}
            </>
          ) : (
            <EmptyState
              icon={<Star className="w-6 h-6 text-text-muted" />}
              title="还没有收藏"
              description="收藏的帖子会出现在这里"
            />
          )}
        </div>
      )}
    </div>
  )
}

// ============================================================
// Utility
// ============================================================

function groupSkillsByCategory(skills: UserProfileResponse['skills']) {
  const groups = new Map<string, typeof skills>()
  for (const skill of skills) {
    const cat = skill.category
    if (!groups.has(cat)) groups.set(cat, [])
    groups.get(cat)!.push(skill)
  }
  return Array.from(groups.entries()).map(([category, skills]) => ({ category, skills }))
}

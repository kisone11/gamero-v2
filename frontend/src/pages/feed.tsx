import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { MessageSquare, Gamepad2, ScrollText, TrendingUp, Users, Zap } from 'lucide-react'
import { discoverApi, type FeedItem } from '@/api/discover'
import { announcementApi } from '@/api/announcement'
import { projectApi } from '@/api/project'
import { userApi } from '@/api/user'
import { useAuthStore } from '@/stores/authStore'
import { Avatar } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { PostCard } from '@/components/shared/PostCard'
import { DevLogCard } from '@/components/shared/DevLogCard'
import { ProjectCard } from '@/components/profile/ProjectCard'

const QUICK_LINKS = [
  { href: '/projects', icon: Gamepad2, label: '发现项目', desc: '探索游戏作品' },
  { href: '/devlogs', icon: ScrollText, label: '开发日志', desc: '记录开发过程' },
  { href: '/community', icon: MessageSquare, label: '社区讨论', desc: '交流与分享' },
  { href: '/recruit', icon: Users, label: '寻找队友', desc: '组建团队' },
]

// ============================================================
// FeedPage
// ============================================================

export default function FeedPage() {
  const me = useAuthStore(s => s.user)
  const [view, setView] = useState<'feed' | 'my-projects'>('feed')

  const feedQuery = useInfiniteQuery({
    queryKey: ['feed'],
    queryFn: async ({ pageParam }) => {
      const data = await discoverApi.getFeed(pageParam as number | undefined)
      return data
    },
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (last) =>
      last?.next_cursor && last.next_cursor > 0 ? last.next_cursor : undefined,
  })

  const { data: hotProjects } = useQuery({
    queryKey: ['hot-projects'],
    queryFn: () => discoverApi.getRecommendedProjects(1, 5),
    staleTime: 60000,
  })

  const { data: announcements } = useQuery({
    queryKey: ['public-announcements'],
    queryFn: () => announcementApi.listPublic(3),
    staleTime: 60000,
  })

  const { data: myStats } = useQuery({
    queryKey: ['my-stats'],
    queryFn: () => userApi.getMyStats(),
    enabled: !!me,
    staleTime: 60000,
  })

  const { data: myProjects, isLoading: myProjectsLoading, isError: myProjectsError, refetch: refetchMyProjects } = useQuery({
    queryKey: ['feed-my-projects', me?.id],
    queryFn: () => projectApi.list({ page: 1, page_size: 50, participant_id: me!.id }),
    enabled: !!me && view === 'my-projects',
  })

  const items = feedQuery.data?.pages.flatMap(p => p.items) ?? []
  const myProjectItems = myProjects?.list ?? []
  const isLoading = feedQuery.isLoading

  return (
    <div className="max-w-[1280px] mx-auto px-6 py-8">
      <div className="flex gap-8">
        {/* Left Sidebar */}
        <aside className="hidden lg:block w-[240px] shrink-0">
          <div className="sticky top-[88px] space-y-4">
            {me && (
              <div className="bg-surface-card border border-white/[0.04] rounded-xl p-4">
                <Link to={`/u/${me.username || me.id}`} className="flex items-center gap-3 group">
                  <Avatar src={me.avatar_url} name={me.nickname} size="md" />
                  <div>
                    <p className="text-[14px] font-semibold text-text-primary group-hover:text-amber transition-colors">{me.nickname}</p>
                    <p className="text-[11px] text-text-muted font-mono">@{me.username}</p>
                  </div>
                </Link>
                {myStats && (
                  <div className="mt-3 pt-3 border-t border-white/[0.04] space-y-1.5">
                    <StatRow label="项目" value={myStats.total_projects} />
                    <StatRow label="日志" value={myStats.total_logs} />
                    <StatRow label="关注者" value={myStats.total_followers} />
                  </div>
                )}
              </div>
            )}
            <div className="space-y-1">
              {QUICK_LINKS.map(link => (
                <Link key={link.href} to={link.href}
                  className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-white/[0.03] transition-colors group">
                  <link.icon className="w-4 h-4 shrink-0 group-hover:text-amber transition-colors" />
                  <div>
                    <p className="text-[13px] font-medium">{link.label}</p>
                    <p className="text-[11px] text-text-muted">{link.desc}</p>
                  </div>
                </Link>
              ))}
              {me && (
                <button
                  type="button"
                  onClick={() => setView('my-projects')}
                  className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left text-text-muted hover:text-text-primary hover:bg-white/[0.03] transition-colors group"
                >
                  <Gamepad2 className="w-4 h-4 shrink-0 group-hover:text-amber transition-colors" />
                  <div>
                    <p className="text-[13px] font-medium">我的项目</p>
                    <p className="text-[11px] text-text-muted">参与和创建的项目</p>
                  </div>
                </button>
              )}
            </div>
            <p className="text-[10px] text-text-muted text-center px-4">&copy; 2026 Gamero &middot; 游戏开发者社区</p>
          </div>
        </aside>

        {/* Main Feed */}
        <main className="flex-1 min-w-0">
          {view === 'my-projects' ? (
            <div className="space-y-3">
              <div className="bg-surface-card border border-white/[0.04] rounded-xl p-4 flex items-center justify-between">
                <div>
                  <h2 className="text-[16px] font-semibold text-text-primary">我的项目</h2>
                  <p className="text-[12px] text-text-muted mt-1">只显示你参与或创建的项目</p>
                </div>
                <Button variant="secondary" size="sm" onClick={() => setView('feed')}>返回动态</Button>
              </div>
              {myProjectsLoading ? (
                <div className="space-y-3">{Array.from({ length: 4 }).map((_, i) => <FeedSkeleton key={i} />)}</div>
              ) : myProjectsError ? (
                <div className="py-20 flex flex-col items-center justify-center text-center">
                  <Zap className="w-7 h-7 text-text-muted mb-4" />
                  <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
                  <p className="text-[13px] text-text-muted mb-6">无法加载我的项目，请重试</p>
                  <Button variant="secondary" size="sm" onClick={() => refetchMyProjects()}>重试</Button>
                </div>
              ) : myProjectItems.length === 0 ? (
                <div className="py-20 flex flex-col items-center justify-center text-center">
                  <Gamepad2 className="w-7 h-7 text-text-muted mb-4" />
                  <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无项目</p>
                  <p className="text-[13px] text-text-muted mb-6">你还没有参与或创建项目</p>
                  <Link to="/projects/new"><Button variant="primary" size="sm">创建项目</Button></Link>
                </div>
              ) : (
                myProjectItems.map((project) => <ProjectCard key={project.id} project={project} />)
              )}
            </div>
          ) : isLoading ? (
            <div className="space-y-3">{Array.from({ length: 4 }).map((_, i) => <FeedSkeleton key={i} />)}</div>
          ) : feedQuery.isError && items.length === 0 ? (
            <div className="py-20 flex flex-col items-center justify-center text-center">
              <Zap className="w-7 h-7 text-text-muted mb-4" />
              <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
              <p className="text-[13px] text-text-muted mb-6">无法加载动态，请重试</p>
              <Button variant="secondary" size="sm" onClick={() => feedQuery.refetch()}>重试</Button>
            </div>
          ) : items.length === 0 ? (
            <div className="py-20 flex flex-col items-center justify-center text-center">
              <TrendingUp className="w-7 h-7 text-text-muted mb-4" />
              <p className="text-[15px] font-semibold text-text-secondary mb-1">关注一些创作者和项目</p>
              <p className="text-[13px] text-text-muted mb-6">他们的动态会出现在这里</p>
              <Link to="/projects"><Button variant="primary" size="sm">发现项目</Button></Link>
            </div>
          ) : (
            <div className="space-y-3">
              {items.map((item) => {
                switch (item.type) {
                  case 'post': return item.post ? <PostCard key={item.id} post={item.post} /> : null
                  case 'devlog': case 'release': return item.devlog ? <DevLogCard key={item.id} log={item.devlog} /> : null
                  case 'project': return item.project ? <ProjectCard key={item.id} project={item.project} /> : null
                  default: return null
                }
              })}
              {feedQuery.isError && items.length > 0 && (
                <div className="text-center py-4">
                  <p className="text-[13px] text-text-muted mb-2">加载更多失败</p>
                  <Button variant="secondary" size="sm" onClick={() => feedQuery.refetch()}>重试</Button>
                </div>
              )}
              {feedQuery.hasNextPage && (
                <div className="flex justify-center pt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    loading={feedQuery.isFetchingNextPage}
                    onClick={() => feedQuery.fetchNextPage()}
                  >
                    加载更多
                  </Button>
                </div>
              )}
            </div>
          )}
        </main>

        {/* Right Sidebar */}
        <aside className="hidden xl:block w-[260px] shrink-0">
          <div className="sticky top-[88px] space-y-4">
            {announcements && announcements.length > 0 && (
              <div className="bg-surface-card border border-amber/10 rounded-xl p-4">
                <h3 className="text-[13px] font-semibold text-text-primary mb-3 flex items-center gap-1.5">
                  <Zap className="w-4 h-4 text-amber" />站内公告
                </h3>
                <div className="space-y-3">
                  {announcements.map((item) => (
                    <div key={item.id} className="rounded-lg border border-white/[0.04] p-3 bg-white/[0.02]">
                      <div className="flex items-center gap-2 mb-1">
                        {item.is_pinned && <span className="text-[10px] text-amber font-semibold">置顶</span>}
                        <span className="text-[10px] text-text-muted font-mono">{item.level}</span>
                      </div>
                      <p className="text-[12px] font-semibold text-text-primary line-clamp-1">{item.title}</p>
                      <p className="text-[11px] text-text-muted mt-1 line-clamp-2">{item.content}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}
            {hotProjects && (hotProjects as any).list?.length > 0 && (
              <div className="bg-surface-card border border-white/[0.04] rounded-xl p-4">
                <h3 className="text-[13px] font-semibold text-text-primary mb-3 flex items-center gap-1.5">
                  <TrendingUp className="w-4 h-4 text-amber" />热门项目
                </h3>
                <div className="space-y-3">
                  {(hotProjects as any).list.slice(0, 5).map((p: any) => (
                    <Link key={p.id} to={`/p/${p.slug || p.id}`} className="flex items-center gap-3 group">
                      <div className="w-8 h-8 rounded-lg bg-surface-deep overflow-hidden shrink-0 border border-white/[0.04]">
                        {p.cover_url ? <img src={p.cover_url} alt="" className="w-full h-full object-cover" />
                          : <div className="w-full h-full flex items-center justify-center"><Gamepad2 className="w-4 h-4 text-text-muted/30" /></div>}
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="text-[12px] font-medium text-text-primary group-hover:text-amber transition-colors truncate">{p.name}</p>
                        <p className="text-[10px] text-text-muted font-mono">{p.follower_count ?? 0} 关注</p>
                      </div>
                    </Link>
                  ))}
                </div>
              </div>
            )}
            <p className="text-[10px] text-text-muted text-center px-4">&copy; 2026 Gamero &middot; 游戏开发者社区</p>
          </div>
        </aside>
      </div>
    </div>
  )
}

function FeedSkeleton() {
  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3">
      <div className="flex items-center gap-3">
        <Skeleton className="w-10 h-10 rounded-xl shrink-0" />
        <div className="space-y-1.5 flex-1">
          <Skeleton className="h-3 w-24 rounded" />
          <Skeleton className="h-2.5 w-16 rounded" />
        </div>
      </div>
      <Skeleton className="h-4 w-3/4 rounded" />
      <Skeleton className="h-4 w-full rounded" />
    </div>
  )
}

function StatRow({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex items-center justify-between text-[12px]">
      <span className="text-text-muted">{label}</span>
      <span className="text-text-primary font-mono">{value?.toLocaleString() ?? '0'}</span>
    </div>
  )
}

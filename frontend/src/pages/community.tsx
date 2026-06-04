import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { MessageSquare, MessageSquareText, Plus } from 'lucide-react'
import { communityApi } from '@/api/community'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Pagination } from '@/components/ui/pagination'
import { cn } from '@/lib/utils'
import { PostCard } from '@/components/shared/PostCard'
import { PostCardSkeleton, CreatePostModal } from '@/components/community'

// ============================================================
// Community Page
// ============================================================

export default function CommunityPage() {
  const user = useAuthStore((s) => s.user)

  const [activeTopicId, setActiveTopicId] = useState<number | undefined>(undefined)
  const [sort, setSort] = useState<'latest' | 'hot'>('latest')
  const [page, setPage] = useState(1)
  const [createOpen, setCreateOpen] = useState(false)

  const topicsQuery = useQuery({
    queryKey: ['community-topics'],
    queryFn: () => communityApi.listTopics(),
  })

  const topics = topicsQuery.data ?? []

  const postsQuery = useQuery({
    queryKey: ['community-posts', { topic_id: activeTopicId, sort, page }],
    queryFn: () =>
      communityApi.list({
        page,
        page_size: 20,
        topic_id: activeTopicId,
        sort,
      }),
  })

  const posts = postsQuery.data?.list ?? []
  const pages = postsQuery.data?.pages ?? 1

  const handleTopicClick = (topicId?: number) => {
    setActiveTopicId((prev) => (prev === topicId ? undefined : topicId))
    setPage(1)
  }

  const handleSortChange = (newSort: 'latest' | 'hot') => {
    setSort(newSort)
    setPage(1)
  }

  return (
    <div className="max-w-[1280px] mx-auto px-6 py-8">
      <div className="flex gap-8">
        {/* Left sidebar — 话题列表 */}
        <aside className="hidden lg:block w-[240px] shrink-0">
          <div className="sticky top-[88px] space-y-4">
            <p className="text-[10px] font-semibold text-text-muted uppercase tracking-wider px-3">
              话题分类
            </p>
            <nav className="space-y-0.5">
              <button
                onClick={() => handleTopicClick(undefined)}
                className={cn(
                  'w-full text-left px-3 py-2 rounded-lg text-[13px] font-medium transition-colors',
                  activeTopicId === undefined
                    ? 'bg-white/[0.06] text-text-primary'
                    : 'text-text-muted hover:text-text-secondary hover:bg-white/[0.03]'
                )}
              >
                全部
              </button>
              {topicsQuery.isLoading &&
                Array.from({ length: 5 }).map((_, i) => (
                  <Skeleton key={i} className="h-8 w-full rounded-lg" />
                ))}
              {topicsQuery.isError && (
                <p className="text-[12px] text-danger px-3">话题加载失败</p>
              )}
              {topics.map((topic) => (
                <button
                  key={topic.id}
                  onClick={() => handleTopicClick(topic.id)}
                  className={cn(
                    'w-full text-left px-3 py-2 rounded-lg text-[13px] font-medium transition-colors flex items-center justify-between',
                    activeTopicId === topic.id
                      ? 'bg-white/[0.06] text-text-primary'
                      : 'text-text-muted hover:text-text-secondary hover:bg-white/[0.03]'
                  )}
                >
                  <span>{topic.name}</span>
                  <span className="text-[11px] font-mono text-text-muted">{topic.post_count}</span>
                </button>
              ))}
            </nav>
          </div>
        </aside>

        {/* Main content — 帖子列表 */}
        <main className="flex-1 min-w-0 max-w-[680px]">
          <div className="flex items-center justify-between mb-6">
            <h1 className="text-[22px] font-bold text-text-primary">社区</h1>
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1">
                {([
                  { key: 'latest' as const, label: '最新' },
                  { key: 'hot' as const, label: '最热' },
                ] as const).map((t) => (
                  <button
                    key={t.key}
                    onClick={() => handleSortChange(t.key)}
                    className={cn(
                      'px-4 py-2 text-[14px] font-medium rounded-lg transition-all duration-200',
                      sort === t.key
                        ? 'bg-white/[0.06] text-text-primary'
                        : 'text-text-muted hover:text-text-secondary hover:bg-white/[0.02]'
                    )}
                  >
                    {t.label}
                  </button>
                ))}
              </div>
              {user && (
                <Button size="sm" onClick={() => setCreateOpen(true)}>
                  <Plus className="w-3.5 h-3.5" />
                  发帖
                </Button>
              )}
            </div>
          </div>

          {/* Mobile topic filter */}
          <div className="lg:hidden mb-4">
            <select
              value={activeTopicId ?? ''}
              onChange={(e) => setActiveTopicId(e.target.value ? Number(e.target.value) : undefined)}
              className="w-full h-10 px-3 bg-surface-card border border-white/[0.04] rounded-xl text-[13px] text-text-primary focus:outline-none focus:border-amber"
            >
              <option value="">全部话题</option>
              {topics.map((t: any) => (
                <option key={t.id} value={t.id}>
                  {t.icon_key ? `${t.icon_key} ` : ''}{t.name}
                </option>
              ))}
            </select>
          </div>

          {/* Loading */}
          {postsQuery.isLoading && (
            <div className="space-y-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <PostCardSkeleton key={i} />
              ))}
            </div>
          )}

          {/* Error */}
          {postsQuery.isError && (
            <div className="text-center py-20">
              <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
                <MessageSquareText className="w-7 h-7 text-text-muted" />
              </div>
              <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
              <p className="text-[13px] text-text-muted mb-6">无法加载帖子列表，请检查网络后重试</p>
              <Button variant="secondary" size="sm" onClick={() => postsQuery.refetch()}>
                重新加载
              </Button>
            </div>
          )}

          {/* Empty */}
          {!postsQuery.isLoading && !postsQuery.isError && posts.length === 0 && (
            <div className="text-center py-20">
              <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
                <MessageSquare className="w-7 h-7 text-text-muted" />
              </div>
              <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无帖子</p>
              <p className="text-[13px] text-text-muted mb-6">
                {activeTopicId ? '该话题下还没有帖子' : '还没有人发帖，来发第一条吧'}
              </p>
              {user && !activeTopicId && (
                <Button variant="secondary" size="sm" onClick={() => setCreateOpen(true)}>
                  发帖
                </Button>
              )}
            </div>
          )}

          {/* Post list */}
          {!postsQuery.isLoading && !postsQuery.isError && posts.length > 0 && (
            <div className="space-y-3">
              {posts.map((post) => (
                <PostCard key={post.id} post={post} showAuthor showPinned />
              ))}
              <Pagination page={page} pages={pages} onChange={setPage} />
            </div>
          )}
        </main>
      </div>

      <CreatePostModal
        open={createOpen}
        onOpenChange={setCreateOpen}
        topics={topics}
      />
    </div>
  )
}

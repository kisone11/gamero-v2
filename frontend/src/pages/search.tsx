import { useState, useEffect, useRef } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Search as SearchIcon, Users, Gamepad2, ScrollText, MessageSquare, Zap, Heart, Eye, FileText, User } from 'lucide-react'
import { discoverApi } from '@/api/discover'
import { userApi } from '@/api/user'
import { Badge, Avatar, EmptyState, Pagination } from '@/components/ui'
import { ImageGallery } from '@/components/shared/ImageGallery'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { timeAgo } from '@/lib/time'
import type { ProjectListItem, DevLog, Post } from '@/types/api'
import type { ProjectGenre } from '@/types/enums'

// ============================================================
// Constants
// ============================================================

const TABS = [
  { key: 'all', label: '全部' },
  { key: 'project', label: '项目' },
  { key: 'log', label: '日志' },
  { key: 'post', label: '帖子' },
  { key: 'user', label: '用户' },
] as const

type TabKey = (typeof TABS)[number]['key']

import { GENRE_LABELS } from '@/lib/constants'

// ============================================================
// Helpers
// ============================================================

function isProject(item: any): item is ProjectListItem {
  return 'slug' in item && 'genre' in item
}

function isDevLog(item: any): item is DevLog {
  return 'log_type' in item
}

function isPost(item: any): item is Post {
  return !isProject(item) && !isDevLog(item) && 'title' in item
}

function isUser(item: any): boolean {
  return 'username' in item && 'nickname' in item && !('slug' in item) && !('title' in item)
}

// ============================================================
// Result Item Components
// ============================================================

function ProjectResultItem({ data }: { data: ProjectListItem }) {
  return (
    <Link to={`/p/${data.slug}`} className="block group">
      <div className="bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden
                      hover:border-amber/12 hover:shadow-[0_4px_24px_rgba(0,0,0,0.25)]
                      hover:-translate-y-[1px] transition-all duration-200">
        {data.cover_url && (
          <div className="aspect-[16/9] w-full border-b border-white/[0.04]">
            <img src={data.cover_url} alt="" className="w-full h-full object-cover" />
          </div>
        )}
        <div className="p-5">
          <div className="flex items-start gap-3">
            <div className="w-10 h-10 rounded-lg bg-amber/10 flex items-center justify-center shrink-0 ring-1 ring-amber/20">
              <Gamepad2 className="w-5 h-5 text-amber" />
            </div>
            <div className="min-w-0 flex-1">
              <h3 className="text-[15px] font-semibold text-text-primary truncate group-hover:text-amber transition-colors mb-1.5">{data.name}</h3>
              <p className="text-[13px] text-text-secondary leading-relaxed line-clamp-2 mb-2.5">
                {data.description || '暂无简介'}
              </p>
              <div className="flex items-center justify-between mt-2">
                <div className="flex items-center gap-2">
                  <Badge variant="default" size="sm">
                    {GENRE_LABELS[data.genre] || data.genre}
                  </Badge>
                </div>
                <div className="flex items-center gap-1 text-[11px] text-text-muted">
                  <Heart className="w-3 h-3" />
                  <span className="font-mono">{data.follower_count} 关注</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Link>
  )
}

function LogResultItem({ data }: { data: DevLog }) {
  return (
    <Link to={`/devlog/${data.id}`} className="block group">
      <div className="bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden
                      hover:border-amber/12 hover:shadow-[0_4px_24px_rgba(0,0,0,0.25)]
                      hover:-translate-y-[1px] transition-all duration-200">
        {(data.image_urls?.length ?? 0) > 0 && (
          <div className="border-t border-white/[0.04]" onClick={(e) => e.preventDefault()}>
            <ImageGallery images={data.image_urls ?? []} thumbnail maxShow={3} />
          </div>
        )}
        <div className="p-5">
          <div className="flex items-start gap-3">
            <div className="w-10 h-10 rounded-lg bg-coral/10 flex items-center justify-center shrink-0 ring-1 ring-coral/20">
              <ScrollText className="w-5 h-5 text-coral" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2 mb-1.5">
                <Badge variant={data.log_type === 'release' ? 'success' : 'primary'} size="sm">
                  {data.log_type === 'release' ? '发布' : '日志'}
                </Badge>
                <span className="text-[12px] text-text-muted font-mono">{timeAgo(data.created_at)}</span>
              </div>
              <h3 className="text-[15px] font-semibold text-text-primary truncate group-hover:text-amber transition-colors mb-1.5">{data.title}</h3>
              <p className="text-[13px] text-text-secondary leading-relaxed line-clamp-2 mb-2.5">
                {data.content?.replace(/<[^>]*>/g, '').slice(0, 200)}
              </p>
              <div className="flex items-center justify-between mt-2">
                <div />
                <div className="flex items-center gap-3 text-[11px] text-text-muted font-mono">
                  <span className="flex items-center gap-1"><Heart className="w-3 h-3" /> {data.like_count ?? 0}</span>
                  <span className="flex items-center gap-1"><Eye className="w-3 h-3" /> {data.view_count ?? 0}</span>
                  <span className="flex items-center gap-1"><MessageSquare className="w-3 h-3" /> {data.comment_count ?? 0}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Link>
  )
}

function PostResultItem({ data }: { data: Post }) {
  return (
    <Link to={`/post/${data.id}`} className="block group">
      <div className="bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden
                      hover:border-amber/12 hover:shadow-[0_4px_24px_rgba(0,0,0,0.25)]
                      hover:-translate-y-[1px] transition-all duration-200">
        {(data.image_urls?.length ?? 0) > 0 && (
          <div className="border-t border-white/[0.04]" onClick={(e) => e.preventDefault()}>
            <ImageGallery images={data.image_urls ?? []} thumbnail maxShow={3} />
          </div>
        )}
        <div className="p-5">
          <div className="flex items-start gap-3">
            <div className="w-10 h-10 rounded-lg bg-purple-500/10 flex items-center justify-center shrink-0 ring-1 ring-purple-500/20">
              <FileText className="w-5 h-5 text-purple-400" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2 mb-1.5">
                <span className="text-[12px] text-text-muted font-mono">{timeAgo(data.created_at)}</span>
                {data.is_pinned && (
                  <Badge variant="info" size="sm">置顶</Badge>
                )}
              </div>
              <h3 className="text-[15px] font-semibold text-text-primary truncate group-hover:text-amber transition-colors mb-1.5">{data.title}</h3>
              {data.content_snippet && (
                <p className="text-[13px] text-text-secondary leading-relaxed line-clamp-2 mb-2.5">
                  {data.content_snippet}
                </p>
              )}
              <div className="flex items-center justify-end mt-2">
                <div className="flex items-center gap-3 text-[11px] text-text-muted font-mono">
                  <span className="flex items-center gap-1"><Heart className="w-3 h-3" /> {data.like_count ?? 0}</span>
                  <span className="flex items-center gap-1"><MessageSquare className="w-3 h-3" /> {data.comment_count ?? 0}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Link>
  )
}

function UserResultItem({ data }: { data: any }) {
  return (
    <Link to={`/u/${data.username || data.id}`} className="block group">
      <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5
                      hover:border-amber/12 hover:shadow-[0_4px_24px_rgba(0,0,0,0.25)]
                      hover:-translate-y-[1px] transition-all duration-200">
        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-lg bg-info/10 flex items-center justify-center shrink-0 ring-1 ring-info/20">
            <User className="w-5 h-5 text-info" />
          </div>
          <div className="min-w-0 flex-1">
            <h3 className="text-[15px] font-semibold text-text-primary truncate group-hover:text-amber transition-colors mb-1">{data.nickname}</h3>
            <p className="text-[13px] text-text-muted font-mono">@{data.username}</p>
            {data.bio && (
              <p className="text-[13px] text-text-secondary leading-relaxed line-clamp-2 mt-1.5 mb-2.5">{data.bio}</p>
            )}
            <div className="flex items-center gap-3 text-[11px] text-text-muted font-mono">
              <span className="flex items-center gap-1"><Users className="w-3 h-3" /> {data.followers_count} 粉丝</span>
              <span className="flex items-center gap-1"><Gamepad2 className="w-3 h-3" /> {data.projects?.length ?? 0} 项目</span>
            </div>
          </div>
          <Avatar src={data.avatar_url} name={data.nickname} size="sm" className="mt-0.5 shrink-0" />
        </div>
      </div>
    </Link>
  )
}

// ============================================================
// SearchPage
// ============================================================

export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const inputRef = useRef<HTMLInputElement>(null)

  const keyword = searchParams.get('q') ?? ''
  const tab = (searchParams.get('tab') as TabKey) ?? 'all'
  const page = Number(searchParams.get('page') ?? '1')

  const [inputValue, setInputValue] = useState(keyword)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams)
    if (value) {
      next.set(key, value)
    } else {
      next.delete(key)
    }
    if (key !== 'page') next.delete('page')
    setSearchParams(next, { replace: true })
  }

  const handleSearch = () => {
    setParam('q', inputValue.trim())
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSearch()
  }

  const isSearching = keyword.length > 0

  const searchQuery = useQuery({
    queryKey: ['search', keyword, tab, page],
    queryFn: async () => {
      if (tab === 'user') {
        const res = await userApi.searchUsers(keyword, page, 20)
        return { list: res.list as any[], pages: res.pages, total: res.total }
      }
      // Unified search returns PageData<T> (flat list)
      const type = tab === 'all' ? 'all' : (tab === 'project' ? 'projects' : tab === 'log' ? 'logs' : 'posts')
      const res: any = await discoverApi.search({ q: keyword, page, page_size: 20, type: type as any })
      return { list: (res.list ?? []) as any[], pages: res.pages ?? 1, total: res.total ?? 0 }
    },
    enabled: isSearching,
  })

  const results = searchQuery.data?.list ?? []
  const pages = searchQuery.data?.pages ?? 1

  return (
    <div className="max-w-5xl mx-auto px-6 py-6">
      {/* Search Input */}
      <div className="mb-6">
        <h1 className="text-[22px] font-bold text-text-primary mb-4">搜索</h1>
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-xl">
            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-text-muted" />
            <input
              ref={inputRef}
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="输入关键词搜索"
              className={cn(
                'w-full h-11 pl-10 pr-4 bg-surface-void border border-white/[0.04] rounded-xl',
                'text-[14px] text-text-primary',
                'placeholder:text-text-muted',
                'focus:outline-none focus:border-amber focus:shadow-[0_0_0_2px_rgba(245,166,35,0.2)]',
              )}
            />
          </div>
          <Button size="sm" onClick={handleSearch}>
            搜索
          </Button>
        </div>
      </div>

      {!isSearching && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <SearchIcon className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">输入关键词搜索</p>
          <p className="text-[13px] text-text-muted">搜索项目、开发日志、社区帖子和创作者</p>
        </div>
      )}

      {isSearching && (
        <>
          {/* Tabs */}
          <div className="flex gap-6 border-b border-white/[0.04] mb-6">
            {TABS.map((t) => (
              <button
                key={t.key}
                onClick={() => setParam('tab', t.key === 'all' ? '' : t.key)}
                className={cn(
                  'relative pb-3 text-[12px] font-semibold uppercase tracking-wider transition-colors',
                  tab === t.key
                    ? 'text-amber'
                    : 'text-text-muted hover:text-text-secondary',
                )}
              >
                {t.label}
                {tab === t.key && (
                  <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-amber" />
                )}
              </button>
            ))}
          </div>

          {/* Loading */}
          {searchQuery.isLoading && (
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-white/[0.04] animate-pulse shrink-0" />
                    <div className="flex-1 space-y-2">
                      <div className="h-4 w-3/4 bg-white/[0.04] rounded animate-pulse" />
                      <div className="h-3 w-1/2 bg-white/[0.04] rounded animate-pulse" />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Error */}
          {searchQuery.isError && (
            <div className="py-20 flex flex-col items-center justify-center text-center">
              <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
                <Zap className="w-7 h-7 text-text-muted" />
              </div>
              <p className="text-[15px] font-semibold text-text-secondary mb-1">搜索失败</p>
              <p className="text-[13px] text-text-muted mb-6">请检查网络后重试</p>
              <Button variant="secondary" size="sm" onClick={() => searchQuery.refetch()}>重试</Button>
            </div>
          )}

          {/* Empty */}
          {!searchQuery.isLoading && !searchQuery.isError && results.length === 0 && (
            <EmptyState
              icon={<SearchIcon className="w-7 h-7 text-text-muted" />}
              title="没有找到相关内容"
              description={`未找到与 "${keyword}" 相关的内容`}
            />
          )}

          {/* Results */}
          {!searchQuery.isLoading && !searchQuery.isError && results.length > 0 && (
            <>
              <div className="space-y-3">
                {results.map((item: any, i: number) => {
                  if (tab === 'user') return <UserResultItem key={item.id ?? i} data={item} />
                  if (tab === 'project') return <ProjectResultItem key={item.id} data={item} />
                  if (tab === 'log') return <LogResultItem key={item.id} data={item} />
                  if (tab === 'post') return <PostResultItem key={item.id} data={item} />
                  if (isUser(item)) return <UserResultItem key={item.id ?? i} data={item} />
                  if (isProject(item)) return <ProjectResultItem key={item.id} data={item} />
                  if (isDevLog(item)) return <LogResultItem key={item.id} data={item} />
                  if (isPost(item)) return <PostResultItem key={item.id} data={item} />
                  return null
                })}
              </div>
              <Pagination
                page={page}
                pages={pages}
                onChange={(p) => setParam('page', String(p))}
              />
            </>
          )}
        </>
      )}
    </div>
  )
}

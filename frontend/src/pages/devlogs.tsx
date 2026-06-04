import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ScrollText, Heart, Eye, MessageSquare, Zap } from 'lucide-react'
import { logApi, type ListLogsParams } from '@/api/log'
import { ImageGallery } from '@/components/shared/ImageGallery'
import { DevLogCard } from '@/components/shared/DevLogCard'
import { Badge, Avatar, Pagination } from '@/components/ui'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { timeAgo } from '@/lib/time'
import type { DevLogDetail, DevLog } from '@/types/api'

// ============================================================
// Constants
// ============================================================

const TYPE_FILTERS = [
  { value: '' as const, label: '全部' },
  { value: 'log' as const, label: '开发日志' },
  { value: 'release' as const, label: '发布' },
]

const SORT_OPTIONS = [
  { value: 'latest' as const, label: '最新' },
  { value: 'most_liked' as const, label: '最热' },
]

// ============================================================
// Helpers
// ============================================================

function DevLogCardSkeleton() {
  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
      <div className="flex items-start gap-4">
        <div className="w-10 h-10 rounded-lg bg-white/[0.04] animate-pulse shrink-0" />
        <div className="flex-1 space-y-3">
          <div className="flex items-center gap-2">
            <div className="h-5 w-14 bg-white/[0.04] rounded animate-pulse" />
            <div className="h-3 w-20 bg-white/[0.04] rounded animate-pulse" />
          </div>
          <div className="h-5 w-3/4 bg-white/[0.04] rounded animate-pulse" />
          <div className="h-4 w-1/2 bg-white/[0.04] rounded animate-pulse" />
          <div className="flex items-center gap-4">
            <div className="h-3 w-12 bg-white/[0.04] rounded animate-pulse" />
            <div className="h-3 w-12 bg-white/[0.04] rounded animate-pulse" />
            <div className="h-3 w-12 bg-white/[0.04] rounded animate-pulse" />
          </div>
        </div>
      </div>
    </div>
  )
}

// ============================================================
// DevLogCard
// ============================================================


// ============================================================
// DevLogsPage
// ============================================================

export default function DevLogsPage() {
  const [logType, setLogType] = useState<string>('')
  const [sort, setSort] = useState<string>('latest')
  const [page, setPage] = useState(1)

  const query = useQuery({
    queryKey: ['devlogs', { logType, sort, page }],
    queryFn: () =>
      logApi.list({
        page,
        page_size: 20,
        log_type: logType || undefined,
        sort,
      } as ListLogsParams),
  })

  const logs = query.data?.list ?? []
  const pages = query.data?.pages ?? 1

  return (
    <div className="max-w-5xl mx-auto px-6 py-6">
      {/* Header */}
      <h1 className="text-[22px] font-bold text-text-primary mb-6">开发日志</h1>

      {/* Filters */}
      <div className="flex items-center gap-4 mb-6 flex-wrap">
        <div className="flex gap-1 border border-white/[0.04] rounded-xl overflow-hidden">
          {TYPE_FILTERS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => { setLogType(opt.value); setPage(1) }}
              className={cn(
                'px-4 py-2 text-[12px] font-semibold uppercase tracking-wider transition-colors',
                logType === opt.value
                  ? 'bg-amber text-surface-void'
                  : 'text-text-muted hover:text-text-secondary hover:bg-white/[0.02]',
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>

        <div className="h-6 w-px bg-white/[0.04]" />

        <div className="flex gap-0 border border-white/[0.04] rounded-xl overflow-hidden">
          {SORT_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => { setSort(opt.value); setPage(1) }}
              className={cn(
                'px-4 py-2 text-[12px] font-semibold uppercase tracking-wider transition-colors',
                sort === opt.value
                  ? 'bg-amber text-surface-void'
                  : 'text-text-muted hover:text-text-secondary hover:bg-white/[0.02]',
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Loading */}
      {query.isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <DevLogCardSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Error */}
      {query.isError && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Zap className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
          <p className="text-[13px] text-text-muted mb-6">无法加载开发日志，请重试</p>
          <Button variant="secondary" size="sm" onClick={() => query.refetch()}>重新加载</Button>
        </div>
      )}

      {/* Empty */}
      {!query.isLoading && !query.isError && logs.length === 0 && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <ScrollText className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无日志</p>
          <p className="text-[13px] text-text-muted">还没有开发日志，来看看热门项目吧</p>
        </div>
      )}

      {/* List */}
      {!query.isLoading && !query.isError && logs.length > 0 && (
        <>
          <div className="space-y-3">
            {logs.map((log) => (
              <DevLogCard key={log.id} log={log} showAuthor showProject />
            ))}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}
    </div>
  )
}

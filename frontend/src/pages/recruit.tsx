import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Search, Zap, Code2, Palette, ClipboardList, Music, Plus } from 'lucide-react'
import { recruitApi, type ListRecruitmentsParams } from '@/api/recruit'
import { RecruitCard } from '@/components/recruit/RecruitCard'
import { Button } from '@/components/ui/button'
import { Pagination } from '@/components/ui'
import { cn } from '@/lib/utils'
import type { RecruitmentPosition } from '@/types/enums'

const FILTERS: { value: RecruitmentPosition | ''; label: string; icon: React.ComponentType<{ className?: string }> | null }[] = [
  { value: '', label: '全部职位', icon: null },
  { value: 'program', label: '程序', icon: Code2 },
  { value: 'art', label: '美术', icon: Palette },
  { value: 'design', label: '策划', icon: ClipboardList },
  { value: 'sound', label: '音效', icon: Music },
]

function Skeleton() {
  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3 skeleton-shimmer">
      <div className="flex items-start gap-3">
        <div className="w-12 h-12 rounded-[10px] bg-white/[0.04] shrink-0" />
        <div className="flex-1 space-y-2">
          <div className="flex gap-2"><div className="h-5 w-12 rounded-md bg-white/[0.04]" /><div className="h-5 w-16 rounded-md bg-white/[0.04]" /></div>
          <div className="h-4 w-3/4 rounded bg-white/[0.04]" /><div className="h-3 w-full rounded bg-white/[0.04]" />
        </div>
      </div>
      <div className="flex gap-4 pt-1"><div className="h-3 w-16 rounded bg-white/[0.04]" /><div className="h-3 w-12 rounded bg-white/[0.04]" /><div className="h-3 w-20 rounded bg-white/[0.04]" /></div>
    </div>
  )
}

export default function RecruitPage() {
  const [position, setPosition] = useState<RecruitmentPosition | ''>('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)

  const query = useQuery({
    queryKey: ['recruitments', { position, keyword, page }],
    queryFn: () => recruitApi.list({ page, page_size: 20, position: position || undefined, keyword: keyword || undefined } as ListRecruitmentsParams),
  })

  const items = query.data?.list ?? []
  const pages = query.data?.pages ?? 1

  return (
    <div className="max-w-5xl mx-auto px-6 py-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-[22px] font-bold text-text-primary">招募广场</h1>
        <Link to="/recruit/new">
          <Button size="sm"><Plus className="w-3.5 h-3.5" /> 发布招募</Button>
        </Link>
      </div>

      <div className="flex items-center gap-2 mb-6 flex-wrap">
        {FILTERS.map((opt) => (
          <button key={opt.value}
            onClick={() => { setPosition(opt.value as RecruitmentPosition | ''); setPage(1) }}
            className={cn('px-4 py-2 text-[13px] font-semibold border rounded-xl transition-all duration-200 active:scale-95',
              position === opt.value ? 'bg-amber text-surface-void border-amber' : 'text-text-muted border-white/[0.04] hover:text-text-secondary hover:border-text-muted')}>
            {opt.icon && <opt.icon className="w-4 h-4 mr-1.5" />}{opt.label}
          </button>
        ))}

        <div className="ml-auto flex items-center gap-2 h-9 rounded-lg border border-transparent hover:border-white/[0.06] bg-white/[0.03] w-56">
          <Search className="w-4 h-4 text-text-muted ml-3 shrink-0" />
          <input placeholder="搜索..."
            className="bg-transparent border-none outline-none text-[13px] text-text-primary placeholder:text-text-muted w-full pr-3"
            value={keyword}
            onChange={(e) => { setKeyword(e.target.value); setPage(1) }}
          />
        </div>
        <span className="text-[12px] text-text-muted font-mono">{query.data?.total ?? 0} 个招募</span>
      </div>

      {query.isLoading && <div className="space-y-3">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} />)}</div>}

      {query.isError && (
        <div className="py-20 flex flex-col items-center justify-center text-center animate-fade-in">
          <Zap className="w-7 h-7 text-text-muted mb-4" />
          <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
          <p className="text-[13px] text-text-muted mb-6">无法加载招募信息，请重试</p>
          <Button variant="secondary" size="sm" onClick={() => query.refetch()}>重新加载</Button>
        </div>
      )}

      {!query.isLoading && !query.isError && items.length === 0 && (
        <div className="py-20 flex flex-col items-center justify-center text-center animate-fade-in">
          <Search className="w-7 h-7 text-text-muted mb-4" />
          <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无招募</p>
          <p className="text-[13px] text-text-muted mb-6">
            {position || keyword ? '换个筛选试试' : '还没有招募信息，去人才库看看吧'}
          </p>
          {position || keyword ? (
            <Button variant="secondary" size="sm" onClick={() => { setPosition(''); setKeyword('') }}>清除筛选</Button>
          ) : (
            <Link to="/talent"><Button variant="secondary" size="sm">去人才库</Button></Link>
          )}
        </div>
      )}

      {!query.isLoading && !query.isError && items.length > 0 && (
        <>
          <div className="space-y-3">
            {items.map((item, i) => (
              <div key={item.id} className="animate-slide-up" style={{ animationDelay: `${i * 60}ms`, animationFillMode: 'backwards' }}>
                <RecruitCard item={item} />
              </div>
            ))}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}
    </div>
  )
}

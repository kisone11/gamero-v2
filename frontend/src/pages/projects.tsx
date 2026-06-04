import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Heart, Plus, Gamepad2, TrendingUp, Zap } from 'lucide-react'
import { projectApi, type ListProjectsParams } from '@/api/project'
import { useAuthStore } from '@/stores/authStore'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Pagination } from '@/components/ui/pagination'
import { cn } from '@/lib/utils'
import type { ProjectGenre, ProjectStatus } from '@/types/enums'
import type { ProjectListItem } from '@/types/api'
import { GENRE_LABELS, PROJECT_STATUS_CONFIG, GENRE_OPTIONS, STATUS_OPTIONS as SHARED_STATUS_OPTIONS } from '@/lib/constants'

const ALL_GENRE_OPTIONS = [{ value: '', label: '全部类型' }, ...GENRE_OPTIONS]

const ALL_STATUS_OPTIONS = [{ value: '', label: '全部状态' }, ...SHARED_STATUS_OPTIONS]

// ============================================================
// Skeleton
// ============================================================

function ProjectCardSkeleton() {
  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
      <div className="flex items-start gap-4">
        <div className="w-10 h-10 rounded-lg bg-white/[0.04] animate-pulse shrink-0" />
        <div className="flex-1 space-y-3">
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-3 w-full" />
          <Skeleton className="h-3 w-2/3" />
          <div className="flex gap-2">
            <Skeleton className="h-5 w-14 rounded-full" />
            <Skeleton className="h-5 w-14 rounded-full" />
          </div>
          <Skeleton className="h-3 w-20" />
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Project Card — matches FeedPage card quality
// ============================================================

function ProjectCard({ project }: { project: ProjectListItem }) {
  const status = PROJECT_STATUS_CONFIG[project.status]
  const projectHref = `/p/${project.slug || project.id}`

  return (
    <Link
      to={projectHref}
      className="block bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden
                 hover:border-amber/12 hover:shadow-[0_4px_24px_rgba(0,0,0,0.25)]
                 hover:-translate-y-[1px] transition-all duration-200 group"
    >
      {project.cover_url && (
        <div className="aspect-[16/9] w-full border-b border-white/[0.04]">
          <img src={project.cover_url} alt={project.name} className="w-full h-full object-cover" />
        </div>
      )}
      <div className="p-5">
        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-lg bg-amber/10 flex items-center justify-center shrink-0 ring-1 ring-amber/20">
            <Gamepad2 className="w-5 h-5 text-amber" />
          </div>
          <div className="min-w-0 flex-1">
            <h3 className="text-[15px] font-semibold text-text-primary leading-snug mb-1.5 line-clamp-1">
              {project.name}
            </h3>
            <p className="text-[13px] text-text-secondary leading-relaxed line-clamp-2 mb-2.5">
              {project.description || '暂无简介'}
            </p>
            <div className="flex items-center justify-between mt-2">
              <div className="flex items-center gap-2 flex-wrap">
                <Badge variant="default" size="sm">
                  {GENRE_LABELS[project.genre]}
                </Badge>
                <Badge variant={status.variant} size="sm">
                  {status.label}
                </Badge>
              </div>
              <div className="flex items-center gap-1 text-[11px] text-text-muted">
                <Heart className="w-3 h-3" />
                <span className="font-mono">{project.follower_count} 关注</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Link>
  )
}

// ============================================================
// Select Dropdown (inline)
// ============================================================

function Select({
  value,
  onChange,
  options,
  className,
}: {
  value: string
  onChange: (value: string) => void
  options: { value: string; label: string }[]
  className?: string
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={cn(
        'h-10 px-3 bg-surface-void border border-white/[0.04] rounded-xl',
        'text-text-primary text-[13px]',
        'focus:outline-none focus:border-amber focus:shadow-[0_0_0_2px_rgba(245,166,35,0.2)]',
        className,
      )}
    >
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  )
}

// ============================================================
// Projects Page
// ============================================================

export default function ProjectsPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const user = useAuthStore((s) => s.user)
  const ownerMe = searchParams.get('owner') === 'me'

  const [genre, setGenre] = useState<ProjectGenre | ''>('')
  const [status, setStatus] = useState<ProjectStatus | ''>('')
  const [sort, setSort] = useState<'latest' | 'popular'>('latest')
  const [page, setPage] = useState(1)

  const projectsQuery = useQuery({
    queryKey: ['projects', { genre, status, sort, page, ownerMe, userId: user?.id }],
    queryFn: () =>
      projectApi.list({
        page,
        page_size: 12,
        genre: genre || undefined,
        status: status || undefined,
        owner_id: ownerMe && user ? user.id : undefined,
        sort,
      }),
    enabled: !ownerMe || !!user,
  })

  const projects = projectsQuery.data?.list ?? []
  const pages = projectsQuery.data?.pages ?? 1

  const handleGenreChange = (value: string) => {
    setGenre(value as ProjectGenre | '')
    setPage(1)
  }

  const handleStatusChange = (value: string) => {
    setStatus(value as ProjectStatus | '')
    setPage(1)
  }

  const handleSortChange = (newSort: 'latest' | 'popular') => {
    setSort(newSort)
    setPage(1)
  }

  const sortTabs = [
    { key: 'latest' as const, label: '最新' },
    { key: 'popular' as const, label: '最热' },
  ]

  return (
    <div className="max-w-[1280px] mx-auto px-6 py-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-[22px] font-bold text-text-primary">{ownerMe ? '我的项目' : '发现项目'}</h1>
        {user && (
          <Button size="sm" onClick={() => navigate('/projects/new')}>
            <Plus className="w-3.5 h-3.5" />
            创建项目
          </Button>
        )}
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 mb-6 flex-wrap">
        <Select
          value={genre}
          onChange={handleGenreChange}
          options={ALL_GENRE_OPTIONS}
          className="w-32"
        />
        <Select
          value={status}
          onChange={handleStatusChange}
          options={ALL_STATUS_OPTIONS}
          className="w-32"
        />
        <div className="h-6 w-px bg-white/[0.04]" />
        <div className="flex items-center gap-1">
          {sortTabs.map((t) => (
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
      </div>

      {/* Loading */}
      {projectsQuery.isLoading && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <ProjectCardSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Error */}
      {projectsQuery.isError && (
        <div className="text-center py-20">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <TrendingUp className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
          <p className="text-[13px] text-text-muted mb-6">无法加载项目列表，请检查网络后重试</p>
          <Button variant="secondary" size="sm" onClick={() => projectsQuery.refetch()}>
            重新加载
          </Button>
        </div>
      )}

      {/* Empty */}
      {!projectsQuery.isLoading && !projectsQuery.isError && projects.length === 0 && (
        <div className="text-center py-20">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Zap className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无项目</p>
          <p className="text-[13px] text-text-muted mb-6">
            {ownerMe ? '你还没有创建项目' : user ? '还没有项目，来创建第一个吧' : '登录后即可创建项目'}
          </p>
          {user ? (
            <Button variant="secondary" size="sm" onClick={() => navigate('/projects/new')}>
              创建项目
            </Button>
          ) : (
            <Button variant="secondary" size="sm" onClick={() => navigate('/login')}>
              去登录
            </Button>
          )}
        </div>
      )}

      {/* Project grid */}
      {!projectsQuery.isLoading && !projectsQuery.isError && projects.length > 0 && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {projects.map((project) => (
              <ProjectCard key={project.id} project={project} />
            ))}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}
    </div>
  )
}

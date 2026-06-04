import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Download, ScrollText, Zap } from 'lucide-react'
import { logApi } from '@/api/log'
import { projectApi } from '@/api/project'
import { Badge, Pagination } from '@/components/ui'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { timeAgo } from '@/lib/time'
import type { DevLogDetail } from '@/types/api'

// ============================================================
// Helpers
// ============================================================

function ReleaseSkeleton() {
  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3">
      <div className="flex items-center gap-2">
        <div className="h-5 w-14 bg-white/[0.04] rounded animate-pulse" />
        <div className="h-3 w-16 bg-white/[0.04] rounded animate-pulse" />
      </div>
      <div className="h-5 w-3/4 bg-white/[0.04] rounded animate-pulse" />
      <div className="h-4 w-full bg-white/[0.04] rounded animate-pulse" />
    </div>
  )
}

// ============================================================
// ProjectReleasesPage
// ============================================================

export default function ProjectReleasesPage() {
  const { id: projectId } = useParams<{ id: string }>()
  const [page, setPage] = useState(1)
  const [projectName, setProjectName] = useState<string>('')

  const projectQuery = useQuery({
    queryKey: ['project', projectId],
    queryFn: async () => {
      const res = await projectApi.get(projectId!)
      setProjectName(res.name)
      return res
    },
    enabled: !!projectId,
    retry: 1,
  })

  const releasesQuery = useQuery({
    queryKey: ['project-releases', projectId, page],
    queryFn: () =>
      logApi.list({
        project_id: Number(projectId) || undefined,
        log_type: 'release',
        page,
        page_size: 20,
        sort: 'latest',
      }),
    enabled: !!projectId,
  })

  const releases = releasesQuery.data?.list ?? []
  const pages = releasesQuery.data?.pages ?? 1

  const loading = releasesQuery.isLoading || projectQuery.isLoading
  const error = releasesQuery.isError || projectQuery.isError

  return (
    <div className="max-w-4xl mx-auto px-6 py-6">
      <Link to={`/p/${projectId}`} className="inline-flex items-center gap-1 text-[13px] text-amber hover:underline mb-6">
        <ArrowLeft className="h-4 w-4" />
        返回项目
      </Link>

      <h1 className="text-[22px] font-bold text-text-primary mb-6">
        {projectName ? `${projectName} - 发布版本` : '发布版本'}
      </h1>

      {/* Loading */}
      {loading && (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => <ReleaseSkeleton key={i} />)}
        </div>
      )}

      {/* Error */}
      {error && !loading && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Zap className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
          <p className="text-[13px] text-text-muted mb-6">无法加载发布版本</p>
          <Button variant="secondary" size="sm" onClick={() => { releasesQuery.refetch(); projectQuery.refetch() }}>重试</Button>
        </div>
      )}

      {/* Empty */}
      {!loading && !error && releases.length === 0 && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <ScrollText className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无发布版本</p>
          <p className="text-[13px] text-text-muted">该项目还没有发布任何版本</p>
        </div>
      )}

      {/* Release list */}
      {!loading && !error && releases.length > 0 && (
        <>
          <div className="space-y-3">
            {releases.map((release) => {
              const log = release as DevLogDetail
              return (
                <div key={log.id} className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <Badge variant="success" size="sm">发布</Badge>
                        {log.version && (
                          <span className="text-[12px] text-amber font-semibold font-mono">v{log.version}</span>
                        )}
                        <span className="text-[12px] text-text-muted font-mono">{timeAgo(log.created_at)}</span>
                      </div>
                      <h3 className="text-[15px] font-semibold text-text-primary mb-1">{log.title}</h3>
                      <p className="text-[13px] text-text-secondary line-clamp-2">
                        {log.content?.replace(/<[^>]*>/g, '').slice(0, 200) || '暂无说明'}
                      </p>
                    </div>

                    <div className="flex items-center gap-2 shrink-0">
                      {log.download_url && (
                        <a href={log.download_url} target="_blank" rel="noopener noreferrer">
                          <Button variant="outline" size="sm">
                            <Download className="h-4 w-4" />
                            下载
                          </Button>
                        </a>
                      )}
                      <Link to={`/devlog/${log.id}`}>
                        <Button variant="ghost" size="sm">详情</Button>
                      </Link>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}
    </div>
  )
}

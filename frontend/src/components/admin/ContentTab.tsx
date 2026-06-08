import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, FileText, Search } from 'lucide-react'
import { adminApi } from '@/api/admin'
import { communityApi } from '@/api/community'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { Pagination } from '@/components/ui/pagination'
import { Input } from '@/components/ui/input'
import { timeAgo } from '@/lib/time'
import { toast } from '@/stores/toastStore'
import { LogSkeleton } from './AdminSkeletons'
import { AdminListHeader, AdminListPanel, AdminListRow, AdminToolbar } from './AdminList'

export function ContentTab() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [searchInput, setSearchInput] = useState('')

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'posts', page, keyword],
    queryFn: () =>
      communityApi.list({
        page,
        page_size: 15,
        keyword: keyword || undefined,
      }),
  })

  const items = data?.list ?? []
  const pages = data?.pages ?? 1

  const hidePostMut = useMutation({
    mutationFn: (id: number) => adminApi.hidePost(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'posts'] })
      toast.success('已隐藏')
    },
    onError: () => toast.error('操作失败'),
  })

  const deletePostMut = useMutation({
    mutationFn: (id: number) => adminApi.deletePost(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'posts'] })
      toast.success('已删除')
    },
    onError: () => toast.error('操作失败'),
  })

  const handleSearch = () => {
    setKeyword(searchInput.trim())
    setPage(1)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSearch()
  }

  return (
    <div>
      <AdminToolbar title="帖子管理" description="按标题检索社区内容，快速打开、隐藏或删除异常帖子。">
        <div className="flex w-full items-center gap-2 sm:w-[360px]">
          <Input
            placeholder="搜索帖子标题..."
            value={searchInput}
            onChange={e => setSearchInput(e.target.value)}
            onKeyDown={handleKeyDown}
            fullWidth
          />
          <Button variant="secondary" size="sm" onClick={handleSearch}>
            <Search className="h-3.5 w-3.5" />
            搜索
          </Button>
        </div>
      </AdminToolbar>

      {/* Loading */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => <LogSkeleton key={i} />)}
        </div>
      )}

      {/* Error */}
      {isError && !isLoading && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <AlertTriangle className="w-7 h-7 text-text-muted mb-3" />
          <p className="text-body text-text-muted mb-4">无法加载内容列表</p>
          <Button variant="secondary" size="sm" onClick={() => refetch()}>重新加载</Button>
        </div>
      )}

      {/* Empty */}
      {!isLoading && !isError && items.length === 0 && (
        <EmptyState icon={<FileText className="w-7 h-7 text-text-muted" />} title="暂无帖子" />
      )}

      {/* List */}
      {!isLoading && !isError && items.length > 0 && (
        <AdminListPanel>
          <AdminListHeader columns={['帖子', '数据', '操作']} />
          {items.map((item: any) => (
            <AdminListRow key={item.id}>
              <div className="grid gap-4 lg:grid-cols-[1fr_170px_180px] lg:items-center">
                <div className="min-w-0">
                  <Link to={`/post/${item.id}`} target="_blank" className="block truncate text-[14px] font-semibold text-text-primary transition-colors hover:text-amber">{item.title}</Link>
                  <p className="mt-1 line-clamp-1 text-[12px] text-text-muted">{item.content_snippet || item.content}</p>
                  <p className="mt-1 text-[11px] text-text-muted font-mono">ID: {item.id} · Author: {item.author_id} · {timeAgo(item.created_at)}</p>
                </div>
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-text-muted lg:block lg:space-y-1">
                  <p>浏览 {item.view_count ?? 0}</p>
                  <p>互动 {(item.like_count ?? 0) + (item.comment_count ?? 0) + (item.collect_count ?? 0)}</p>
                </div>
                <div className="flex flex-wrap gap-2 lg:justify-end">
                  <Button variant="outline" size="sm" onClick={() => hidePostMut.mutate(item.id)} loading={hidePostMut.isPending}>隐藏</Button>
                  <Button variant="danger" size="sm" onClick={() => deletePostMut.mutate(item.id)} loading={deletePostMut.isPending}>删除</Button>
                </div>
              </div>
            </AdminListRow>
          ))}
        </AdminListPanel>
      )}

      <Pagination page={page} pages={pages} onChange={setPage} />
    </div>
  )
}

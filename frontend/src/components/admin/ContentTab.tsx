import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, FileText } from 'lucide-react'
import { adminApi } from '@/api/admin'
import { communityApi } from '@/api/community'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'
import { Pagination } from '@/components/ui/pagination'
import { Input } from '@/components/ui/input'
import { timeAgo } from '@/lib/time'
import { toast } from '@/stores/toastStore'
import { LogSkeleton } from './AdminSkeletons'

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
      {/* Header */}
      <h3 className="text-h3 text-text-primary mb-4">帖子管理</h3>
      {/* Search */}
      <div className="flex gap-2 mb-4">
        <Input
          placeholder="搜索帖子标题..."
          value={searchInput}
          onChange={e => setSearchInput(e.target.value)}
          onKeyDown={handleKeyDown}
          fullWidth
        />
        <Button variant="secondary" size="sm" onClick={handleSearch}>搜索</Button>
      </div>

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
          <p className="text-body text-text-muted mb-4">加载失败</p>
          <Button variant="secondary" size="sm" onClick={() => refetch()}>重新加载</Button>
        </div>
      )}

      {/* Empty */}
      {!isLoading && !isError && items.length === 0 && (
        <EmptyState icon={<FileText className="w-7 h-7 text-text-muted" />} title="暂无帖子" />
      )}

      {/* List */}
      {!isLoading && !isError && items.length > 0 && (
        <div className="space-y-3">
          {items.map((item: any) => (
            <Card key={item.id} padding="md">
              <div className="flex items-center gap-3">
                <div className="flex-1 min-w-0">
                  <Link to={`/post/${item.id}`} target="_blank" className="text-h4 text-text-primary truncate hover:text-amber transition-colors block">{item.title}</Link>
                  <p className="text-caption text-text-muted font-mono mt-0.5">ID: {item.id} · Author: {item.author_id} · {timeAgo(item.created_at)}</p>
                </div>
                <div className="flex gap-2 shrink-0">
                  <Button variant="outline" size="sm" onClick={() => hidePostMut.mutate(item.id)} loading={hidePostMut.isPending}>隐藏</Button>
                  <Button variant="ghost" size="sm" onClick={() => deletePostMut.mutate(item.id)} loading={deletePostMut.isPending}>删除</Button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      <Pagination page={page} pages={pages} onChange={setPage} />
    </div>
  )
}

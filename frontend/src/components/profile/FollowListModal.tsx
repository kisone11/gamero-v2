import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { userApi } from '@/api/user'
import { Avatar } from '@/components/ui/avatar'
import { Skeleton } from '@/components/ui/skeleton'
import { Pagination } from '@/components/ui/pagination'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import type { FollowUserItem } from '@/types/api'

export const FOLLOW_PAGE_SIZE = 20

export function FollowListModal({
  username,
  type,
  open,
  onClose,
}: {
  username: string
  type: 'followers' | 'following'
  open: boolean
  onClose: () => void
}) {
  const [page, setPage] = useState(1)

  const { data, isLoading } = useQuery({
    queryKey: ['follow-list', username, type, page],
    queryFn: () =>
      type === 'followers'
        ? userApi.getFollowers(username, page, FOLLOW_PAGE_SIZE)
        : userApi.getFollowing(username, page, FOLLOW_PAGE_SIZE),
    enabled: open && !!username,
  })

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { setPage(1); onClose() } }}>
      <DialogContent className="max-w-md max-h-[70vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>{type === 'followers' ? '粉丝' : '关注'}</DialogTitle>
        </DialogHeader>
        <div className="flex-1 overflow-y-auto -mx-6 -mb-6 px-6 pb-6">
          {isLoading ? (
            <div className="space-y-3 pt-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="w-10 h-10 shrink-0 rounded-lg" />
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-4 w-24" />
                    <Skeleton className="h-3 w-32" />
                  </div>
                </div>
              ))}
            </div>
          ) : !data || data.list.length === 0 ? (
            <div className="py-10 text-center">
              <p className="text-body text-text-muted">
                {type === 'followers' ? '暂无粉丝' : '暂未关注任何人'}
              </p>
            </div>
          ) : (
            <div className="space-y-2 pt-2">
              {data.list.map((user: FollowUserItem) => (
                <Link
                  key={user.id}
                  to={`/u/${user.username || user.id}`}
                  onClick={onClose}
                  className="flex items-center gap-3 p-2 rounded-lg hover:bg-white/[0.03] transition-colors group"
                >
                  <Avatar src={user.avatar_url} name={user.nickname} size="md" />
                  <div className="min-w-0 flex-1">
                    <p className="text-body font-medium text-text-primary group-hover:text-amber transition-colors truncate">
                      {user.nickname}
                    </p>
                    <p className="text-caption text-text-muted font-mono truncate">@{user.username}</p>
                  </div>
                </Link>
              ))}
            </div>
          )}
          {data && data.pages > 1 && (
            <div className="mt-2">
              <Pagination page={data.page} pages={data.pages} onChange={setPage} />
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

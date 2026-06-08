import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Search, Shield, Ban, CheckCircle, Users, AlertTriangle } from 'lucide-react'
import { adminApi } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { EmptyState } from '@/components/ui/empty-state'
import { Pagination } from '@/components/ui/pagination'
import { Input } from '@/components/ui/input'
import { formatDate } from '@/lib/time'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'
import { UserSkeleton } from './AdminSkeletons'
import { AdminListHeader, AdminListPanel, AdminListRow, AdminToolbar } from './AdminList'
import type { UserRole } from '@/types/enums'
import type { UserProfileResponse } from '@/types/api'

const USER_ROLE_LABELS: Record<UserRole, string> = {
  user: '用户',
  creator: '创作者',
  moderator: '管理员',
  admin: '高级管理员',
  superadmin: '超级管理员',
}

const USER_ROLE_VARIANTS: Record<UserRole, 'default' | 'amber' | 'info' | 'success' | 'danger'> = {
  user: 'default',
  creator: 'amber',
  moderator: 'info',
  admin: 'success',
  superadmin: 'danger',
}

const MANAGEABLE_ROLES: { value: 'user' | 'moderator' | 'admin'; label: string }[] = [
  { value: 'user', label: '用户' },
  { value: 'moderator', label: '管理员' },
  { value: 'admin', label: '高级管理员' },
]

export function UsersTab() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [unbanTarget, setUnbanTarget] = useState<UserProfileResponse | null>(null)
  const [roleChangeTarget, setRoleChangeTarget] = useState<{ user: UserProfileResponse; role: string } | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'users', page, keyword],
    queryFn: () =>
      adminApi.listUsers({
        page,
        page_size: 15,
        keyword: keyword || undefined,
      }),
  })

  const users = data?.list ?? []
  const pages = data?.pages ?? 1

  const banMut = useMutation({
    mutationFn: ({ userId, reason }: { userId: number; reason: string }) =>
      adminApi.banUser(userId, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
      toast.success('已封禁用户')
    },
    onError: (err: unknown) => toast.error((err as { message?: string })?.message || '封禁失败'),
  })

  const unbanMut = useMutation({
    mutationFn: (userId: number) => adminApi.unbanUser(userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
      toast.success('已解封用户')
    },
    onError: (err: unknown) => toast.error((err as { message?: string })?.message || '解封失败'),
  })

  const roleMut = useMutation({
    mutationFn: ({ userId, role }: { userId: number; role: 'user' | 'moderator' | 'admin' }) =>
      adminApi.setUserRole(userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
      toast.success('角色已更新')
    },
    onError: (err: unknown) => toast.error((err as { message?: string })?.message || '角色更新失败'),
  })

  const handleSearch = () => {
    setKeyword(searchInput.trim())
    setPage(1)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSearch()
  }

  const handleBan = (user: UserProfileResponse) => {
    const reason = window.prompt(`确定封禁用户 ${user.nickname || user.username}？请输入封禁原因：`)
    if (reason !== null) {
      banMut.mutate({ userId: user.id, reason: reason || '违规行为' })
    }
  }

  const handleUnban = (user: UserProfileResponse) => {
    setUnbanTarget(user)
  }

  const handleRoleChange = (user: UserProfileResponse, role: string) => {
    if (role === (user.role ?? 'user')) return
    setRoleChangeTarget({ user, role })
  }

  return (
    <div>
      <AdminToolbar title="用户管理" description="检索用户、调整角色和处理封禁状态。">
        <div className="flex w-full items-center gap-2 sm:w-[360px]">
          <Input
            placeholder="搜索用户名或昵称..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={handleKeyDown}
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
          {Array.from({ length: 5 }).map((_, i) => (
            <UserSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Error */}
      {isError && !isLoading && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <AlertTriangle className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-h3 text-text-secondary mb-1">加载失败</p>
          <p className="text-body text-text-muted mb-6">无法加载用户列表，请重试</p>
          <Button variant="secondary" size="sm" onClick={() => refetch()}>重新加载</Button>
        </div>
      )}

      {/* Empty */}
      {!isLoading && !isError && users.length === 0 && (
        <EmptyState
          icon={<Users className="w-7 h-7 text-text-muted" />}
          title="暂无用户"
          description={keyword ? '没有找到匹配的用户' : '暂无用户数据'}
        />
      )}

      {/* List */}
      {!isLoading && !isError && users.length > 0 && (
        <>
          <AdminListPanel>
            <AdminListHeader columns={['用户', '权限', '操作']} />
            {users.map((user: UserProfileResponse) => (
              <AdminListRow key={user.id}>
                <div className="grid gap-4 lg:grid-cols-[1fr_170px_180px] lg:items-center">
                  <div className="flex min-w-0 items-start gap-3">
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white/[0.04]">
                      {user.avatar_url ? (
                        <img src={user.avatar_url} alt="" className="w-full h-full object-cover" />
                      ) : (
                        <Shield className="w-5 h-5 text-text-muted" />
                      )}
                    </div>
                    <div className="min-w-0">
                      <div className="flex min-w-0 flex-wrap items-center gap-2">
                        <Link to={`/u/${user.username || user.id}`} className="min-w-0 max-w-full truncate text-h4 text-text-primary transition-colors hover:text-amber" target="_blank">
                          {user.nickname || user.username}
                        </Link>
                        {user.is_banned && (
                          <Badge variant="danger" size="sm">已封禁</Badge>
                        )}
                      </div>
                      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-caption text-text-muted">
                        <span>@{user.username}</span>
                        <span>ID: {user.id}</span>
                        <span>注册于 {formatDate(user.created_at)}</span>
                      </div>
                    </div>
                  </div>

                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={USER_ROLE_VARIANTS[user.role ?? 'user']} size="sm">
                      {USER_ROLE_LABELS[user.role ?? 'user']}
                    </Badge>
                    <select
                      value={user.role ?? 'user'}
                      onChange={(e) => handleRoleChange(user, e.target.value)}
                      className="h-8 min-w-[128px] rounded-lg border border-white/[0.06] bg-surface-void px-2 text-small text-text-secondary outline-none focus:border-amber/40"
                    >
                      {MANAGEABLE_ROLES.map((r) => (
                        <option key={r.value} value={r.value}>{r.label}</option>
                      ))}
                    </select>
                  </div>

                  <div className="flex flex-wrap items-center gap-2 lg:justify-end">
                    {user.is_banned ? (
                      <Button
                        variant="secondary"
                        size="sm"
                        loading={unbanMut.isPending}
                        onClick={() => handleUnban(user)}
                      >
                        <CheckCircle className="h-3.5 w-3.5" />
                        解封
                      </Button>
                    ) : (
                      <Button
                        variant="danger"
                        size="sm"
                        loading={banMut.isPending}
                        onClick={() => handleBan(user)}
                      >
                        <Ban className="h-3.5 w-3.5" />
                        封禁
                      </Button>
                    )}
                  </div>
                </div>
              </AdminListRow>
            ))}
          </AdminListPanel>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}

      {/* Unban confirmation dialog */}
      <Dialog open={unbanTarget !== null} onOpenChange={(o) => { if (!o) setUnbanTarget(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>解封用户</DialogTitle><DialogDescription>确定解封用户 {unbanTarget?.nickname || unbanTarget?.username}？</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setUnbanTarget(null)}>取消</Button>
            <Button loading={unbanMut.isPending} onClick={() => {
              if (unbanTarget) {
                unbanMut.mutate(unbanTarget.id)
                setUnbanTarget(null)
              }
            }}>确认解封</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Role change confirmation dialog */}
      <Dialog open={roleChangeTarget !== null} onOpenChange={(o) => { if (!o) setRoleChangeTarget(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>更改角色</DialogTitle><DialogDescription>确定将用户 {roleChangeTarget?.user.nickname || roleChangeTarget?.user.username} 的角色改为 "{roleChangeTarget ? USER_ROLE_LABELS[roleChangeTarget.role as UserRole] ?? roleChangeTarget.role : ''}"？</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setRoleChangeTarget(null)}>取消</Button>
            <Button loading={roleMut.isPending} onClick={() => {
              if (roleChangeTarget) {
                roleMut.mutate({ userId: roleChangeTarget.user.id, role: roleChangeTarget.role as 'user' | 'moderator' | 'admin' })
                setRoleChangeTarget(null)
              }
            }}>确认更改</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

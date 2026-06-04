import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, UserMinus, UserPlus, Zap, Users } from 'lucide-react'
import { projectApi } from '@/api/project'
import { useAuthStore } from '@/stores/authStore'
import { Avatar } from '@/components/ui'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { timeAgo } from '@/lib/time'
import { toast } from '@/stores/toastStore'
import { MEMBER_ROLE_LABELS } from '@/lib/constants'
import type { MemberDetail } from '@/types/api'
import type { MemberRole } from '@/types/enums'

// ============================================================
// Constants
// ============================================================

const ROLE_LABELS: Record<MemberRole, string> = {
  owner: '创建者',
  lead_programmer: '主程',
  lead_artist: '主美',
  lead_designer: '主策',
  sound: '音频',
  tester: '测试',
  member: '成员',
}

const ROLE_COLORS: Record<MemberRole, string> = {
  owner: 'text-amber border-amber/30 bg-amber/10',
  lead_programmer: 'text-blue-400 border-blue-500/30 bg-blue-500/10',
  lead_artist: 'text-pink-400 border-pink-500/30 bg-pink-500/10',
  lead_designer: 'text-purple-400 border-purple-500/30 bg-purple-500/10',
  sound: 'text-green-400 border-green-500/30 bg-green-500/10',
  tester: 'text-orange-400 border-orange-500/30 bg-orange-500/10',
  member: 'text-text-secondary border-white/[0.04] bg-surface-deep',
}

const ROLE_OPTIONS = (Object.entries(MEMBER_ROLE_LABELS) as [MemberRole, string][]).filter(([role]) => role !== 'owner').map(([value, label]) => ({ value, label }))

// ============================================================
// Helpers
// ============================================================

function MemberSkeleton() {
  return (
    <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 flex items-center gap-3">
      <div className="w-10 h-10 rounded-lg bg-white/[0.04] animate-pulse shrink-0" />
      <div className="flex-1 space-y-2">
        <div className="h-4 w-32 bg-white/[0.04] rounded animate-pulse" />
        <div className="h-3 w-24 bg-white/[0.04] rounded animate-pulse" />
      </div>
    </div>
  )
}

// ============================================================
// AddMemberModal
// ============================================================

function AddMemberModal({ projectId, slug, open, onClose }: { projectId: number; slug: string; open: boolean; onClose: () => void }) {
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)
  const [role, setRole] = useState<MemberRole>('member')
  const queryClient = useQueryClient()

  const searchQuery = useQuery({
    queryKey: ['search-users', searchTerm],
    queryFn: async () => {
      const { userApi } = await import('@/api/user')
      return userApi.searchUsers(searchTerm, 1, 10)
    },
    enabled: searchTerm.length >= 2,
  })

  const addMemberMut = useMutation({
    mutationFn: () => projectApi.addMember(projectId, { user_id: selectedUserId!, role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', slug] })
      toast.success('成员已添加')
      onClose()
      setSearchTerm('')
      setSelectedUserId(null)
    },
    onError: (err: any) => toast.error(err?.message || '添加失败'),
  })

  const candidates = searchQuery.data?.list ?? []

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>添加成员</DialogTitle>
          <DialogDescription>搜索用户并设置角色</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <Input value={searchTerm} onChange={(e) => { setSearchTerm(e.target.value); setSelectedUserId(null) }} placeholder="搜索用户名..." fullWidth />
          {candidates.length > 0 && !selectedUserId && (
            <div className="max-h-40 overflow-y-auto space-y-1 border border-white/[0.04] rounded-xl p-1">
              {candidates.map((u: any) => (
                <button key={u.id} onClick={() => setSelectedUserId(u.id)} className="flex items-center gap-3 w-full p-2 rounded-xl hover:bg-white/[0.03] transition-colors text-left">
                  <Avatar src={u.avatar_url} name={u.nickname} size="sm" />
                  <div>
                    <p className="text-[13px] font-semibold text-text-primary">{u.nickname}</p>
                    <p className="text-[11px] text-text-muted font-mono">@{u.username}</p>
                  </div>
                </button>
              ))}
            </div>
          )}
          {searchTerm.length >= 2 && !searchQuery.isLoading && candidates.length === 0 && !selectedUserId && (
            <p className="text-[13px] text-text-muted text-center">未找到用户</p>
          )}
          {selectedUserId && (
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">角色</label>
              <select
                value={role}
                onChange={(e) => setRole(e.target.value as MemberRole)}
                className={cn('h-10 px-3 bg-surface-void border border-white/[0.04] rounded-xl text-[14px] text-text-primary focus:outline-none focus:border-amber')}
              >
                {ROLE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>取消</Button>
          <Button disabled={!selectedUserId} loading={addMemberMut.isPending} onClick={() => addMemberMut.mutate()}>
            <UserPlus className="h-4 w-4" />添加
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================
// ProjectMembersPage
// ============================================================

export default function ProjectMembersPage() {
  const { id: slug } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const me = useAuthStore((s) => s.user)
  const queryClient = useQueryClient()

  const [showAddModal, setShowAddModal] = useState(false)
  const [removeMemberTarget, setRemoveMemberTarget] = useState<MemberDetail | null>(null)
  const [roleChangeTarget, setRoleChangeTarget] = useState<{ member: MemberDetail; role: MemberRole } | null>(null)

  const projectQuery = useQuery({
    queryKey: ['project', slug],
    queryFn: () => projectApi.get(slug!),
    enabled: !!slug,
  })

  const project = projectQuery.data
  const isOwner = project ? me?.id === project.owner_id : false
  const members = project?.members ?? []

  const updateRoleMut = useMutation({
    mutationFn: ({ userId, role }: { userId: number; role: MemberRole }) =>
      projectApi.updateMember(project!.id, userId, { role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', slug] })
      toast.success('角色已更新')
    },
    onError: (err: any) => toast.error(err?.message || '更新失败'),
  })

  const removeMemberMut = useMutation({
    mutationFn: (userId: number) => projectApi.removeMember(project!.id, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', slug] })
      toast.success('成员已移除')
    },
    onError: (err: any) => toast.error(err?.message || '移除失败'),
  })

  const handleRemoveMember = (member: MemberDetail) => {
    setRemoveMemberTarget(member)
  }

  return (
    <div className="max-w-4xl mx-auto px-6 py-6">
      <Link to={`/p/${slug}`} className="inline-flex items-center gap-1 text-[13px] text-amber hover:underline mb-6">
        <ArrowLeft className="h-4 w-4" />
        返回项目
      </Link>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-[22px] font-bold text-text-primary">项目成员</h1>
          {project && <p className="text-[13px] text-text-muted mt-1">{project.name}</p>}
        </div>
        {isOwner && (
          <Button size="sm" onClick={() => setShowAddModal(true)}>
            <UserPlus className="h-4 w-4" />添加成员
          </Button>
        )}
      </div>

      {projectQuery.isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => <MemberSkeleton key={i} />)}
        </div>
      )}

      {projectQuery.isError && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Zap className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
          <p className="text-[13px] text-text-muted mb-6">无法加载项目信息</p>
          <Button variant="secondary" size="sm" onClick={() => projectQuery.refetch()}>重试</Button>
        </div>
      )}

      {!projectQuery.isLoading && !projectQuery.isError && members.length > 0 && (
        <div className="space-y-3">
          {members.map((member) => {
            const isOwnerRole = member.role === 'owner'
            return (
              <div key={member.id} className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
                <div className="flex items-center justify-between gap-4">
                  <div className="flex items-center gap-3 min-w-0 flex-1">
                    <Link to={`/u/${member.username || member.user_id}`} onClick={(e) => e.stopPropagation()}>
                      <Avatar src={member.avatar_url} name={member.nickname || '?'} size="md" />
                    </Link>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <Link to={`/u/${member.username || member.user_id}`} className="text-[14px] font-semibold text-text-primary hover:text-amber transition-colors truncate">
                          {member.nickname || member.username || '未知用户'}
                        </Link>
                        <span className={cn('px-2 py-0.5 text-[11px] font-mono border rounded-lg', ROLE_COLORS[member.role] || ROLE_COLORS.member)}>
                          {ROLE_LABELS[member.role] || member.role}
                        </span>
                      </div>
                      <div className="flex items-center gap-3 mt-0.5 text-[12px] text-text-muted font-mono">
                        {member.contribution && <span>{member.contribution}</span>}
                        <span>加入于 {timeAgo(member.joined_at)}</span>
                      </div>
                    </div>
                  </div>

                  {isOwner && !isOwnerRole && (
                    <div className="flex items-center gap-2 shrink-0">
                      <select
                        value={member.role}
                        onChange={(e) => {
                          const role = e.target.value as MemberRole
                          setRoleChangeTarget({ member, role })
                        }}
                        className={cn('h-9 px-2 bg-surface-void border border-white/[0.04] rounded-xl text-[13px] text-text-primary focus:outline-none focus:border-amber')}
                      >
                        {ROLE_OPTIONS.map((opt) => (
                          <option key={opt.value} value={opt.value}>{opt.label}</option>
                        ))}
                      </select>
                      <Button variant="ghost" size="sm" onClick={() => handleRemoveMember(member)} className="text-danger">
                        <UserMinus className="h-4 w-4" />
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {!projectQuery.isLoading && !projectQuery.isError && members.length === 0 && (
        <div className="py-20 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Users className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">暂无成员</p>
          <p className="text-[13px] text-text-muted">这个项目还没有成员</p>
        </div>
      )}

      {isOwner && project && slug && (
        <AddMemberModal projectId={project.id} slug={slug} open={showAddModal} onClose={() => setShowAddModal(false)} />
      )}

      {/* Remove member confirmation dialog */}
      <Dialog open={removeMemberTarget !== null} onOpenChange={(o) => { if (!o) setRemoveMemberTarget(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>移除成员</DialogTitle><DialogDescription>确定要将 {removeMemberTarget?.nickname || '该成员'} 移出项目吗？</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setRemoveMemberTarget(null)}>取消</Button>
            <Button variant="danger" onClick={() => {
              if (removeMemberTarget) {
                removeMemberMut.mutate(removeMemberTarget.user_id)
                setRemoveMemberTarget(null)
              }
            }}>确认移除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Role change confirmation dialog */}
      <Dialog open={roleChangeTarget !== null} onOpenChange={(o) => { if (!o) setRoleChangeTarget(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>更改角色</DialogTitle><DialogDescription>将 {roleChangeTarget?.member.nickname || '该成员'} 的角色改为 {roleChangeTarget ? ROLE_LABELS[roleChangeTarget.role] : ''} 吗？</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setRoleChangeTarget(null)}>取消</Button>
            <Button onClick={() => {
              if (roleChangeTarget) {
                updateRoleMut.mutate({ userId: roleChangeTarget.member.user_id, role: roleChangeTarget.role })
                setRoleChangeTarget(null)
              }
            }}>确认更改</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

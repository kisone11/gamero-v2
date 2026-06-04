import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Users, UserMinus } from 'lucide-react'
import { projectApi } from '@/api/project'
import { collabApi } from '@/api/collab'
import { Button, Badge, Card, Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, EmptyState } from '@/components/ui'
import { Avatar } from '@/components/ui/avatar'
import { MEMBER_ROLE_LABELS } from '@/lib/constants'
import { toast } from '@/stores/toastStore'
import { CollabReviewDialog } from './CollabReviewDialog'
import type { MemberDetail } from '@/types/api'

export function ProjectMembersTab({
  members, projectId, isOwner, currentUser,
}: {
  members: MemberDetail[]
  projectId: number
  isOwner: boolean
  currentUser: { id: number } | null
}) {
  const queryClient = useQueryClient()
  const [removeMemberTarget, setRemoveMemberTarget] = useState<MemberDetail | null>(null)
  const [leaveOpen, setLeaveOpen] = useState(false)
  const [collabTarget, setCollabTarget] = useState<MemberDetail | null>(null)
  const [collabReviewOpen, setCollabReviewOpen] = useState(false)

  const removeMemberMutation = useMutation({
    mutationFn: (userId: number) => projectApi.removeMember(projectId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project'] })
      toast.success('已移除成员')
    },
    onError: () => toast.error('移除失败'),
  })

  const leaveProjectMutation = useMutation({
    mutationFn: () => projectApi.leaveProject(projectId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project'] })
      toast.success('已退出项目')
    },
    onError: () => toast.error('退出失败'),
  })

  const createCollabReviewMutation = useMutation({
    mutationFn: (data: { reviewee_id: number; rating: number; comment: string; tags?: string[] }) =>
      collabApi.createReview({ project_id: projectId, ...data }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-reviews', projectId] })
      queryClient.invalidateQueries({ queryKey: ['review-summary', projectId] })
      setCollabReviewOpen(false)
      setCollabTarget(null)
      toast.success('评价已提交')
    },
    onError: () => toast.error('提交失败'),
  })

  if (members.length === 0) {
    return (
      <div className="py-6">
        <EmptyState icon={<Users className="w-7 h-7 text-text-muted" />} title="暂无成员" description="该项目还没有任何成员" />
      </div>
    )
  }

  const sorted = [...members].sort((a, b) => {
    if (a.role === 'owner') return -1
    if (b.role === 'owner') return 1
    return a.joined_at.localeCompare(b.joined_at)
  })

  return (
    <div className="py-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-h3 text-text-primary">项目成员 ({members.length})</h3>
        {isOwner && (
          <Link to={`/project/${projectId}/members`}>
            <Button variant="secondary" size="sm">
              <Users className="h-4 w-4" />
              管理成员
            </Button>
          </Link>
        )}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {sorted.map(member => {
          const isCurrentUser = currentUser?.id === member.user_id
          return (
            <Card key={member.id} padding="lg">
              <div className="flex items-center gap-3">
                <Link to={`/u/${member.username || member.user_id}`}>
                  <Avatar src={member.avatar_url} name={member.nickname} size="md" />
                </Link>
                <div className="min-w-0 flex-1">
                  <Link
                    to={`/u/${member.username || member.user_id}`}
                    className="text-h4 text-text-primary hover:text-amber transition-colors"
                  >
                    {member.nickname || member.username}
                  </Link>
                  <div className="flex items-center gap-2 mt-0.5">
                    <Badge variant={member.role === 'owner' ? 'primary' : 'default'} size="sm">
                      {MEMBER_ROLE_LABELS[member.role]}
                    </Badge>
                  </div>
                  {member.contribution && (
                    <p className="text-small text-text-muted mt-1 line-clamp-1">{member.contribution}</p>
                  )}
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  {isOwner && !isCurrentUser && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setRemoveMemberTarget(member)}
                      className="text-danger"
                    >
                      <UserMinus className="h-4 w-4" />
                    </Button>
                  )}
                  {!isCurrentUser && currentUser && (
                    <Button variant="ghost" size="sm" onClick={() => {
                      setCollabTarget(member)
                      setCollabReviewOpen(true)
                    }}>
                      评价队友
                    </Button>
                  )}
                </div>
              </div>
            </Card>
          )
        })}
      </div>

      {!isOwner && currentUser && members.some(m => m.user_id === currentUser.id) && (
        <div className="mt-6 text-center">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setLeaveOpen(true)}
            className="text-danger"
          >
            退出项目
          </Button>
        </div>
      )}

      {/* Remove member confirmation dialog */}
      <Dialog open={removeMemberTarget !== null} onOpenChange={(o) => { if (!o) setRemoveMemberTarget(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>移除成员</DialogTitle><DialogDescription>确定要移除成员 {removeMemberTarget?.nickname || removeMemberTarget?.username} 吗？</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setRemoveMemberTarget(null)}>取消</Button>
            <Button variant="danger" onClick={() => {
              if (removeMemberTarget) {
                removeMemberMutation.mutate(removeMemberTarget.user_id)
                setRemoveMemberTarget(null)
              }
            }}>确认移除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Leave project confirmation dialog */}
      <Dialog open={leaveOpen} onOpenChange={setLeaveOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>退出项目</DialogTitle><DialogDescription>确定要退出项目吗？</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setLeaveOpen(false)}>取消</Button>
            <Button variant="danger" onClick={() => { leaveProjectMutation.mutate(); setLeaveOpen(false) }}>确认退出</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Collab review dialog */}
      <CollabReviewDialog
        open={collabReviewOpen}
        onOpenChange={(open) => {
          setCollabReviewOpen(open)
          if (!open) setCollabTarget(null)
        }}
        target={collabTarget}
        onSubmit={(data) => createCollabReviewMutation.mutate(data)}
        loading={createCollabReviewMutation.isPending}
      />
    </div>
  )
}

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Inbox, Zap, Check, X } from 'lucide-react'
import { talentApi } from '@/api/talent'
import { InvitationItem } from '@/components/recruit/InvitationItem'
import { Button, Pagination } from '@/components/ui'
import { toast } from '@/stores/toastStore'
import type { InvitationDetail } from '@/types/api'

function Skeleton() { return <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3 skeleton-shimmer"><div className="flex items-center gap-2"><div className="h-4 w-14 rounded bg-white/[0.04]" /><div className="h-3 w-16 rounded bg-white/[0.04]" /></div><div className="h-4 w-3/4 rounded bg-white/[0.04]" /><div className="h-3 w-1/2 rounded bg-white/[0.04]" /></div> }

export default function MyInvitationsPage() {
  const qc = useQueryClient()
  const [page, setPage] = useState(1)
  const query = useQuery({ queryKey: ['my-invitations', page], queryFn: () => talentApi.listMyInvitations(page, 20) })
  const invs = query.data?.list ?? []
  const pages = query.data?.pages ?? 1

  const acceptMut = useMutation({
    mutationFn: (id: number) => talentApi.acceptInvitation(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['my-invitations'] }); toast.success('已接受邀请') },
    onError: (e: any) => toast.error(e?.message || '操作失败'),
  })
  const declineMut = useMutation({
    mutationFn: (id: number) => talentApi.declineInvitation(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['my-invitations'] }); toast.success('已拒绝邀请') },
    onError: (e: any) => toast.error(e?.message || '操作失败'),
  })

  return (
    <div className="max-w-4xl mx-auto px-6 py-6 animate-fade-in">
      <h1 className="text-[22px] font-bold text-text-primary mb-6">我的邀请</h1>
      {query.isLoading && <div className="space-y-3">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} />)}</div>}
      {query.isError && (
        <div className="py-16 flex flex-col items-center justify-center text-center">
          <Zap className="w-6 h-6 text-text-muted mb-3" /><p className="text-[13px] text-text-muted mb-4">加载失败</p>
          <Button variant="secondary" size="sm" onClick={() => query.refetch()}>重试</Button>
        </div>
      )}
      {!query.isLoading && !query.isError && invs.length === 0 && (
        <div className="py-16 flex flex-col items-center justify-center text-center">
          <Inbox className="w-6 h-6 text-text-muted mb-3" /><p className="text-[15px] font-semibold text-text-secondary mb-1">暂无邀请</p>
          <p className="text-[13px] text-text-muted">你还没有收到任何邀请</p>
        </div>
      )}
      {!query.isLoading && !query.isError && invs.length > 0 && (
        <>
          <div className="space-y-3">
            {invs.map((inv: InvitationDetail, i: number) => (
              <div key={inv.id} className="animate-slide-up" style={{ animationDelay: `${i * 50}ms`, animationFillMode: 'backwards' }}>
                <InvitationItem inv={inv} actions={
                  inv.status === 'pending' ? (
                    <div className="flex items-center gap-2">
                      <Button variant="outline" size="sm" loading={declineMut.isPending} onClick={() => declineMut.mutate(inv.id)} className="text-danger"><X className="h-3.5 w-3.5" /> 拒绝</Button>
                      <Button size="sm" loading={acceptMut.isPending} onClick={() => acceptMut.mutate(inv.id)}><Check className="h-3.5 w-3.5" /> 接受</Button>
                    </div>
                  ) : undefined
                } />
              </div>
            ))}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}
    </div>
  )
}

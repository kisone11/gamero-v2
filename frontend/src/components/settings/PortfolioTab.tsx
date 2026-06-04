import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2, Briefcase } from 'lucide-react'
import { userApi } from '@/api/user'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'
import { Select } from '@/components/settings/Select'
import type { Portfolio } from '@/types/api'
import type { PortfolioType } from '@/types/enums'

const PORTFOLIO_TYPES: { value: PortfolioType; label: string }[] = [
  { value: 'game', label: '游戏' },
  { value: 'demo', label: 'Demo' },
  { value: 'art', label: '美术作品' },
  { value: 'code', label: '代码' },
]

export function PortfolioTab() {
  const queryClient = useQueryClient()
  const user = useAuthStore((s) => s.user)

  const [items, setItems] = useState<Portfolio[]>([])
  const [loaded, setLoaded] = useState(false)
  const [deletePortfolioTarget, setDeletePortfolioTarget] = useState<Portfolio | null>(null)

  const [pName, setPName] = useState('')
  const [pType, setPType] = useState<PortfolioType>('game')
  const [pLink, setPLink] = useState('')
  const [pDescription, setPDescription] = useState('')

  const profileQuery = useQuery({
    queryKey: ['user-profile', user?.username],
    queryFn: () => userApi.getProfile(user!.username),
    enabled: !!user?.username,
  })

  useEffect(() => {
    if (profileQuery.data && !loaded) {
      setItems(profileQuery.data.portfolio ?? [])
      setLoaded(true)
    }
  }, [profileQuery.data, loaded])

  const createPortfolioMut = useMutation({
    mutationFn: () =>
      userApi.createPortfolio({
        name: pName.trim(),
        type: pType,
        link: pLink.trim() || undefined,
        description: pDescription.trim() || undefined,
      }),
    onSuccess: (newItem) => {
      queryClient.invalidateQueries({ queryKey: ['user-profile', user?.username] })
      setItems((prev) => [...prev, newItem])
      setPName('')
      setPType('game')
      setPLink('')
      setPDescription('')
      toast.success('作品已添加')
    },
    onError: (err: any) => toast.error(err?.message || '添加失败'),
  })

  const deletePortfolioMut = useMutation({
    mutationFn: (id: number) => userApi.deletePortfolio(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['user-profile', user?.username] })
      setItems((prev) => prev.filter((item) => item.id !== id))
      toast.success('作品已删除')
    },
    onError: (err: any) => toast.error(err?.message || '删除失败'),
  })

  const portfolioTypeLabel = (t: PortfolioType) =>
    PORTFOLIO_TYPES.find((pt) => pt.value === t)?.label ?? t

  if (profileQuery.isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full rounded-xl" />
        <Skeleton className="h-10 w-full rounded-xl" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-[16px] font-semibold text-text-primary">作品集</h2>
        <p className="text-body text-text-muted mt-1">展示你的代表作品</p>
      </div>

      {items.length > 0 && (
        <div className="space-y-3">
          <label className="text-meta font-semibold text-text-muted uppercase tracking-wider">
            作品列表 ({items.length})
          </label>
          {items.map((item) => (
            <Card key={item.id}>
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <h4 className="text-h4 text-text-primary">{item.name}</h4>
                    <Badge variant="info" size="sm">{portfolioTypeLabel(item.type)}</Badge>
                  </div>
                  {item.description && (
                    <p className="text-body text-text-secondary line-clamp-2">{item.description}</p>
                  )}
                  {item.link && (
                    <a
                      href={item.link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-caption font-mono text-amber hover:underline mt-1 inline-block"
                    >
                      {item.link}
                    </a>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setDeletePortfolioTarget(item)}
                  className="text-danger shrink-0"
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {items.length === 0 && (
        <div className="text-center py-8 bg-white/[0.02] border border-white/[0.04] rounded-xl">
          <div className="w-12 h-12 mx-auto mb-3 rounded-xl bg-white/[0.03] flex items-center justify-center">
            <Briefcase className="w-6 h-6 text-text-muted" />
          </div>
          <p className="text-h4 text-text-secondary">暂无作品</p>
          <p className="text-small text-text-muted mt-1">添加你的作品展示</p>
        </div>
      )}

      <div className="space-y-3 p-4 border border-white/[0.04] rounded-xl bg-surface-card">
        <label className="text-meta font-semibold text-text-muted uppercase tracking-wider">
          添加作品
        </label>
        <Input
          label="作品名称"
          value={pName}
          onChange={(e) => setPName(e.target.value)}
          fullWidth
        />
        <div className="flex flex-col gap-1.5">
          <label className="text-meta font-semibold text-text-muted uppercase tracking-wider">
            类型
          </label>
          <Select
            value={pType}
            onChange={(v) => setPType(v as PortfolioType)}
            options={PORTFOLIO_TYPES}
          />
        </div>
        <Input
          label="链接（选填）"
          value={pLink}
          onChange={(e) => setPLink(e.target.value)}
          fullWidth
          placeholder="https://"
        />
        <Textarea
          label="描述（选填）"
          value={pDescription}
          onChange={(e) => setPDescription(e.target.value)}
          rows={3}
          fullWidth
        />
        <Button
          size="sm"
          loading={createPortfolioMut.isPending}
          disabled={!pName.trim()}
          onClick={() => createPortfolioMut.mutate()}
        >
          <Plus className="w-4 h-4" />
          添加
        </Button>
      </div>

      <Dialog open={deletePortfolioTarget !== null} onOpenChange={(o) => { if (!o) setDeletePortfolioTarget(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>删除作品</DialogTitle><DialogDescription>确定要删除作品 "{deletePortfolioTarget?.name}" 吗？</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeletePortfolioTarget(null)}>取消</Button>
            <Button variant="danger" onClick={() => {
              if (deletePortfolioTarget) {
                deletePortfolioMut.mutate(deletePortfolioTarget.id)
                setDeletePortfolioTarget(null)
              }
            }}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

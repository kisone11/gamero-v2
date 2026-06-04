import { useEffect, useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { Search, Zap, Users, Code2, Palette, ClipboardList, Music, Mail } from 'lucide-react'
import { talentApi, type ListTalentsParams } from '@/api/talent'
import { useAuthStore } from '@/stores/authStore'
import { TalentCard } from '@/components/recruit/TalentCard'
import { Button, Pagination } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Textarea } from '@/components/ui/textarea'
import { toast } from '@/stores/toastStore'
import { cn } from '@/lib/utils'
import type { TalentListItem } from '@/types/api'
import type { RecruitmentPosition, SkillCategory } from '@/types/enums'

const POS_OPTIONS: { value: RecruitmentPosition; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { value: 'program', label: '程序', icon: Code2 }, { value: 'art', label: '美术', icon: Palette },
  { value: 'design', label: '策划', icon: ClipboardList }, { value: 'sound', label: '音效', icon: Music },
]

function Skeleton() { return <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3"><div className="flex items-center gap-3"><div className="w-12 h-12 rounded-full bg-white/[0.04] animate-pulse" /><div className="flex-1 space-y-2"><div className="h-4 w-24 bg-white/[0.04] rounded animate-pulse" /><div className="h-3 w-16 bg-white/[0.04] rounded animate-pulse" /></div></div><div className="h-3 w-full bg-white/[0.04] rounded animate-pulse" /><div className="flex gap-2"><div className="h-5 w-16 bg-white/[0.04] rounded animate-pulse" /><div className="h-5 w-16 bg-white/[0.04] rounded animate-pulse" /></div></div> }

export default function TalentPage() {
  const user = useAuthStore((s) => s.user)
  const [keyword, setKeyword] = useState('')
  const [debouncedKeyword, setDebouncedKeyword] = useState('')
  const [skillCategory, setSkillCategory] = useState<SkillCategory | ''>('')
  const [level, setLevel] = useState('')
  const [coopPreference, setCoopPreference] = useState('')
  const [onlyAvailable, setOnlyAvailable] = useState(true)
  const [page, setPage] = useState(1)
  const [inviteTarget, setInviteTarget] = useState<TalentListItem | null>(null)
  const [invProject, setInvProject] = useState('')
  const [invPosition, setInvPosition] = useState<RecruitmentPosition>('program')
  const [invMessage, setInvMessage] = useState('')
  const [invSent, setInvSent] = useState(false)

  // Debounce keyword
  useEffect(() => {
    const t = setTimeout(() => setDebouncedKeyword(keyword), 300)
    return () => clearTimeout(t)
  }, [keyword])

  // Reset page to 1 when any filter changes
  useEffect(() => {
    setPage(1)
  }, [debouncedKeyword, skillCategory, level, coopPreference, onlyAvailable])

  const query = useQuery({
    queryKey: ['talents', { page, keyword: debouncedKeyword, skillCategory, level, coopPreference, onlyAvailable }],
    queryFn: () => talentApi.list({
      page,
      page_size: 12,
      only_available: onlyAvailable,
      ...(debouncedKeyword && { keyword: debouncedKeyword }),
      ...(skillCategory && { category: skillCategory }),
      ...(level && { level }),
      ...(coopPreference && { coop_preference: coopPreference }),
    } as ListTalentsParams),
  })
  const { data: myProjects } = useQuery({ queryKey: ['my-projects-for-invite'], queryFn: async () => { const { projectApi } = await import('@/api/project'); return projectApi.list({ page: 1, page_size: 50 }) }, enabled: !!inviteTarget })
  const inviteMut = useMutation({
    mutationFn: () => talentApi.invite({ project_id: Number(invProject), talent_id: inviteTarget!.id, position: invPosition, message: invMessage.trim() || undefined }),
    onSuccess: () => { setInvSent(true); setTimeout(() => { setInviteTarget(null); setInvSent(false); setInvProject(''); setInvMessage('') }, 2000) },
    onError: (e: any) => toast.error(e?.message || '邀请失败'),
  })

  const items = query.data?.list ?? []
  const pages = query.data?.pages ?? 1

  const skillCategories: { value: SkillCategory | ''; label: string }[] = [
    { value: '', label: '全部' },
    { value: 'program', label: '程序' },
    { value: 'art', label: '美术' },
    { value: 'design', label: '设计' },
    { value: 'sound', label: '音效' },
  ]

  return (
    <div className="max-w-6xl mx-auto px-6 py-6">
      <h1 className="text-[22px] font-bold text-text-primary mb-6">人才库</h1>

      {/* Filter bar */}
      <div className="flex flex-wrap items-center gap-3 mb-6">
        {/* Keyword search */}
        <div className="relative flex-1 min-w-[180px] max-w-[280px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted pointer-events-none" />
          <input
            type="text"
            placeholder="搜索..."
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="w-full h-10 pl-9 pr-3.5 bg-surface-void rounded-lg text-[14px] text-text-primary placeholder:text-text-muted border border-white/[0.06] hover:border-white/[0.10] focus:border-amber/40 focus:shadow-[0_0_0_3px_rgba(245,166,35,0.08)] outline-none transition-all duration-150"
          />
        </div>

        {/* Skill category toggles */}
        <div className="flex gap-1">
          {skillCategories.map((cat) => (
            <button
              key={cat.value}
              onClick={() => setSkillCategory(cat.value)}
              className={cn(
                'px-3 py-1.5 text-[12px] font-medium rounded-lg border transition-all active:scale-95',
                skillCategory === cat.value
                  ? 'bg-amber/10 text-amber border-amber/15'
                  : 'bg-surface-void text-text-muted border-white/[0.06] hover:border-white/[0.10]'
              )}
            >
              {cat.label}
            </button>
          ))}
        </div>

        {/* Level select */}
        <select
          value={level}
          onChange={(e) => setLevel(e.target.value)}
          className="h-10 px-3 bg-surface-void rounded-lg text-[13px] text-text-primary border border-white/[0.06] outline-none cursor-pointer hover:border-white/[0.10] focus:border-amber/40 transition-all duration-150"
        >
          <option value="">等级</option>
          <option value="beginner">入门</option>
          <option value="intermediate">熟练</option>
          <option value="advanced">精通</option>
        </select>

        {/* Coop preference select */}
        <select
          value={coopPreference}
          onChange={(e) => setCoopPreference(e.target.value)}
          className="h-10 px-3 bg-surface-void rounded-lg text-[13px] text-text-primary border border-white/[0.06] outline-none cursor-pointer hover:border-white/[0.10] focus:border-amber/40 transition-all duration-150"
        >
          <option value="">合作方式</option>
          <option value="online">在线</option>
          <option value="offline">线下</option>
          <option value="hybrid">混合</option>
        </select>

      </div>

      {query.isLoading && <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} />)}</div>}
      {query.isError && (
        <div className="py-20 flex flex-col items-center justify-center text-center animate-fade-in">
          <Zap className="w-7 h-7 text-text-muted mb-4" /><p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
          <p className="text-[13px] text-text-muted mb-6">无法加载人才列表，请重试</p>
          <Button variant="secondary" size="sm" onClick={() => query.refetch()}>重新加载</Button>
        </div>
      )}
      {!query.isLoading && !query.isError && items.length === 0 && (
        <div className="py-20 flex flex-col items-center justify-center text-center animate-fade-in">
          <Users className="w-7 h-7 text-text-muted mb-4" /><p className="text-[15px] font-semibold text-text-secondary mb-1">暂无人才</p>
          <p className="text-[13px] text-text-muted">还没有创作者入驻人才库</p>
        </div>
      )}
      {!query.isLoading && !query.isError && items.length > 0 && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {items.map((item, i) => (
              <div key={item.id} className="animate-slide-up" style={{ animationDelay: `${i * 60}ms`, animationFillMode: 'backwards' }}>
                <TalentCard talent={item} onInvite={user ? setInviteTarget : () => { }} />
              </div>
            ))}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}

      <Dialog open={!!inviteTarget} onOpenChange={(o) => { if (!o && !inviteMut.isPending) { setInviteTarget(null); setTimeout(() => setInvSent(false), 300) } }}>
        <DialogContent>
          {invSent ? (
            <div className="text-center py-8 animate-bounce-in">
              <Mail className="w-10 h-10 text-success mb-3 mx-auto" />
              <h3 className="text-[16px] font-bold text-success mb-1">邀请已发送！</h3>
              <p className="text-[13px] text-text-muted">{inviteTarget?.nickname} 将在通知中收到你的邀请</p>
            </div>
          ) : (
            <>
              <DialogHeader><DialogTitle>邀请 {inviteTarget?.nickname} 合作</DialogTitle><DialogDescription>向他发送合作邀请</DialogDescription></DialogHeader>
              <div className="space-y-4">
                <div className="flex flex-col gap-1.5">
                  <label className="text-[11px] font-semibold text-text-secondary uppercase tracking-wider">选择项目</label>
                  <div className="flex flex-wrap gap-2">
                    {(myProjects as any)?.list?.map((p: any) => (
                      <button key={p.id} onClick={() => setInvProject(String(p.id))}
                        className={cn('px-3 py-2 text-[12px] font-medium rounded-lg border transition-all active:scale-95',
                          invProject === String(p.id) ? 'border-amber bg-amber/10 text-amber' : 'border-white/[0.04] text-text-muted hover:border-white/[0.08]')}>{p.name}</button>
                    ))}
                  </div>
                </div>
                <div className="flex flex-col gap-1.5">
                  <label className="text-[11px] font-semibold text-text-secondary uppercase tracking-wider">邀请职位</label>
                  <div className="flex gap-2">
                    {POS_OPTIONS.map((opt) => (
                      <button key={opt.value} onClick={() => setInvPosition(opt.value)}
                        className={cn('flex items-center gap-1 px-3 py-2 text-[12px] font-medium rounded-lg border transition-all active:scale-95',
                          invPosition === opt.value ? 'border-amber bg-amber/10 text-amber' : 'border-white/[0.04] text-text-muted hover:border-white/[0.08]')}><opt.icon className="w-4 h-4" /> {opt.label}</button>
                    ))}
                  </div>
                </div>
                <Textarea label="邀请留言" value={invMessage} onChange={(e) => setInvMessage(e.target.value)} placeholder="简单介绍一下你的项目..." rows={3} />
              </div>
              <DialogFooter>
                <Button variant="ghost" onClick={() => setInviteTarget(null)}>取消</Button>
                <Button loading={inviteMut.isPending} disabled={!invProject} onClick={() => inviteMut.mutate()}>发送邀请</Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

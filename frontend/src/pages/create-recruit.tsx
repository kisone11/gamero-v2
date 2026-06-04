import { useEffect, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useMutation, useQuery } from '@tanstack/react-query'
import { ArrowLeft, Code2, Palette, ClipboardList, Music } from 'lucide-react'
import { recruitApi } from '@/api/recruit'
import { projectApi } from '@/api/project'
import { Button, Input, Textarea, Skeleton } from '@/components/ui'
import { toast } from '@/stores/toastStore'
import { useAuthStore } from '@/stores/authStore'
import { cn } from '@/lib/utils'
import type { CooperationType, ContactType, RecruitmentPosition } from '@/types/enums'

const POSITIONS: { value: RecruitmentPosition; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { value: 'program', label: '程序', icon: Code2 }, { value: 'art', label: '美术', icon: Palette },
  { value: 'design', label: '策划', icon: ClipboardList }, { value: 'sound', label: '音效', icon: Music },
]
const COOP_TYPES: { value: CooperationType; label: string }[] = [{ value: 'online', label: '线上合作' }, { value: 'offline', label: '线下合作' }, { value: 'hybrid', label: '混合模式' }]
const CONTACT_OPTIONS: { value: ContactType; label: string }[] = [{ value: 'platform', label: '平台私信' }, { value: 'wechat', label: '微信' }, { value: 'qq', label: 'QQ' }]
const EXPIRE_OPTIONS = [{ value: 7, label: '7 天' }, { value: 14, label: '14 天' }, { value: 30, label: '30 天' }, { value: 60, label: '60 天' }, { value: 90, label: '90 天' }]
const HEADCOUNT_OPTIONS = Array.from({ length: 10 }, (_, i) => i + 1)

export default function CreateRecruitPage() {
  const { slug, id } = useParams<{ slug?: string; id?: string }>()
  const navigate = useNavigate()
  const user = useAuthStore(s => s.user)
  const recruitId = id ? Number(id) : undefined
  const isEdit = recruitId != null && !Number.isNaN(recruitId)
  const { data: recruit, isLoading: recruitLoading } = useQuery({ queryKey: ['recruitment', recruitId], queryFn: () => recruitApi.get(recruitId!), enabled: isEdit })
  const projectSlug = slug ?? recruit?.project_slug
  const { data: project, isLoading: projectLoading, isError: projectError, error: projectLoadError } = useQuery({ queryKey: ['project', projectSlug], queryFn: () => projectApi.get(projectSlug!), enabled: !!projectSlug && !isEdit })
  const { data: myProjectsData, isLoading: myProjectsLoading } = useQuery({
    queryKey: ['my-projects-for-recruit', user?.id],
    queryFn: () => projectApi.list({ owner_id: user!.id, page: 1, page_size: 50 }),
    enabled: !isEdit && !projectSlug && !!user?.id,
  })

  const [position, setPosition] = useState<RecruitmentPosition>('program')
  const [headcount, setHeadcount] = useState(1)
  const [coopType, setCoopType] = useState<CooperationType>('online')
  const [description, setDescription] = useState('')
  const [contactType, setContactType] = useState<ContactType>('platform')
  const [contactInfo, setContactInfo] = useState('')
  const [expireDays, setExpireDays] = useState(30)
  const [selectedProjectId, setSelectedProjectId] = useState<number | ''>('')

  const myProjects = myProjectsData?.list ?? []
  const selectedProject = project ?? myProjects.find((item) => item.id === selectedProjectId)

  useEffect(() => {
    if (!recruit) return
    setPosition(recruit.position)
    setHeadcount(recruit.headcount)
    setCoopType(recruit.cooperation_type)
    setDescription(recruit.description ?? '')
    setContactType(recruit.contact_type ?? 'platform')
    setContactInfo(recruit.contact_info ?? '')
  }, [recruit])

  const createMut = useMutation({
    mutationFn: () => recruitApi.create({ project_id: selectedProject!.id, position, headcount, description: description.trim() || undefined, cooperation_type: coopType, contact_type: contactType, contact_info: contactType !== 'platform' ? contactInfo.trim() || undefined : undefined, expire_days: expireDays }),
    onSuccess: (data) => { toast.success('招募已发布'); navigate(`/recruit/${data.id}`) },
    onError: (e: any) => toast.error(e?.message || '发布失败'),
  })
  const updateMut = useMutation({
    mutationFn: () => recruitApi.update(recruitId!, { position, headcount, description: description.trim() || undefined, cooperation_type: coopType, contact_type: contactType, contact_info: contactType !== 'platform' ? contactInfo.trim() || undefined : undefined, expire_days: expireDays }),
    onSuccess: (data) => { toast.success('招募已更新'); navigate(`/recruit/${data.id}`) },
    onError: (e: any) => toast.error(e?.message || '更新失败'),
  })

  const isLoading = projectLoading || recruitLoading || myProjectsLoading
  const backUrl = isEdit ? `/recruit/${recruitId}` : slug ? `/p/${slug}` : '/recruit'
  const canSubmit = isEdit || !!selectedProject
  const isPending = createMut.isPending || updateMut.isPending
  const handleSubmit = () => {
    if (isEdit) {
      updateMut.mutate()
      return
    }
    if (!selectedProject) {
      toast.error((projectLoadError as { message?: string })?.message || '项目加载失败，无法发布招募')
      return
    }
    createMut.mutate()
  }

  return (
    <div className="max-w-2xl mx-auto px-6 py-8 animate-fade-in">
      <Link to={backUrl} className="inline-flex items-center gap-1.5 text-[13px] text-text-muted hover:text-amber transition-colors mb-6"><ArrowLeft className="w-4 h-4" /> {isEdit ? '返回招募' : '返回项目'}</Link>
      <h1 className="text-[22px] font-bold text-text-primary mb-1">{isEdit ? '编辑招募' : '发布招募'}</h1>
      <p className="text-[14px] text-text-muted mb-8">{isEdit ? `更新 ${recruit?.project_name || '项目'} 的招募信息` : `为 ${selectedProject?.name || '你的项目'} 寻找合适的团队成员`}</p>

      {projectError && !isEdit && (
        <div className="mb-5 rounded-xl border border-danger/20 bg-danger/5 px-4 py-3 text-[13px] text-danger">
          {(projectLoadError as { message?: string })?.message || '项目加载失败，请确认项目存在且后端服务正常。'}
        </div>
      )}
      {isLoading ? <div className="space-y-4"><Skeleton className="h-10 w-full" /><Skeleton className="h-10 w-full" /><Skeleton className="h-24 w-full" /></div> : (
        <div className="space-y-5">
          {!isEdit && !projectSlug && (
            <div>
              <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider block mb-2">选择项目</label>
              {myProjects.length > 0 ? (
                <div className="grid gap-2">
                  {myProjects.map((item) => (
                    <button key={item.id} onClick={() => setSelectedProjectId(item.id)}
                      className={cn('rounded-xl border px-4 py-3 text-left transition-all active:scale-[0.99]',
                        selectedProjectId === item.id ? 'border-amber bg-amber/10 text-amber' : 'border-white/[0.04] bg-white/[0.02] text-text-secondary hover:border-white/[0.08]')}>
                      <span className="block text-[14px] font-semibold">{item.name}</span>
                      <span className="mt-1 block text-[12px] text-text-muted">/{item.slug}</span>
                    </button>
                  ))}
                </div>
              ) : (
                <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-4">
                  <p className="text-[13px] text-text-secondary mb-3">你还没有可发布招募的项目。请先创建项目。</p>
                  <Link to="/projects/new"><Button size="sm">创建项目</Button></Link>
                </div>
              )}
            </div>
          )}
          <div>
            <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider block mb-2">选择岗位</label>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              {POSITIONS.map((p) => (
                <button key={p.value} onClick={() => setPosition(p.value)}
                  className={cn('flex flex-col items-center gap-2 p-4 rounded-xl border-2 transition-all duration-200 active:scale-95',
                    position === p.value ? 'border-amber bg-amber/[0.04]' : 'border-transparent bg-white/[0.02] hover:border-white/[0.06]')}>
                  <p.icon className="w-6 h-6" />
                  <span className={cn('text-[13px] font-semibold', position === p.value ? 'text-amber' : 'text-text-secondary')}>{p.label}</span>
                </button>
              ))}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">需求人数</label>
              <div className="flex flex-wrap gap-2">
                {HEADCOUNT_OPTIONS.map((n) => (
                  <button key={n} onClick={() => setHeadcount(n)}
                    className={cn('px-3 py-1.5 text-[13px] font-semibold rounded-lg border transition-all active:scale-90',
                      headcount === n ? 'border-amber bg-amber/10 text-amber' : 'border-white/[0.04] text-text-muted hover:border-white/[0.08]')}>{n}人</button>
                ))}
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">合作方式</label>
              <div className="flex flex-col gap-2">
                {COOP_TYPES.map((c) => (
                  <button key={c.value} onClick={() => setCoopType(c.value)}
                    className={cn('px-3 py-2 text-[13px] font-medium text-left rounded-lg border transition-all active:scale-95',
                      coopType === c.value ? 'border-amber bg-amber/10 text-amber' : 'border-white/[0.04] text-text-muted hover:border-white/[0.08]')}>{c.label}</button>
                ))}
              </div>
            </div>
          </div>
          <Textarea label="需求说明" placeholder="描述你需要的技能、经验要求、工作内容等..." value={description} onChange={(e) => setDescription(e.target.value)} rows={5} />
          <p className="text-right text-[11px] text-text-muted -mt-4">{description.length} 字</p>
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">联系方式</label>
            <div className="flex gap-2 flex-wrap">
              {CONTACT_OPTIONS.map((opt) => (
                <button key={opt.value} onClick={() => setContactType(opt.value)}
                  className={cn('px-4 py-2 text-[13px] font-medium rounded-xl border transition-all active:scale-95',
                    contactType === opt.value ? 'border-amber bg-amber/10 text-amber' : 'border-white/[0.04] text-text-muted hover:border-white/[0.08]')}>{opt.label}</button>
              ))}
            </div>
          </div>
          {contactType !== 'platform' && <Input label="联系方式详情" placeholder={contactType === 'wechat' ? '微信号' : 'QQ号'} value={contactInfo} onChange={(e) => setContactInfo(e.target.value)} />}
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">有效期</label>
            <div className="flex gap-2 flex-wrap">
              {EXPIRE_OPTIONS.map((opt) => (
                <button key={opt.value} onClick={() => setExpireDays(opt.value)}
                  className={cn('px-4 py-2 text-[13px] font-medium rounded-xl border transition-all active:scale-95',
                    expireDays === opt.value ? 'border-amber bg-amber/10 text-amber' : 'border-white/[0.04] text-text-muted hover:border-white/[0.08]')}>{opt.label}</button>
              ))}
            </div>
          </div>
          <div className="flex gap-3 pt-4">
            <Button variant="secondary" onClick={() => navigate(backUrl)}>取消</Button>
            <Button loading={isPending} onClick={handleSubmit} disabled={!canSubmit}>{isEdit ? '保存修改' : '发布招募'}</Button>
          </div>
        </div>
      )}
    </div>
  )
}

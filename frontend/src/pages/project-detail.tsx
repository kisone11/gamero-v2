import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Search, Plus, LayoutDashboard, Users, ClipboardList, ScrollText, Rocket, UserPlus, Flag, Archive, Radar, ClipboardCheck } from 'lucide-react'
import { projectApi } from '@/api/project'
import { recruitApi } from '@/api/recruit'
import { RecruitCard } from '@/components/recruit/RecruitCard'
import { useAuthStore } from '@/stores/authStore'
import { Button, Tabs, EmptyState, Skeleton, Card, Badge } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'
import {
  HeroSection, ProjectSidebar, ProjectOverviewTab,
  ProjectLogsTab, ProjectMembersTab, ProjectReviewsTab, ProjectTasksTab,
  ProjectMilestonesTab, ProjectResourcesTab, ProjectRiskRadarTab, ProjectQAGateTab,
} from '@/components/project'
import type { ProjectDetail } from '@/types/api'

const TABS = [
  { value: 'overview', label: '概览' },
  { value: 'dev-logs', label: '日志' },
  { value: 'members', label: '成员' },
  { value: 'tasks', label: '任务' },
  { value: 'milestones', label: '里程碑' },
  { value: 'resources', label: '资料库' },
  { value: 'risks', label: '风险雷达' },
  { value: 'qa-gate', label: '上线验收' },
  { value: 'recruit', label: '招募' },
  { value: 'reviews', label: '评测' },
]

function ProjectWorkspace({
  project,
  isOwner,
  onOpenTab,
}: {
  project: ProjectDetail
  isOwner: boolean
  onOpenTab: (tab: string) => void
}) {
  return (
    <Card hover={false} padding="lg" className="mb-5 bg-gradient-to-br from-amber/[0.05] to-white/[0.02] border-amber/10">
      <div className="flex items-start justify-between gap-4 mb-4">
        <div>
          <div className="flex items-center gap-2 mb-1.5">
            <LayoutDashboard className="w-4 h-4 text-amber" />
            <h2 className="text-[15px] font-semibold text-text-primary">项目工作台</h2>
            <Badge variant={isOwner ? 'success' : 'default'} size="sm">{isOwner ? '负责人' : '成员'}</Badge>
          </div>
          <p className="text-[12px] text-text-muted">集中处理任务、成员、版本、日志和招募，不影响下方原有详情内容。</p>
        </div>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-9 gap-2">
        <button onClick={() => onOpenTab('tasks')} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
          <ClipboardList className="w-3.5 h-3.5" />任务看板
        </button>
        <button onClick={() => onOpenTab('members')} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
          <Users className="w-3.5 h-3.5" />成员协作
        </button>
        <button onClick={() => onOpenTab('milestones')} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
          <Flag className="w-3.5 h-3.5" />里程碑
        </button>
        <button onClick={() => onOpenTab('resources')} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
          <Archive className="w-3.5 h-3.5" />资料库
        </button>
        <button onClick={() => onOpenTab('risks')} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
          <Radar className="w-3.5 h-3.5" />风险雷达
        </button>
        <button onClick={() => onOpenTab('qa-gate')} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
          <ClipboardCheck className="w-3.5 h-3.5" />上线验收
        </button>
        <Link to={`/project/${project.id}/releases`} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
          <Rocket className="w-3.5 h-3.5" />版本发布
        </Link>
        {isOwner ? (
          <Link to={`/projects/${project.id}/logs/new`} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-text-secondary hover:text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
            <ScrollText className="w-3.5 h-3.5" />写日志
          </Link>
        ) : (
          <button onClick={() => onOpenTab('dev-logs')} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-text-secondary hover:text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
            <ScrollText className="w-3.5 h-3.5" />看日志
          </button>
        )}
        {isOwner ? (
          <Link to={`/p/${project.slug}/recruit/new`} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-text-secondary hover:text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
            <UserPlus className="w-3.5 h-3.5" />发招募
          </Link>
        ) : (
          <button onClick={() => onOpenTab('recruit')} className="flex items-center justify-center gap-1.5 rounded-lg border border-white/[0.06] px-3 py-2 text-[12px] text-text-secondary hover:text-amber hover:border-amber/20 hover:bg-amber/[0.04] transition-colors">
            <UserPlus className="w-3.5 h-3.5" />看招募
          </button>
        )}
      </div>
    </Card>
  )
}

// ============================================================
// LoadingSkeleton
// ============================================================

function LoadingSkeleton() {
  return (
    <div>
      <div className="h-72 md:h-96 w-full bg-white/[0.02] animate-pulse" />
      <div className="max-w-6xl mx-auto px-6 py-6">
        <div className="grid grid-cols-12 gap-6">
          <main className="col-span-12 lg:col-span-9">
            <div className="h-10 w-full bg-white/[0.04] rounded animate-pulse mb-6" />
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3">
                  <div className="h-4 w-3/4 bg-white/[0.04] rounded animate-pulse" />
                  <div className="h-3 w-full bg-white/[0.04] rounded animate-pulse" />
                  <div className="h-3 w-2/3 bg-white/[0.04] rounded animate-pulse" />
                </div>
              ))}
            </div>
          </main>
          <aside className="col-span-12 lg:col-span-3">
            <div className="space-y-4">
              <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3">
                <div className="h-3 w-16 bg-white/[0.04] rounded animate-pulse" />
                <div className="h-4 w-full bg-white/[0.04] rounded animate-pulse" />
                <div className="h-4 w-full bg-white/[0.04] rounded animate-pulse" />
              </div>
              <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5 space-y-3">
                <div className="h-3 w-16 bg-white/[0.04] rounded animate-pulse" />
                <div className="h-4 w-full bg-white/[0.04] rounded animate-pulse" />
              </div>
            </div>
          </aside>
        </div>
      </div>
    </div>
  )
}

// ============================================================
// RecruitTab
// ============================================================

function RecruitTab({ project, isOwner }: { project: ProjectDetail; isOwner: boolean }) {
  const { data: data, isLoading } = useQuery({
    queryKey: ['project-recruits', project.id],
    queryFn: () => recruitApi.list({ project_id: project.id }),
  })
  const recruits = (data as any)?.list ?? []

  return (
    <div className="space-y-4">
      {isOwner && (
        <div className="flex items-center justify-between">
          <p className="text-[13px] text-text-muted">管理项目招募</p>
          <Link to={`/p/${project.slug}/recruit/new`}>
            <Button size="sm"><Plus className="w-3.5 h-3.5" /> 发布招募</Button>
          </Link>
        </div>
      )}
      {isLoading ? <Skeleton className="h-20 w-full" />
      : recruits.length === 0 ? <p className="text-[13px] text-text-muted py-8 text-center">暂无招募</p>
      : <div className="space-y-3">{recruits.map((r: any) => <RecruitCard key={r.id} item={r} />)}</div>}
    </div>
  )
}

// ============================================================
// ProjectDetailPage
// ============================================================

export default function ProjectDetailPage() {
  const { slug } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const user = useAuthStore(s => s.user)

  const [tab, setTab] = useState('overview')
  const [deleteProjectOpen, setDeleteProjectOpen] = useState(false)

  const projectQuery = useQuery({
    queryKey: ['project', slug],
    queryFn: () => projectApi.get(slug!),
    enabled: !!slug,
  })

  const project = projectQuery.data
  const isOwner = !!(user && project?.owner_id === user.id)
  const isProjectMember = !!(user && project?.members?.some((member) => member.user_id === user.id))

  // ── Mutations ──

  const followMutation = useMutation({
    mutationFn: () =>
      project!.is_followed ? projectApi.unfollow(project!.id) : projectApi.follow(project!.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', slug] })
      toast.success(project!.is_followed ? '已取消关注' : '已关注项目')
    },
    onError: () => toast.error('操作失败，请重试'),
  })

  const deleteProjectMutation = useMutation({
    mutationFn: () => projectApi.delete(project!.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      toast.success('项目已删除')
      navigate('/projects')
    },
    onError: () => toast.error('删除失败'),
  })

  // ── Handlers ──

  const handleShare = () => {
    if (!project) return
    if (navigator.share) {
      navigator.share({ title: project.name, url: window.location.href }).catch(() => {})
    } else {
      navigator.clipboard.writeText(window.location.href).then(
        () => toast.success('链接已复制'),
        () => toast.error('复制失败'),
      )
    }
  }

  // ── Loading ──
  if (projectQuery.isLoading) {
    return <LoadingSkeleton />
  }

  // ── Error ──
  if (projectQuery.isError) {
    return (
      <div className="max-w-6xl mx-auto px-6 py-6 min-h-[60vh] flex items-center justify-center">
        <EmptyState
          icon={<AlertTriangle className="w-7 h-7 text-text-muted" />}
          title="加载失败"
          description="无法加载项目信息，请检查网络后重试"
          action={{ label: '重新加载', onClick: () => projectQuery.refetch() }}
        />
      </div>
    )
  }

  // ── Not found ──
  if (!project) {
    return (
      <div className="max-w-6xl mx-auto px-6 py-6 min-h-[60vh] flex items-center justify-center">
        <EmptyState
          icon={<Search className="w-7 h-7 text-text-muted" />}
          title="项目不存在"
          description="该项目可能已被删除或链接地址有误"
          action={{ label: '返回首页', onClick: () => navigate('/') }}
        />
      </div>
    )
  }

  // ── Render ──
  return (
    <div className="min-h-screen bg-surface-void">
      <HeroSection
        project={project}
        isOwner={isOwner}
        isFollowed={project.is_followed}
        onFollow={() => followMutation.mutate()}
        onShare={handleShare}
        onEdit={() => navigate(`/p/${project.slug}/edit`)}
        onDelete={() => setDeleteProjectOpen(true)}
        followLoading={followMutation.isPending}
        currentUser={user}
      />

      <div className="max-w-6xl mx-auto px-6 py-6">
        <div className="grid grid-cols-12 gap-6">
          <main className="col-span-12 lg:col-span-9">
            {(isOwner || isProjectMember) && (
              <ProjectWorkspace project={project} isOwner={isOwner} onOpenTab={setTab} />
            )}
            <Tabs tabs={TABS} value={tab} onChange={setTab} />

            {tab === 'overview' && <ProjectOverviewTab project={project} />}
            {tab === 'dev-logs' && (
              <ProjectLogsTab
                projectId={project.id}
                isOwner={isOwner}
                projectSlug={project.slug}
              />
            )}
            {tab === 'members' && (
              <ProjectMembersTab
                members={project.members}
                projectId={project.id}
                isOwner={isOwner}
                currentUser={user}
              />
            )}
            {tab === 'tasks' && <ProjectTasksTab projectId={project.id} isOwner={isOwner} />}
            {tab === 'milestones' && <ProjectMilestonesTab projectId={project.id} isOwner={isOwner} />}
            {tab === 'resources' && <ProjectResourcesTab projectId={project.id} isOwner={isOwner} currentUserId={user?.id} />}
            {tab === 'risks' && <ProjectRiskRadarTab projectId={project.id} isOwner={isOwner} currentUserId={user?.id} />}
            {tab === 'qa-gate' && <ProjectQAGateTab projectId={project.id} isOwner={isOwner} currentUserId={user?.id} />}
            {tab === 'recruit' && <RecruitTab project={project} isOwner={isOwner} />}
            {tab === 'reviews' && (
              <ProjectReviewsTab
                projectId={project.id}
                currentUser={user}
                isOwner={isOwner}
              />
            )}
          </main>

          <aside className="col-span-12 lg:col-span-3">
            <ProjectSidebar project={project} />
          </aside>
        </div>
      </div>

      {/* Delete project confirmation dialog */}
      <Dialog open={deleteProjectOpen} onOpenChange={setDeleteProjectOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>删除项目</DialogTitle><DialogDescription>确定要删除这个项目吗？<strong className="text-danger">此操作不可撤销。</strong></DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteProjectOpen(false)}>取消</Button>
            <Button variant="danger" loading={deleteProjectMutation.isPending} onClick={() => {
              deleteProjectMutation.mutate()
              setDeleteProjectOpen(false)
            }}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  Bell,
  ClipboardCheck,
  Gamepad2,
  Inbox,
  LayoutDashboard,
  Plus,
  ScrollText,
  Send,
  Sparkles,
  Users,
  Zap,
} from 'lucide-react'
import { projectApi } from '@/api/project'
import { recruitApi } from '@/api/recruit'
import { talentApi } from '@/api/talent'
import { notificationApi } from '@/api/notification'
import { logApi } from '@/api/log'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { timeAgo } from '@/lib/time'
import type { ApplicationDetail, InvitationDetail, Notification, ProjectListItem, DevLogDetail } from '@/types/api'

function MetricCard({ icon: Icon, label, value, href }: { icon: React.ComponentType<{ className?: string }>; label: string; value: number; href: string }) {
  return (
    <Link to={href}>
      <Card padding="lg" className="h-full">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-[12px] text-text-muted mb-1">{label}</p>
            <p className="text-[28px] font-bold text-text-primary font-mono">{value}</p>
          </div>
          <div className="w-11 h-11 rounded-xl bg-amber/10 flex items-center justify-center ring-1 ring-amber/20">
            <Icon className="w-5 h-5 text-amber" />
          </div>
        </div>
      </Card>
    </Link>
  )
}

function MiniList<T>({ title, empty, items, renderItem, action }: {
  title: string
  empty: string
  items: T[]
  renderItem: (item: T) => React.ReactNode
  action?: React.ReactNode
}) {
  return (
    <Card padding="lg" hover={false}>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-[15px] font-semibold text-text-primary">{title}</h2>
        {action}
      </div>
      {items.length === 0 ? (
        <div className="py-10 text-center">
          <Inbox className="w-6 h-6 mx-auto mb-2 text-text-muted/50" />
          <p className="text-[13px] text-text-muted">{empty}</p>
        </div>
      ) : (
        <div className="space-y-3">{items.map(renderItem)}</div>
      )}
    </Card>
  )
}

export default function CollaborationCenterPage() {
  const user = useAuthStore((s) => s.user)

  const myProjectsQuery = useQuery({
    queryKey: ['collab-center-projects', user?.id],
    queryFn: () => projectApi.list({ page: 1, page_size: 8, participant_id: user!.id }),
    enabled: !!user,
  })
  const ownedProjectsQuery = useQuery({
    queryKey: ['collab-center-owned-projects', user?.id],
    queryFn: () => projectApi.list({ page: 1, page_size: 8, owner_id: user!.id }),
    enabled: !!user,
  })
  const applicationsQuery = useQuery({
    queryKey: ['collab-center-applications', user?.id],
    queryFn: () => recruitApi.listMyApplications(1, 8),
    enabled: !!user,
  })
  const invitationsQuery = useQuery({
    queryKey: ['collab-center-invitations', user?.id],
    queryFn: () => talentApi.listMyInvitations(1, 8),
    enabled: !!user,
  })
  const notificationsQuery = useQuery({
    queryKey: ['collab-center-notifications', user?.id],
    queryFn: () => notificationApi.list(1, 6),
    enabled: !!user,
  })
  const logsQuery = useQuery({
    queryKey: ['collab-center-logs', user?.id],
    queryFn: () => logApi.list({ author_id: user!.id, page: 1, page_size: 6, sort: 'latest' }),
    enabled: !!user,
  })

  if (!user) {
    return (
      <div className="max-w-4xl mx-auto px-6 py-20 flex flex-col items-center justify-center text-center">
        <Zap className="w-8 h-8 text-text-muted mb-4" />
        <p className="text-[16px] font-semibold text-text-primary mb-2">请先登录</p>
        <p className="text-[13px] text-text-muted mb-6">协作中心需要登录后查看</p>
        <Link to="/login"><Button>去登录</Button></Link>
      </div>
    )
  }

  const projects = myProjectsQuery.data?.list ?? []
  const ownedProjects = ownedProjectsQuery.data?.list ?? []
  const applications = applicationsQuery.data?.list ?? []
  const invitations = invitationsQuery.data?.list ?? []
  const notifications = notificationsQuery.data?.list ?? []
  const logs = logsQuery.data?.list ?? []
  const pendingApplications = applications.filter((app) => app.status === 'pending').length
  const pendingInvitations = invitations.filter((inv) => inv.status === 'pending').length

  return (
    <div className="max-w-[1180px] mx-auto px-6 py-8">
      <div className="flex items-start justify-between gap-4 mb-7">
        <div>
          <div className="flex items-center gap-2 mb-2">
            <LayoutDashboard className="w-5 h-5 text-amber" />
            <span className="text-[12px] text-amber font-semibold uppercase tracking-wider">Collaboration Hub</span>
          </div>
          <h1 className="text-[26px] font-bold text-text-primary">我的协作中心</h1>
          <p className="text-[13px] text-text-muted mt-1">聚合项目、申请、邀请、日志和通知，不改变现有业务流程。</p>
        </div>
        <div className="flex gap-2 shrink-0">
          <Link to="/projects/new"><Button size="sm"><Plus className="w-3.5 h-3.5" />创建项目</Button></Link>
          <Link to="/talent"><Button variant="secondary" size="sm"><Users className="w-3.5 h-3.5" />找人才</Button></Link>
        </div>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <MetricCard icon={Gamepad2} label="参与/创建项目" value={myProjectsQuery.data?.total ?? projects.length} href="/projects?mine=1" />
        <MetricCard icon={Sparkles} label="我负责的项目" value={ownedProjectsQuery.data?.total ?? ownedProjects.length} href="/projects?owner=me" />
        <MetricCard icon={ClipboardCheck} label="申请处理中" value={pendingApplications} href="/me/applications" />
        <MetricCard icon={Send} label="待处理邀请" value={pendingInvitations} href="/me/invitations" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
        <MiniList<ProjectListItem>
          title="我的项目"
          empty="还没有参与或创建项目"
          items={projects.slice(0, 5)}
          action={<Link to="/projects?mine=1" className="text-[12px] text-amber hover:underline">查看全部</Link>}
          renderItem={(project) => (
            <Link key={project.id} to={`/p/${project.slug || project.id}`} className="block rounded-lg border border-white/[0.04] p-3 hover:border-amber/20 transition-colors">
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="text-[14px] font-semibold text-text-primary truncate">{project.name}</p>
                  <p className="text-[12px] text-text-muted truncate">{project.description || '暂无简介'}</p>
                </div>
                <Badge variant={project.owner_id === user.id ? 'success' : 'default'} size="sm">{project.owner_id === user.id ? '负责' : '参与'}</Badge>
              </div>
            </Link>
          )}
        />

        <MiniList<ApplicationDetail>
          title="我的申请"
          empty="暂无申请记录"
          items={applications.slice(0, 5)}
          action={<Link to="/me/applications" className="text-[12px] text-amber hover:underline">查看全部</Link>}
          renderItem={(app) => (
            <Link key={app.id} to={`/recruit/${app.recruitment_id}`} className="block rounded-lg border border-white/[0.04] p-3 hover:border-amber/20 transition-colors">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="text-[14px] font-semibold text-text-primary">{app.project_name || `招募 #${app.recruitment_id}`}</p>
                  <p className="text-[12px] text-text-muted">{timeAgo(app.created_at)}</p>
                </div>
                <Badge variant={app.status === 'approved' ? 'success' : app.status === 'pending' ? 'warning' : 'default'} size="sm">
                  {app.status === 'pending' ? '审核中' : app.status === 'approved' ? '已通过' : app.status === 'rejected' ? '未通过' : '已撤销'}
                </Badge>
              </div>
            </Link>
          )}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <MiniList<InvitationDetail>
          title="我的邀请"
          empty="暂无收到的邀请"
          items={invitations.slice(0, 4)}
          action={<Link to="/me/invitations" className="text-[12px] text-amber hover:underline">查看全部</Link>}
          renderItem={(inv) => (
            <Link key={inv.id} to="/me/invitations" className="block rounded-lg border border-white/[0.04] p-3 hover:border-amber/20 transition-colors">
              <p className="text-[14px] font-semibold text-text-primary truncate">{inv.project_name || `项目 #${inv.project_id}`}</p>
              <p className="text-[12px] text-text-muted mt-1">{timeAgo(inv.created_at)}</p>
            </Link>
          )}
        />

        <MiniList<DevLogDetail>
          title="我的日志"
          empty="暂无开发日志"
          items={logs.slice(0, 4)}
          action={<Link to="/me/logs" className="text-[12px] text-amber hover:underline">查看全部</Link>}
          renderItem={(log) => (
            <Link key={log.id} to={`/devlog/${log.id}`} className="block rounded-lg border border-white/[0.04] p-3 hover:border-amber/20 transition-colors">
              <div className="flex items-center gap-2 mb-1">
                <ScrollText className="w-3.5 h-3.5 text-amber" />
                <p className="text-[14px] font-semibold text-text-primary truncate">{log.title || '无标题'}</p>
              </div>
              <p className="text-[12px] text-text-muted">{timeAgo(log.created_at)}</p>
            </Link>
          )}
        />

        <MiniList<Notification>
          title="最近通知"
          empty="暂无通知"
          items={notifications.slice(0, 4)}
          action={<Link to="/notifications" className="text-[12px] text-amber hover:underline">查看全部</Link>}
          renderItem={(notif) => (
            <Link key={notif.id} to="/notifications" className="block rounded-lg border border-white/[0.04] p-3 hover:border-amber/20 transition-colors">
              <div className="flex items-start gap-2">
                <Bell className="w-3.5 h-3.5 mt-0.5 text-amber" />
                <div className="min-w-0">
                  <p className="text-[14px] font-semibold text-text-primary truncate">{notif.title}</p>
                  <p className="text-[12px] text-text-muted truncate">{notif.content}</p>
                </div>
              </div>
            </Link>
          )}
        />
      </div>
    </div>
  )
}

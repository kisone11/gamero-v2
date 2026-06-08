import { useQuery } from '@tanstack/react-query'
import { Activity, AlertTriangle, ArrowRight, BarChart3, Clock, FileText, Flag, Gamepad2, ShieldCheck, Users, Zap } from 'lucide-react'
import { adminApi } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { timeAgo } from '@/lib/time'
import { StatSkeleton } from './AdminSkeletons'
import type { AdminDashboardAction, AdminDashboardMetric, AuditLog, DailyStats } from '@/types/api'

const severityClass: Record<string, string> = {
  good: 'text-success bg-success/10 border-success/20',
  info: 'text-text-secondary bg-white/[0.04] border-white/[0.08]',
  warning: 'text-amber bg-amber/10 border-amber/20',
  danger: 'text-danger bg-danger/10 border-danger/20',
}

const healthCopy = {
  healthy: { label: '运行健康', desc: '核心运营指标稳定，暂无高优先级风险。', variant: 'success' as const },
  watch: { label: '需要关注', desc: '存在待处理事项，建议今天完成巡检。', variant: 'warning' as const },
  risk: { label: '高风险', desc: '风险项较多，请优先处理举报和封禁复核。', variant: 'danger' as const },
}

function metricIcon(key: string) {
  if (key.includes('report')) return Flag
  if (key.includes('user')) return Users
  if (key.includes('project')) return Gamepad2
  if (key.includes('recruitment') || key.includes('application')) return Zap
  return Activity
}

function GrowthChart({ days }: { days: DailyStats[] }) {
  const maxValue = Math.max(1, ...days.flatMap((day) => [day.new_users, day.active_users, day.new_projects, day.new_logs, day.new_posts]))
  return <Card padding="lg" hover={false} className="overflow-x-auto">
    <div className="flex items-center justify-between mb-5">
      <div>
        <h3 className="text-[15px] font-semibold text-text-primary flex items-center gap-2"><BarChart3 className="w-4 h-4 text-amber" />增长趋势</h3>
        <p className="text-[12px] text-text-muted mt-1">用户、项目、日志、帖子和活跃度的 7 日组合视图</p>
      </div>
      <Badge variant="default">7 days</Badge>
    </div>
    <div className="min-w-[680px]">
      <div className="flex items-end gap-3 h-52 border-b border-white/[0.06] pb-3">
        {days.map((day) => (
          <div key={day.date} className="flex-1 flex flex-col items-center gap-2">
            <div className="w-full flex items-end justify-center gap-1 h-36">
              {[
                { value: day.new_users, color: 'bg-amber', title: '新增用户' },
                { value: day.active_users, color: 'bg-amber-light', title: '活跃用户' },
                { value: day.new_projects, color: 'bg-amber-dim', title: '新增项目' },
                { value: day.new_logs, color: 'bg-white/35', title: '新增日志' },
                { value: day.new_posts, color: 'bg-white/20', title: '新增帖子' },
              ].map((bar) => <div key={bar.title} className="group relative w-2.5 rounded-t bg-white/[0.04] overflow-hidden" style={{ height: '100%' }}>
                <div className={`${bar.color} absolute bottom-0 left-0 right-0 rounded-t`} style={{ height: `${Math.max(4, (bar.value / maxValue) * 100)}%` }} />
                <span className="pointer-events-none absolute -top-8 left-1/2 -translate-x-1/2 whitespace-nowrap rounded bg-surface-void px-2 py-1 text-[10px] text-text-secondary opacity-0 shadow-lg group-hover:opacity-100">{bar.title}: {bar.value}</span>
              </div>)}
            </div>
            <span className="text-[10px] text-text-muted font-mono">{day.date.slice(5)}</span>
          </div>
        ))}
      </div>
      <div className="flex flex-wrap gap-3 mt-3 text-[11px] text-text-muted">
        <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-amber" />新增用户</span>
        <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-amber-light" />活跃用户</span>
        <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-amber-dim" />新增项目</span>
        <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-white/35" />新增日志</span>
        <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-white/20" />新增帖子</span>
      </div>
    </div>
  </Card>
}

function RiskMetricCard({ metric }: { metric: AdminDashboardMetric }) {
  const Icon = metricIcon(metric.key)
  return <Card padding="md" hover={false} className="border-white/[0.06] bg-surface-card/90">
    <div className="flex items-center gap-3">
      <div className={`w-10 h-10 rounded-lg border flex items-center justify-center ${severityClass[metric.severity] ?? severityClass.info}`}><Icon className="w-5 h-5" /></div>
      <div className="min-w-0">
        <p className="text-[12px] text-text-muted">{metric.label}</p>
        <p className="text-[22px] font-bold text-text-primary tabular-nums">{metric.value.toLocaleString()}</p>
      </div>
    </div>
  </Card>
}

function ActionCard({ action, onOpenTab }: { action: AdminDashboardAction; onOpenTab?: (tab: string) => void }) {
  return <Card padding="md" hover={false} className="border-white/[0.06]">
    <div className="flex items-start justify-between gap-4">
      <div className="min-w-0">
        <div className="flex items-center gap-2 mb-1.5"><span className={`rounded-full border px-2 py-0.5 text-[11px] font-semibold ${severityClass[action.severity] ?? severityClass.info}`}>{action.count.toLocaleString()}</span><p className="text-[14px] font-semibold text-text-primary">{action.title}</p></div>
        <p className="text-[12px] text-text-muted line-clamp-2">{action.description}</p>
      </div>
      <Button variant="ghost" size="sm" onClick={() => onOpenTab?.(action.target_tab)}><ArrowRight className="w-3.5 h-3.5" /></Button>
    </div>
  </Card>
}

export function OverviewTab({ onOpenTab }: { onOpenTab?: (tab: string) => void }) {
  const { data, isLoading, isError, error, refetch } = useQuery({ queryKey: ['admin', 'dashboard'], queryFn: () => adminApi.getDashboardOverview(), retry: 1 })
  const errorMessage = typeof error === 'object' && error !== null && 'message' in error ? String((error as { message?: unknown }).message) : '无法加载运营聚合数据'

  if (isLoading) return <div className="space-y-5"><div className="grid grid-cols-2 md:grid-cols-4 gap-3">{Array.from({ length: 8 }).map((_, i) => <StatSkeleton key={i} />)}</div><Skeleton className="h-72 rounded-2xl" /><Skeleton className="h-56 rounded-2xl" /></div>
  if (isError || !data) return <div className="py-16 flex flex-col items-center justify-center text-center"><div className="w-16 h-16 mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center"><AlertTriangle className="w-7 h-7 text-text-muted" /></div><p className="text-h3 text-text-secondary mb-1">驾驶舱加载失败</p><p className="text-body text-text-muted mb-6">{errorMessage}</p><Button variant="secondary" size="sm" onClick={() => refetch()}>重新加载</Button></div>

  const health = healthCopy[data.health_level] ?? healthCopy.watch
  const stats = data.stats
  const topStats = [
    { label: '总用户', value: stats.total_users, icon: Users },
    { label: '活跃用户', value: stats.active_users, icon: Activity },
    { label: '项目总数', value: stats.total_projects, icon: Gamepad2 },
    { label: '内容总量', value: stats.total_posts + stats.total_logs, icon: FileText },
  ]

  return <div className="space-y-6">
    <Card padding="lg" hover={false} className="relative overflow-hidden border-white/[0.06] bg-surface-card/95">
      <div className="relative grid gap-6 lg:grid-cols-[0.95fr_1.25fr]">
        <div>
          <div className="flex items-center gap-2 mb-3"><ShieldCheck className="w-5 h-5 text-amber" /><Badge variant={health.variant}>{health.label}</Badge><span className="text-[11px] text-text-muted font-mono">更新于 {new Date(data.generated_at).toLocaleTimeString('zh-CN')}</span></div>
          <div className="flex items-end gap-4 mb-3"><p className="text-[52px] leading-none font-black text-text-primary tabular-nums">{data.health_score}</p><span className="text-[13px] text-text-muted mb-2">/ 100 运营健康分</span></div>
          <p className="text-[14px] text-text-secondary max-w-xl">{health.desc}</p>
          <div className="mt-5 flex flex-wrap gap-2"><Button size="sm" onClick={() => onOpenTab?.('reports')}>处理举报</Button><Button size="sm" variant="secondary" onClick={() => onOpenTab?.('audit')}>查看审计</Button><Button size="sm" variant="ghost" onClick={() => onOpenTab?.('content')}>内容巡检</Button></div>
        </div>
        <div className="grid content-start grid-cols-2 gap-3 xl:grid-cols-4">
          {topStats.map((item) => {
            const Icon = item.icon
            return <Card key={item.label} padding="md" hover={false} className="h-fit bg-white/[0.025] border-white/[0.06]"><div className="flex items-center gap-3"><div className="w-9 h-9 rounded-lg bg-amber/10 flex items-center justify-center"><Icon className="w-4 h-4 text-amber" /></div><div><p className="text-[11px] text-text-muted">{item.label}</p><p className="text-[20px] font-bold text-text-primary tabular-nums">{item.value.toLocaleString()}</p></div></div></Card>
          })}
        </div>
      </div>
    </Card>

    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">{data.risk_metrics.map((metric) => <RiskMetricCard key={metric.key} metric={metric} />)}</div>

    <div className="grid gap-6 lg:grid-cols-[1fr_0.9fr]">
      <div>
        <h3 className="text-[15px] font-semibold text-text-primary mb-3 flex items-center gap-2"><Zap className="w-4 h-4 text-amber" />今日运营待办</h3>
        <div className="space-y-3">{data.pending_actions.map((action) => <ActionCard key={action.key} action={action} onOpenTab={onOpenTab} />)}</div>
      </div>
      <div>
        <h3 className="text-[15px] font-semibold text-text-primary mb-3 flex items-center gap-2"><Clock className="w-4 h-4 text-amber" />最近敏感操作</h3>
        <Card padding="md" hover={false}>
          <div className="space-y-2">
            {data.recent_audit_logs.length === 0 ? <p className="text-[13px] text-text-muted text-center py-8">暂无审计记录</p> : data.recent_audit_logs.map((log: AuditLog) => <div key={log.id} className="flex items-center gap-3 py-2 border-b border-white/[0.03] last:border-0"><Badge variant="default" size="sm" className="shrink-0">{log.action}</Badge><div className="min-w-0 flex-1"><p className="text-[12px] text-text-secondary truncate">{log.target_type} #{log.target_id}</p>{log.note && <p className="text-[11px] text-text-muted truncate">{log.note}</p>}</div><span className="text-[11px] text-text-muted shrink-0">{timeAgo(log.created_at)}</span></div>)}
          </div>
        </Card>
      </div>
    </div>

    <GrowthChart days={data.daily_stats} />
  </div>
}

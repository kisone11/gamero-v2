import { useQuery } from '@tanstack/react-query'
import { BarChart3, Clock, AlertTriangle, Users, UserCheck, FileText, Gamepad2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { timeAgo } from '@/lib/time'
import { StatSkeleton } from './AdminSkeletons'
import type { DailyStats, AuditLog, PlatformStats } from '@/types/api'

const STAT_ITEMS: { key: keyof PlatformStats; label: string; Icon: React.ElementType }[] = [
  { key: 'total_users', label: '总用户数', Icon: Users },
  { key: 'active_users', label: '活跃用户', Icon: UserCheck },
  { key: 'total_projects', label: '项目总数', Icon: Gamepad2 },
  { key: 'total_logs', label: '日志总数', Icon: FileText },
  { key: 'total_posts', label: '帖子总数', Icon: FileText },
  { key: 'new_users_today', label: '今日新增用户', Icon: Users },
  { key: 'new_projects_today', label: '今日新增项目', Icon: Gamepad2 },
]

export function OverviewTab() {
  const { data: stats, isLoading: statsLoading, isError: statsError, refetch: refetchStats } = useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: () => adminApi.getPlatformStats(),
  })

  const { data: dailyStats, isLoading: dailyLoading, isError: dailyError } = useQuery({
    queryKey: ['admin', 'daily-stats'],
    queryFn: () => adminApi.getDailyStats(7),
    retry: false,
  })

  const { data: auditData, isLoading: auditLoading } = useQuery({
    queryKey: ['admin', 'audit-logs', 'recent'],
    queryFn: () => adminApi.getAuditLogs({ page: 1, page_size: 10 }),
  })

  const auditLogs = auditData?.list ?? []
  const maxDailyValue = Math.max(
    1,
    ...(dailyStats ?? []).flatMap((day) => [day.new_users, day.active_users, day.new_projects, day.new_logs, day.new_posts]),
  )

  return (
    <div className="space-y-6">
      {/* Stats grid */}
      {statsLoading ? (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {Array.from({ length: 7 }).map((_, i) => <StatSkeleton key={i} />)}
        </div>
      ) : statsError ? (
        <div className="py-12 flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <AlertTriangle className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-h3 text-text-secondary mb-1">加载失败</p>
          <p className="text-body text-text-muted mb-6">无法加载统计数据</p>
          <Button variant="secondary" size="sm" onClick={() => refetchStats()}>重新加载</Button>
        </div>
      ) : stats ? (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {STAT_ITEMS.map((item) => {
            const value = stats[item.key] ?? 0
            const Icon = item.Icon
            return (
              <Card key={item.key} padding="md" hover={false}>
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-white/[0.04] flex items-center justify-center shrink-0">
                    <Icon className="w-5 h-5 text-text-muted" />
                  </div>
                  <div>
                    <p className="text-caption text-text-muted">{item.label}</p>
                    <p className="text-[20px] font-bold text-text-primary mt-0.5 tabular-nums">
                      {value.toLocaleString()}
                    </p>
                  </div>
                </div>
              </Card>
            )
          })}
        </div>
      ) : null}

      {/* Daily Stats */}
      <div>
        <h3 className="text-h3 text-text-primary mb-3 flex items-center gap-2">
          <BarChart3 className="w-4 h-4 text-text-muted" />
          近 7 日数据
        </h3>
        {dailyLoading ? (
          <Card padding="md" hover={false}>
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-4 w-full" />
              ))}
            </div>
          </Card>
        ) : dailyError ? (
          <Card padding="md" hover={false}>
            <p className="text-body text-text-muted text-center py-4">每日统计数据暂时无法加载</p>
          </Card>
        ) : dailyStats && dailyStats.length > 0 ? (
          <Card padding="md" hover={false} className="overflow-x-auto space-y-6">
            <div className="min-w-[640px]">
              <div className="flex items-end gap-3 h-56 border-b border-white/[0.06] pb-3">
                {dailyStats.map((day: DailyStats) => (
                  <div key={day.date} className="flex-1 flex flex-col items-center gap-2">
                    <div className="w-full flex items-end justify-center gap-1 h-40">
                      {[
                        { value: day.new_users, color: 'bg-amber', title: '新增用户' },
                        { value: day.active_users, color: 'bg-success', title: '活跃用户' },
                        { value: day.new_projects, color: 'bg-cyan', title: '新增项目' },
                        { value: day.new_logs, color: 'bg-coral', title: '新增日志' },
                        { value: day.new_posts, color: 'bg-purple-400', title: '新增帖子' },
                      ].map((bar) => (
                        <div key={bar.title} className="group relative w-2.5 rounded-t bg-white/[0.04] overflow-hidden" style={{ height: '100%' }}>
                          <div className={`${bar.color} absolute bottom-0 left-0 right-0 rounded-t transition-all`} style={{ height: `${Math.max(4, (bar.value / maxDailyValue) * 100)}%` }} />
                          <span className="pointer-events-none absolute -top-8 left-1/2 -translate-x-1/2 whitespace-nowrap rounded bg-surface-void px-2 py-1 text-[10px] text-text-secondary opacity-0 shadow-lg group-hover:opacity-100">
                            {bar.title}: {bar.value}
                          </span>
                        </div>
                      ))}
                    </div>
                    <span className="text-[10px] text-text-muted font-mono">{day.date.slice(5)}</span>
                  </div>
                ))}
              </div>
              <div className="flex flex-wrap gap-3 mt-3 text-[11px] text-text-muted">
                <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-amber" />新增用户</span>
                <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-success" />活跃用户</span>
                <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-cyan" />新增项目</span>
                <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-coral" />新增日志</span>
                <span className="flex items-center gap-1.5"><i className="w-2.5 h-2.5 rounded bg-purple-400" />新增帖子</span>
              </div>
            </div>
            <table className="w-full text-body">
              <thead>
                <tr className="text-text-muted border-b border-white/[0.04]">
                  <th className="text-left py-2 pr-4 font-medium">日期</th>
                  <th className="text-right px-3 py-2 font-medium">新增用户</th>
                  <th className="text-right px-3 py-2 font-medium">活跃用户</th>
                  <th className="text-right px-3 py-2 font-medium">新增项目</th>
                  <th className="text-right px-3 py-2 font-medium">新增日志</th>
                  <th className="text-right pl-3 py-2 font-medium">新增帖子</th>
                </tr>
              </thead>
              <tbody>
                {dailyStats.map((day: DailyStats) => (
                  <tr key={day.date} className="border-b border-white/[0.02] last:border-0">
                    <td className="py-2.5 pr-4 text-text-primary whitespace-nowrap">{day.date}</td>
                    <td className="py-2.5 px-3 text-right text-text-secondary tabular-nums">{day.new_users}</td>
                    <td className="py-2.5 px-3 text-right text-text-secondary tabular-nums">{day.active_users}</td>
                    <td className="py-2.5 px-3 text-right text-text-secondary tabular-nums">{day.new_projects}</td>
                    <td className="py-2.5 px-3 text-right text-text-secondary tabular-nums">{day.new_logs}</td>
                    <td className="py-2.5 pl-3 text-right text-text-secondary tabular-nums">{day.new_posts}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        ) : (
          <Card padding="md" hover={false}>
            <p className="text-body text-text-muted text-center py-4">暂无日统计数据</p>
          </Card>
        )}
      </div>

      {/* Recent Audit Logs */}
      <div>
        <h3 className="text-h3 text-text-primary mb-3 flex items-center gap-2">
          <Clock className="w-4 h-4 text-text-muted" />
          最近操作记录
        </h3>
        {auditLoading ? (
          <Card padding="md" hover={false}>
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-4 w-full" />
              ))}
            </div>
          </Card>
        ) : auditLogs.length > 0 ? (
          <Card padding="md" hover={false}>
            <div className="space-y-2">
              {auditLogs.map((log: AuditLog) => (
                <div key={log.id} className="flex items-center gap-3 text-body py-1.5 border-b border-white/[0.02] last:border-0">
                  <span className="text-text-muted shrink-0 font-mono text-caption">#{log.id}</span>
                  <Badge variant="default" size="sm" className="shrink-0">{log.action}</Badge>
                  <span className="text-text-secondary shrink-0">{log.target_type}</span>
                  <span className="text-text-muted font-mono text-caption">ID: {log.target_id}</span>
                  {log.note && <span className="text-text-muted truncate">{log.note}</span>}
                  <span className="text-text-muted ml-auto shrink-0 text-caption">{timeAgo(log.created_at)}</span>
                </div>
              ))}
            </div>
          </Card>
        ) : (
          <Card padding="md" hover={false}>
            <p className="text-body text-text-muted text-center py-4">暂无操作记录</p>
          </Card>
        )}
      </div>
    </div>
  )
}

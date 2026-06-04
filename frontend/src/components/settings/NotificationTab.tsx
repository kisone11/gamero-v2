import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as Switch from '@radix-ui/react-switch'
import { notificationApi } from '@/api/notification'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/stores/toastStore'
import { cn } from '@/lib/utils'
import { Card } from '@/components/ui/card'

// ============================================================
// Constants
// ============================================================

const NOTIF_TYPE_GROUPS: { group: string; label: string; items: { type: string; label: string }[] }[] = [
  {
    group: 'social',
    label: '社交',
    items: [
      { type: 'mention', label: '提及我' },
      { type: 'new_follower', label: '新粉丝' },
      { type: 'post_liked', label: '帖子被赞' },
      { type: 'post_commented', label: '帖子被评论' },
      { type: 'log_liked', label: '开发日志被赞' },
      { type: 'log_commented', label: '开发日志被评论' },
    ],
  },
  {
    group: 'project',
    label: '项目',
    items: [
      { type: 'recruitment_applied', label: '招募申请' },
      { type: 'application_approved', label: '申请通过' },
      { type: 'application_rejected', label: '申请被拒' },
      { type: 'project_status_changed', label: '项目状态变更' },
      { type: 'talent_invited', label: '被邀请加入项目' },
      { type: 'invite_accepted', label: '邀请被接受' },
      { type: 'invite_declined', label: '邀请被拒绝' },
      { type: 'project_banned', label: '项目被禁用' },
      { type: 'project_unbanned', label: '项目解禁' },
    ],
  },
  {
    group: 'system',
    label: '系统',
    items: [
      { type: 'system', label: '系统通知' },
      { type: 'report_handled', label: '举报处理结果' },
    ],
  },
]

// ============================================================
// NotificationTab
// ============================================================

export function NotificationTab() {
  const queryClient = useQueryClient()

  const { data: preferences, isLoading } = useQuery({
    queryKey: ['notification-preferences'],
    queryFn: () => notificationApi.getPreferences(),
  })

  const prefMap = new Map<string, boolean>(
    (preferences?.preferences ?? []).map((p) => [p.type, p.enabled])
  )

  const updateMut = useMutation({
    mutationFn: ({ type, enabled }: { type: string; enabled: boolean }) =>
      notificationApi.updatePreference(type, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notification-preferences'] })
    },
    onError: (err: any) => toast.error(err?.message || '更新失败'),
  })

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full rounded-xl" />
        <Skeleton className="h-10 w-full rounded-xl" />
        <Skeleton className="h-10 w-full rounded-xl" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-[16px] font-semibold text-text-primary">通知偏好</h2>
        <p className="text-[13px] text-text-muted mt-1">选择您希望接收的通知类型</p>
      </div>

      {NOTIF_TYPE_GROUPS.map((group) => (
        <div key={group.group}>
          <h3 className="text-[14px] font-semibold text-text-secondary mb-3">{group.label}</h3>
          <div className="space-y-2">
            {group.items.map((item) => {
              const enabled = prefMap.get(item.type) ?? true
              return (
                <Card key={item.type} padding="md" className="flex items-center justify-between">
                  <p className="text-[14px] text-text-primary">{item.label}</p>
                  <Switch.Root
                    checked={enabled}
                    onCheckedChange={(checked) => {
                      updateMut.mutate({ type: item.type, enabled: checked })
                    }}
                    className={cn(
                      'w-11 h-6 rounded-full relative cursor-default transition-colors shrink-0',
                      enabled ? 'bg-amber' : 'bg-white/[0.06] border border-white/[0.04]',
                    )}
                  >
                    <Switch.Thumb
                      className={cn(
                        'block w-4 h-4 bg-white rounded-full shadow-sm transition-transform duration-100 will-change-transform',
                        enabled ? 'translate-x-[22px]' : 'translate-x-[3px]',
                      )}
                    />
                  </Switch.Root>
                </Card>
              )
            })}
          </div>
        </div>
      ))}
    </div>
  )
}

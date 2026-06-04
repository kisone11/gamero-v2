import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as Switch from '@radix-ui/react-switch'
import { userApi } from '@/api/user'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/stores/toastStore'
import { cn } from '@/lib/utils'
import { Card } from '@/components/ui/card'
import { Select } from '@/components/settings/Select'
import type { CooperationType } from '@/types/enums'

const COOP_OPTIONS: { value: CooperationType; label: string }[] = [
  { value: 'online', label: '线上合作' },
  { value: 'offline', label: '线下合作' },
  { value: 'hybrid', label: '混合模式' },
]

export function AvailabilityTab() {
  const queryClient = useQueryClient()
  const user = useAuthStore((s) => s.user)

  const [isAvailable, setIsAvailable] = useState(false)
  const [coopPreference, setCoopPreference] = useState<CooperationType>('online')
  const [location, setLocation] = useState('')
  const [loaded, setLoaded] = useState(false)

  const profileQuery = useQuery({
    queryKey: ['user-profile', user?.username],
    queryFn: () => userApi.getProfile(user!.username),
    enabled: !!user?.username,
  })

  useEffect(() => {
    if (profileQuery.data && !loaded) {
      setIsAvailable(profileQuery.data.is_available ?? false)
      setCoopPreference(profileQuery.data.coop_preference ?? 'online')
      setLocation(profileQuery.data.location ?? '')
      setLoaded(true)
    }
  }, [profileQuery.data, loaded])

  const saveMut = useMutation({
    mutationFn: () =>
      userApi.setAvailability({
        is_available: isAvailable,
        coop_preference: isAvailable ? coopPreference : undefined,
        location: isAvailable ? location || undefined : undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-profile', user?.username] })
      toast.success('合作状态已更新')
    },
    onError: (err: any) => toast.error(err?.message || '保存失败'),
  })

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
        <h2 className="text-[16px] font-semibold text-text-primary">合作状态</h2>
        <p className="text-body text-text-muted mt-1">设置是否开放合作以及合作方式偏好</p>
      </div>

      <Card padding="md" className="flex items-center justify-between">
        <div>
          <p className="text-h4 text-text-primary">开放合作</p>
          <p className="text-small text-text-muted mt-0.5">
            开启后，其他用户可以在你的个人主页看到合作意向
          </p>
        </div>
        <Switch.Root
          checked={isAvailable}
          onCheckedChange={setIsAvailable}
          className={cn(
            'w-11 h-6 rounded-full relative cursor-default transition-colors shrink-0',
            isAvailable ? 'bg-amber' : 'bg-white/[0.06] border border-white/[0.04]',
          )}
        >
          <Switch.Thumb
            className={cn(
              'block w-4 h-4 bg-white rounded-full shadow-sm transition-transform duration-100 will-change-transform',
              isAvailable ? 'translate-x-[22px]' : 'translate-x-[3px]',
            )}
          />
        </Switch.Root>
      </Card>

      {isAvailable && (
        <>
          <div className="flex flex-col gap-1.5">
            <label className="text-meta font-semibold text-text-muted uppercase tracking-wider">
              合作方式偏好
            </label>
            <Select
              value={coopPreference}
              onChange={(v) => setCoopPreference(v as CooperationType)}
              options={COOP_OPTIONS}
            />
          </div>

          <Input
            label="所在地"
            value={location}
            onChange={(e) => setLocation(e.target.value)}
            fullWidth
            placeholder="例如：北京"
          />
        </>
      )}

      {!isAvailable && (
        <Card>
          <p className="text-body text-text-muted">
            开启合作状态后，可以设置合作方式偏好和所在地信息。
          </p>
        </Card>
      )}

      <Button
        loading={saveMut.isPending}
        onClick={() => saveMut.mutate()}
      >
        保存
      </Button>
    </div>
  )
}

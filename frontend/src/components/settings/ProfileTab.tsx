import { useState, useEffect, useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Upload } from 'lucide-react'
import { userApi } from '@/api/user'
import { uploadFile } from '@/lib/upload'
import { useAuthStore } from '@/stores/authStore'
import { Avatar } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/stores/toastStore'

export function ProfileTab() {
  const queryClient = useQueryClient()
  const user = useAuthStore((s) => s.user)
  const updateUser = useAuthStore((s) => s.updateUser)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [nickname, setNickname] = useState(user?.nickname ?? '')
  const [bio, setBio] = useState('')
  const [location, setLocation] = useState('')
  const [origBio, setOrigBio] = useState('')
  const [origLocation, setOrigLocation] = useState('')
  const [avatarUploading, setAvatarUploading] = useState(false)
  const [loaded, setLoaded] = useState(false)

  const profileQuery = useQuery({
    queryKey: ['user-profile', user?.username],
    queryFn: () => userApi.getProfile(user!.username),
    enabled: !!user?.username,
  })

  useEffect(() => {
    if (profileQuery.data && !loaded) {
      setNickname(profileQuery.data.nickname ?? user?.nickname ?? '')
      setBio(profileQuery.data.bio ?? '')
      setLocation(profileQuery.data.location ?? '')
      setOrigBio(profileQuery.data.bio ?? '')
      setOrigLocation(profileQuery.data.location ?? '')
      setLoaded(true)
    }
  }, [profileQuery.data, user?.nickname, loaded])

  const handleAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setAvatarUploading(true)
    try {
      const key = await uploadFile('avatar', file)
      await userApi.updateAvatar(key)
      queryClient.invalidateQueries({ queryKey: ['user-profile', user?.username] })
      updateUser({ avatar_url: key })
      toast.success('头像已更新')
    } catch (err: any) {
      toast.error(err?.message || '头像上传失败')
    } finally {
      setAvatarUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const updateProfileMut = useMutation({
    mutationFn: () => {
      const payload: Record<string, string> = { nickname }
      if (bio || (origBio && !bio)) payload.bio = bio
      if (location || (origLocation && !location)) payload.location = location
      return userApi.updateProfile(payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-profile', user?.username] })
      if (nickname) updateUser({ nickname })
      toast.success('个人资料已更新')
    },
    onError: (err: any) => toast.error(err?.message || '保存失败'),
  })

  if (profileQuery.isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-24 w-full rounded-xl" />
        <Skeleton className="h-10 w-full rounded-xl" />
        <Skeleton className="h-10 w-full rounded-xl" />
        <Skeleton className="h-24 w-full rounded-xl" />
      </div>
    )
  }

  const currentAvatar = profileQuery.data?.avatar_url ?? user?.avatar_url

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-[16px] font-semibold text-text-primary">个人资料</h2>
        <p className="text-body text-text-muted mt-1">管理你的个人主页展示信息</p>
      </div>

      <div className="flex items-center gap-4">
        <div className="relative">
          <Avatar src={currentAvatar} name={user?.nickname} size="xl" />
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={avatarUploading}
            className="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 hover:opacity-100 transition-opacity rounded-lg cursor-pointer"
          >
            <Upload className="w-5 h-5 text-white" />
          </button>
        </div>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={handleAvatarChange}
        />
        <div>
          <p className="text-body font-semibold text-text-primary">{user?.nickname}</p>
          <p className="text-caption text-text-muted font-mono">@{user?.username}</p>
          <Button
            variant="outline"
            size="sm"
            loading={avatarUploading}
            onClick={() => fileInputRef.current?.click()}
            className="mt-2"
          >
            更换头像
          </Button>
        </div>
      </div>

      <Input
        label="昵称"
        value={nickname}
        onChange={(e) => setNickname(e.target.value)}
        fullWidth
      />

      <Textarea
        label="个人简介"
        value={bio}
        onChange={(e) => setBio(e.target.value)}
        rows={4}
        fullWidth
        placeholder="介绍一下你自己……"
        maxLength={500}
      />

      <Input
        label="所在地"
        value={location}
        onChange={(e) => setLocation(e.target.value)}
        fullWidth
        placeholder="例如：北京"
      />

      <div className="pt-2">
        <Button
          loading={updateProfileMut.isPending}
          disabled={updateProfileMut.isPending}
          onClick={() => updateProfileMut.mutate()}
        >
          保存
        </Button>
      </div>
    </div>
  )
}

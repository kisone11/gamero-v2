import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Trash2 } from 'lucide-react'
import { authApi } from '@/api/auth'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'

export function SecurityTab() {
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()

  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  const changePwdMut = useMutation({
    mutationFn: () => authApi.changePassword({ old_password: oldPassword, new_password: newPassword }),
    onSuccess: () => {
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      toast.success('密码已修改')
    },
    onError: (err: any) => toast.error(err?.message || '修改密码失败'),
  })

  const handleChangePassword = () => {
    if (!oldPassword || !newPassword) {
      toast.error('请填写完整')
      return
    }
    if (newPassword.length < 6) {
      toast.error('新密码至少 6 位')
      return
    }
    if (newPassword !== confirmPassword) {
      toast.error('两次密码不一致')
      return
    }
    changePwdMut.mutate()
  }

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState('')

  const deleteAccountMut = useMutation({
    mutationFn: () => authApi.deleteAccount({ confirmation: deleteConfirm }),
    onSuccess: () => {
      toast.success('账号已删除')
      logout()
      navigate('/')
    },
    onError: (err: any) => toast.error(err?.message || '删除账号失败'),
  })

  const currentUsername = useAuthStore.getState().user?.username

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-[16px] font-semibold text-text-primary">安全设置</h2>
        <p className="text-body text-text-muted mt-1">管理你的账号安全</p>
      </div>

      <section>
        <h3 className="text-h4 text-text-primary mb-4">修改密码</h3>
        <div className="space-y-4">
          <Input
            label="当前密码"
            type="password"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            fullWidth
          />
          <Input
            label="新密码"
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            fullWidth
            hint="至少 6 位字符"
          />
          <Input
            label="确认新密码"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            fullWidth
          />
          <Button
            loading={changePwdMut.isPending}
            disabled={!oldPassword || !newPassword || !confirmPassword}
            onClick={handleChangePassword}
          >
            修改密码
          </Button>
        </div>
      </section>

      <section className="border-t border-white/[0.04] pt-8">
        <h3 className="text-h4 text-danger mb-2">危险区域</h3>
        <p className="text-body text-text-muted mb-4">
          删除账号后，所有数据将被永久清除且不可恢复。
        </p>
        <Button variant="danger" onClick={() => setDeleteDialogOpen(true)}>
          <Trash2 className="h-4 w-4" />
          删除账号
        </Button>
      </section>

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除账号</DialogTitle>
            <DialogDescription>
              此操作不可撤销。所有项目、日志、帖子等数据将被永久删除。
              请输入你的用户名 <strong>{currentUsername}</strong> 以确认。
            </DialogDescription>
          </DialogHeader>
          <Input
            value={deleteConfirm}
            onChange={(e) => setDeleteConfirm(e.target.value)}
            placeholder={currentUsername}
            fullWidth
          />
          <DialogFooter>
            <Button variant="ghost" onClick={() => { setDeleteDialogOpen(false); setDeleteConfirm('') }}>
              取消
            </Button>
            <Button
              variant="danger"
              loading={deleteAccountMut.isPending}
              disabled={deleteConfirm !== currentUsername}
              onClick={() => deleteAccountMut.mutate()}
            >
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

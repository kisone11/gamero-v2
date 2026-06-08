import { Navigate } from 'react-router-dom'
import { Shield } from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { getAccessTokenRole, isAdminRole } from '@/lib/auth-role'

export function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user, isAuthenticated, logout } = useAuthStore()

  if (!isAuthenticated || !user) {
    return <Navigate to="/login" replace />
  }

  const tokenRole = getAccessTokenRole()
  const canAccessAdmin = isAdminRole(tokenRole)

  if (!canAccessAdmin) {
    return (
      <div className="min-h-[70vh] flex items-center justify-center px-6">
        <div className="max-w-md text-center bg-surface-card border border-white/[0.04] rounded-2xl p-8">
          <div className="w-14 h-14 mx-auto mb-4 rounded-2xl bg-danger/10 flex items-center justify-center">
            <Shield className="w-7 h-7 text-danger" />
          </div>
          <h1 className="text-[20px] font-bold text-text-primary mb-2">无权访问管理后台</h1>
          <p className="text-[14px] text-text-secondary mb-6 leading-relaxed">
            当前账号不是管理员。请使用 admin 或 superadmin 账号登录。
          </p>
          <Button onClick={() => { logout(); window.location.assign('/login') }}>重新登录管理员账号</Button>
        </div>
      </div>
    )
  }

  return children
}

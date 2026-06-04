import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation } from '@tanstack/react-query'
import { authApi } from '@/api/auth'
import { useAuthStore } from '@/stores/authStore'
import { Input, Button } from '@/components/ui'
import { toast } from '@/stores/toastStore'

const loginSchema = z.object({
  email: z.string().email('请输入有效的邮箱地址'),
  password: z.string().min(8, '密码至少 8 位'),
})
type LoginFormData = z.infer<typeof loginSchema>

const inputClass =
  'h-11 rounded-lg border-white/[0.06] text-[14px] px-3.5 focus:border-amber/40 focus:shadow-[0_0_0_3px_rgba(245,166,35,0.08)]'
const btnClass = '!h-11 !rounded-lg w-full'

export default function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const login = useAuthStore((s) => s.login)
  const isExpired = searchParams.get('expired') === '1'

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  })

  const loginMut = useMutation({
    mutationFn: (data: LoginFormData) => authApi.login(data),
    onSuccess: (res) => {
      login(res.user, { access: res.access_token, refresh: res.refresh_token })
      toast.success('登录成功')
      const adminRoles = ['admin', 'superadmin']
      navigate(adminRoles.includes(res.user.role ?? '') ? '/admin' : '/feed', { replace: true })
    },
    onError: (err: any) => toast.error(err?.response?.data?.message || '登录失败'),
  })

  return (
    <>
      {isExpired && (
        <div className="mb-5 px-4 py-3 rounded-lg bg-warning/10 border border-warning/20 text-[13px] text-warning text-center">
          登录已过期，请重新登录
        </div>
      )}
      <h1 className="text-[22px] font-bold text-center mb-1.5">欢迎回来</h1>
      <p className="text-[14px] text-text-muted text-center mb-7">
        登录你的 Gamero 账号
      </p>

      <form
        onSubmit={handleSubmit((d) => loginMut.mutate(d))}
        className="space-y-4"
        noValidate
      >
        <Input
          label="邮箱"
          type="email"
          placeholder="your@email.com"
          error={errors.email?.message}
          fullWidth
          className={inputClass}
          {...register('email')}
        />
        <Input
          label="密码"
          type="password"
          placeholder="输入密码"
          error={errors.password?.message}
          fullWidth
          className={inputClass}
          {...register('password')}
        />

        <div className="flex justify-end">
          <Link
            to="/forgot-password"
            className="text-[12px] font-medium text-text-muted hover:text-amber transition-colors no-underline hover:underline"
          >
            忘记密码？
          </Link>
        </div>

        <Button
          type="submit"
          loading={loginMut.isPending}
          className={btnClass}
        >
          登录
        </Button>
      </form>

      {/* Divider: 或 */}
      <div className="relative my-6">
        <div className="absolute inset-0 flex items-center">
          <div className="w-full border-t border-white/[0.06]" />
        </div>
        <div className="relative flex justify-center">
          <span className="bg-surface-card px-3 text-[12px] text-text-muted">
            或
          </span>
        </div>
      </div>

      <p className="text-center text-[14px] text-text-muted">
        还没有账号？{' '}
        <Link
          to="/register"
          className="text-amber font-medium hover:underline no-underline"
        >
          注册
        </Link>
      </p>
    </>
  )
}

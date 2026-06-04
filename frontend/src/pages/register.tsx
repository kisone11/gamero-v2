import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation } from '@tanstack/react-query'
import { authApi } from '@/api/auth'
import { useAuthStore } from '@/stores/authStore'
import { Input, Button } from '@/components/ui'
import { toast } from '@/stores/toastStore'

// ---------- schema ----------

const registerSchema = z
  .object({
    email: z.string().email('请输入有效的邮箱地址'),
    code: z.string().length(6, '验证码为 6 位'),
    password: z
      .string()
      .min(8, '密码至少 8 位')
      .max(32, '密码最多 32 位')
      .regex(/[a-zA-Z]/, '密码必须包含字母')
      .regex(/[0-9]/, '密码必须包含数字'),
    confirmPassword: z.string().min(8, '请再次输入密码'),
    nickname: z
      .string()
      .optional()
      .refine(
        (val) => !val || val.length === 0 || val.trim().length >= 2,
        '昵称至少 2 个字符'
      )
      .refine(
        (val) => !val || val.length === 0 || val.trim().length <= 32,
        '昵称最多 32 个字符'
      )
      .or(z.literal('')),
  })
  .refine((d) => d.password === d.confirmPassword, {
    message: '两次密码不一致',
    path: ['confirmPassword'],
  })

type RegisterFormData = z.infer<typeof registerSchema>

// ---------- shared classes ----------

const inputClass =
  'h-11 rounded-lg border-white/[0.06] text-[14px] px-3.5 focus:border-amber/40 focus:shadow-[0_0_0_3px_rgba(245,166,35,0.08)]'
const btnClass = '!h-11 !rounded-lg'

// ---------- component ----------

export default function RegisterPage() {
  const navigate = useNavigate()
  const login = useAuthStore((s) => s.login)

  const [countdown, setCountdown] = useState(0)
  const [emailError, setEmailError] = useState<string>('') // 邮箱发送验证码的错误

  const form = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    mode: 'onChange',
  })

  const emailValue = form.watch('email')
  const isEmailValid = !form.formState.errors.email && emailValue?.includes('@')

  // Countdown tick
  useEffect(() => {
    if (countdown <= 0) return
    const timer = setInterval(() => setCountdown((prev) => prev - 1), 1000)
    return () => clearInterval(timer)
  }, [countdown])

  // ---------- mutations ----------

  const sendCodeMut = useMutation({
    mutationFn: () =>
      authApi.sendEmailCode({ email: emailValue, scene: 'register' }),
    onSuccess: () => {
      toast.success('验证码已发送，请查看邮箱')
      setCountdown(60)
      setEmailError('') // 清除错误
    },
    onError: (err: any) => {
      console.log('发送验证码错误 - 完整对象:', err)
      console.log('发送验证码错误 - err.message:', err?.message)
      console.log('发送验证码错误 - err.code:', err?.code)
      
      // extractData 抛出的是整个 res.data 对象: { code: 2006, message: "邮箱已被注册", data: null }
      // 所以直接访问 err.message 即可获取后端返回的错误消息
      const message = err?.message || '发送验证码失败'
      toast.error(message)
      setEmailError(message) // 显示在输入框下方
      setCountdown(0)
    },
  })

  const registerMut = useMutation({
    mutationFn: (data: RegisterFormData) =>
      authApi.register({
        email: data.email,
        code: data.code,
        password: data.password,
        nickname: data.nickname || undefined,
      }),
    onSuccess: (res) => {
      login(res.user, { access: res.access_token, refresh: res.refresh_token })
      toast.success('注册成功')
      navigate('/feed', { replace: true })
    },
    onError: (err: any) => {
      const message = 
        err?.response?.data?.message || 
        err?.message || 
        '注册失败，请检查输入信息'
      
      console.log('注册错误详情:', {
        response: err?.response?.data,
        message: err?.message,
        err
      })
      
      toast.error(message)
    },
  })

  // ---------- handlers ----------

  const handleSendCode = () => {
    if (!isEmailValid || countdown > 0) return
    setEmailError('') // 清除之前的错误
    sendCodeMut.mutate()
  }

  const handleSubmit = form.handleSubmit((data) => {
    registerMut.mutate(data)
  })

  return (
    <>
      <h1 className="text-[22px] font-bold text-center mb-1.5">创建账号</h1>
      <p className="text-[14px] text-text-muted text-center mb-7">
        加入 Gamero 游戏开发者社区
      </p>

      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        {/* 邮箱 + 发送验证码按钮 */}
        <div>
          <label className="block text-[12px] font-medium text-text-secondary mb-1.5">
            邮箱
          </label>
          <div className="flex gap-2">
            <div className="flex-1">
              <Input
                type="email"
                placeholder="your@email.com"
                error={form.formState.errors.email?.message || emailError}
                className={inputClass}
                {...form.register('email')}
                onChange={(e) => {
                  form.register('email').onChange(e)
                  setEmailError('') // 用户修改邮箱时清除错误
                }}
              />
            </div>
            <Button
              type="button"
              onClick={handleSendCode}
              disabled={!isEmailValid || countdown > 0 || sendCodeMut.isPending}
              loading={sendCodeMut.isPending}
              className={`${btnClass} min-w-[110px]`}
            >
              {countdown > 0 ? `${countdown}s` : '发送验证码'}
            </Button>
          </div>
        </div>

        {/* 验证码 */}
        <Input
          label="验证码"
          type="text"
          placeholder="6 位验证码"
          error={form.formState.errors.code?.message}
          fullWidth
          maxLength={6}
          className={inputClass}
          {...form.register('code')}
        />

        {/* 密码 */}
        <Input
          label="密码"
          type="password"
          placeholder="8-32 位，需包含字母和数字"
          error={form.formState.errors.password?.message}
          fullWidth
          className={inputClass}
          {...form.register('password')}
        />

        {/* 确认密码 */}
        <Input
          label="确认密码"
          type="password"
          placeholder="再次输入密码"
          error={form.formState.errors.confirmPassword?.message}
          fullWidth
          className={inputClass}
          {...form.register('confirmPassword')}
        />

        {/* 昵称（选填） */}
        <Input
          label="昵称（选填）"
          type="text"
          placeholder="你的展示名称"
          error={form.formState.errors.nickname?.message}
          fullWidth
          className={inputClass}
          {...form.register('nickname')}
        />

        {/* 提交按钮 */}
        <Button
          type="submit"
          loading={registerMut.isPending}
          className={`${btnClass} w-full`}
        >
          创建账号
        </Button>
      </form>

      <p className="mt-6 text-center text-[14px] text-text-muted">
        已有账号？{' '}
        <Link
          to="/login"
          className="text-amber font-medium hover:underline no-underline"
        >
          登录
        </Link>
      </p>
    </>
  )
}

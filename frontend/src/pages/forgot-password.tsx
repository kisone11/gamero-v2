import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation } from '@tanstack/react-query'
import { authApi } from '@/api/auth'
import { Input, Button } from '@/components/ui'
import { toast } from '@/stores/toastStore'

// ---------- schemas ----------

const step1Schema = z.object({
  email: z.string().email('请输入有效的邮箱地址'),
})
type Step1Data = z.infer<typeof step1Schema>

const step2Schema = z
  .object({
    code: z.string().length(6, '验证码为 6 位'),
    newPassword: z
      .string()
      .min(8, '密码至少 8 位')
      .max(32, '密码最多 32 位')
      .regex(/[a-zA-Z]/, '密码必须包含字母')
      .regex(/[0-9]/, '密码必须包含数字'),
    confirmPassword: z.string().min(8, '请再次输入密码'),
  })
  .refine((d) => d.newPassword === d.confirmPassword, {
    message: '两次密码不一致',
    path: ['confirmPassword'],
  })
type Step2Data = z.infer<typeof step2Schema>

// ---------- shared classes ----------

const inputClass =
  'h-11 rounded-lg border-white/[0.06] text-[14px] px-3.5 focus:border-amber/40 focus:shadow-[0_0_0_3px_rgba(245,166,35,0.08)]'
const btnClass = '!h-11 !rounded-lg w-full'

// ---------- component ----------

export default function ForgotPasswordPage() {
  const navigate = useNavigate()

  const [step, setStep] = useState(1)
  const [email, setEmail] = useState('')
  const [countdown, setCountdown] = useState(0)

  const step1Form = useForm<Step1Data>({
    resolver: zodResolver(step1Schema),
    mode: 'onChange', // 实时验证
  })

  const step2Form = useForm<Step2Data>({
    resolver: zodResolver(step2Schema),
    mode: 'onChange', // 实时验证
  })

  // Countdown tick
  useEffect(() => {
    if (countdown <= 0) return
    const timer = setInterval(() => setCountdown((prev) => prev - 1), 1000)
    return () => clearInterval(timer)
  }, [countdown])

  // ---------- mutations ----------

  const sendCodeMut = useMutation({
    mutationFn: (data: Step1Data) =>
      authApi.sendEmailCode({ email: data.email, scene: 'reset' }),
    onSuccess: () => {
      toast.success('验证码已发送，请查看邮箱')
      setEmail(step1Form.getValues('email'))
      setStep(2)

      // 设置 60 秒倒计时
      setCountdown(60)
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.message || '发送验证码失败')
      setCountdown(0)
    },
  })

  const resetMut = useMutation({
    mutationFn: (data: Step2Data) =>
      authApi.resetPasswordByCode({
        email,
        code: data.code,
        new_password: data.newPassword,
      }),
    onSuccess: () => {
      toast.success('密码重置成功，请登录')
      navigate('/login', { replace: true })
    },
    onError: (err: any) =>
      toast.error(err?.response?.data?.message || '重置密码失败'),
  })

  // ---------- handlers ----------

  const handleSendCode = step1Form.handleSubmit((d) => {
    // 不在这里设置倒计时，而是在 onSuccess 中根据情况设置
    sendCodeMut.mutate(d)
  })

  const handleBack = () => {
    setStep(1)
    // 不清除倒计时，保持发送间隔限制
    // setCountdown(0)
  }

  // ========== Step 1 ==========

  if (step === 1) {
    return (
      <>
        <h1 className="text-[22px] font-bold text-center mb-1.5">重置密码</h1>
        <p className="text-[14px] text-text-muted text-center mb-7">
          输入邮箱地址获取验证码
        </p>

        <form onSubmit={handleSendCode} className="space-y-4" noValidate>
          <Input
            label="邮箱"
            type="email"
            placeholder="your@email.com"
            error={step1Form.formState.errors.email?.message}
            fullWidth
            className={inputClass}
            {...step1Form.register('email')}
          />
          <Button
            type="submit"
            loading={sendCodeMut.isPending}
            disabled={countdown > 0}
            className={btnClass}
          >
            {countdown > 0 ? `${countdown}s` : '发送验证码'}
          </Button>
        </form>

        <p className="mt-6 text-center text-[14px] text-text-muted">
          想起密码了？{' '}
          <Link
            to="/login"
            className="text-amber font-medium hover:underline no-underline"
          >
            返回登录
          </Link>
        </p>
      </>
    )
  }

  // ========== Step 2 ==========

  return (
    <>
      {/* Back + email indicator */}
      <div className="flex items-center gap-2 mb-5">
        <button
          type="button"
          onClick={handleBack}
          className="text-[12px] font-medium text-amber hover:underline cursor-pointer"
        >
          &larr; 返回
        </button>
        <span className="text-[12px] text-text-muted">{email}</span>
      </div>

      <h1 className="text-[22px] font-bold text-center mb-1.5">设置新密码</h1>
      <p className="text-[14px] text-text-muted text-center mb-7">
        输入验证码和新密码
      </p>

      <form
        onSubmit={step2Form.handleSubmit((d) => resetMut.mutate(d))}
        className="space-y-4"
        noValidate
      >
        <Input
          label="验证码"
          type="text"
          placeholder="6 位验证码"
          error={step2Form.formState.errors.code?.message}
          fullWidth
          maxLength={6}
          className={inputClass}
          {...step2Form.register('code')}
        />
        <Input
          label="新密码"
          type="password"
          placeholder="8-32 位，需包含字母和数字"
          error={step2Form.formState.errors.newPassword?.message}
          fullWidth
          className={inputClass}
          {...step2Form.register('newPassword')}
        />
        <Input
          label="确认密码"
          type="password"
          placeholder="再次输入新密码"
          error={step2Form.formState.errors.confirmPassword?.message}
          fullWidth
          className={inputClass}
          {...step2Form.register('confirmPassword')}
        />
        <Button
          type="submit"
          loading={resetMut.isPending}
          className={btnClass}
        >
          重置密码
        </Button>
      </form>

      <p className="mt-6 text-center text-[14px] text-text-muted">
        想起密码了？{' '}
        <Link
          to="/login"
          className="text-amber font-medium hover:underline no-underline"
        >
          返回登录
        </Link>
      </p>
    </>
  )
}

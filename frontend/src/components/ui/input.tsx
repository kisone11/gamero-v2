import { cn } from '@/lib/utils'
import { forwardRef, useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  hint?: string
  error?: string
  fullWidth?: boolean
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, hint, error, fullWidth, className, id, type, ...props }, ref) => {
    const inputId = id || (label ? label.replace(/\s+/g, '-').toLowerCase() : undefined)
    const [showPassword, setShowPassword] = useState(false)
    const isPasswordInput = type === 'password'
    const inputType = isPasswordInput && showPassword ? 'text' : type

    return (
      <div className={cn(fullWidth ? 'w-full' : '')}>
        {label && (
          <label htmlFor={inputId} className="block text-[12px] font-medium text-text-secondary mb-1.5">
            {label}
          </label>
        )}
        <div className="relative">
          <input
            ref={ref}
            id={inputId}
            type={inputType}
            className={cn(
              'w-full h-10 px-3.5 bg-surface-void rounded-lg',
              'text-[14px] text-text-primary placeholder:text-text-muted',
              'border transition-all duration-150',
              error
                ? 'border-danger/40 focus:border-danger focus:shadow-[0_0_0_3px_rgba(217,86,94,0.10)]'
                : 'border-white/[0.06] hover:border-white/[0.10] focus:border-amber/40 focus:shadow-[0_0_0_3px_rgba(245,166,35,0.08)]',
              'outline-none',
              props.disabled && 'opacity-50 cursor-not-allowed',
              isPasswordInput && 'pr-10', // 为眼睛图标留出空间
              className
            )}
            {...props}
          />
          {isPasswordInput && (
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
              tabIndex={-1}
            >
              {showPassword ? (
                <EyeOff className="w-4 h-4" />
              ) : (
                <Eye className="w-4 h-4" />
              )}
            </button>
          )}
        </div>
        {hint && !error && (
          <p className="mt-1.5 text-[11px] text-text-muted">{hint}</p>
        )}
        {error && (
          <p className="mt-1.5 text-[11px] text-danger">{error}</p>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'
export { Input }

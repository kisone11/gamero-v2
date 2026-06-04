import { cn } from '@/lib/utils'
import { Loader2 } from 'lucide-react'
import { forwardRef } from 'react'

type Variant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'default' | 'info' | 'success'
type Size = 'sm' | 'md' | 'lg'

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
  icon?: React.ReactNode
}

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ variant = 'primary', size = 'md', loading, icon, className, disabled, children, ...props }, ref) => {
    const base = cn(
      'inline-flex items-center justify-center gap-2 font-medium rounded-lg',
      'transition-all duration-150 ease-out',
      'select-none whitespace-nowrap',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber/40 focus-visible:ring-offset-2 focus-visible:ring-offset-surface-void',
      'disabled:opacity-40 disabled:cursor-not-allowed',
      'active:scale-[0.97]',
    )

    const variants: Record<Variant, string> = {
      primary: cn(
        'bg-amber text-surface-void font-semibold shadow-[0_1px_2px_rgba(0,0,0,0.1),0_0_0_1px_rgba(245,166,35,0.2)]',
        'hover:bg-amber-light hover:shadow-[0_2px_8px_rgba(245,166,35,0.25)]',
        'active:bg-amber-dim',
      ),
      default: cn(
        'bg-white/[0.04] text-text-secondary border border-white/[0.08]',
        'hover:text-text-primary hover:border-white/[0.12] hover:bg-white/[0.06]',
      ),
      secondary: cn(
        'bg-transparent text-text-secondary border border-white/[0.08]',
        'hover:text-amber hover:border-amber/30 hover:bg-amber/[0.04]',
      ),
      outline: cn(
        'bg-transparent text-text-secondary border border-white/[0.04]',
        'hover:text-text-primary hover:border-white/[0.10] hover:bg-white/[0.02]',
      ),
      ghost: cn(
        'bg-transparent text-text-muted',
        'hover:text-text-primary hover:bg-white/[0.04]',
      ),
      danger: cn(
        'bg-transparent text-danger border border-danger/20',
        'hover:text-danger hover:border-danger/40 hover:bg-danger/[0.04] hover:shadow-[0_0_12px_rgba(217,86,94,0.15)]',
      ),
      info: cn(
        'bg-cyan/10 text-cyan border border-cyan/20',
        'hover:bg-cyan/15 hover:border-cyan/35',
      ),
      success: cn(
        'bg-success/10 text-success border border-success/20',
        'hover:bg-success/15 hover:border-success/35',
      ),
    }

    const sizes: Record<Size, string> = {
      sm: 'h-8 px-3 text-[12px] rounded-md',
      md: 'h-10 px-4 text-[13px]',
      lg: 'h-11 px-6 text-[14px]',
    }

    return (
      <button
        ref={ref}
        disabled={disabled || loading}
        className={cn(base, variants[variant], sizes[size], className)}
        {...props}
      >
        {loading ? (
          <Loader2 className="h-4 w-4 animate-spin shrink-0" />
        ) : icon ? (
          <span className="shrink-0">{icon}</span>
        ) : null}
        {children}
      </button>
    )
  }
)

Button.displayName = 'Button'
export { Button }

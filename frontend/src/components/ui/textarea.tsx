import { cn } from '@/lib/utils'
import { forwardRef } from 'react'

interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string
  error?: string
  fullWidth?: boolean
}

const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ label, error, fullWidth, className, id, ...props }, ref) => {
    const inputId = id || (label ? label.replace(/\s+/g, '-').toLowerCase() : undefined)
    return (
      <div className="w-full">
        {label && (
          <label htmlFor={inputId} className="block text-[12px] font-medium text-text-secondary mb-1.5">
            {label}
          </label>
        )}
        <textarea
          ref={ref}
          id={inputId}
          className={cn(
            'w-full min-h-[100px] px-3.5 py-2.5 bg-surface-void rounded-lg resize-y',
            'text-[14px] text-text-primary placeholder:text-text-muted',
            'border transition-all duration-150',
            error
              ? 'border-danger/40 focus:border-danger focus:shadow-[0_0_0_3px_rgba(217,86,94,0.10)]'
              : 'border-white/[0.06] hover:border-white/[0.10] focus:border-amber/40 focus:shadow-[0_0_0_3px_rgba(245,166,35,0.08)]',
            'outline-none',
            props.disabled && 'opacity-50 cursor-not-allowed',
            className
          )}
          {...props}
        />
        {error && (
          <p className="mt-1.5 text-[11px] text-danger">{error}</p>
        )}
      </div>
    )
  }
)

Textarea.displayName = 'Textarea'
export { Textarea }

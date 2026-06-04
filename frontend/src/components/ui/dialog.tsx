import * as DialogPrimitive from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

export function Dialog({ children, ...props }: DialogPrimitive.DialogProps) {
  return <DialogPrimitive.Root {...props}>{children}</DialogPrimitive.Root>
}

export function DialogTrigger({ children, asChild = true, ...props }: DialogPrimitive.DialogTriggerProps) {
  return <DialogPrimitive.Trigger asChild={asChild} {...props}>{children}</DialogPrimitive.Trigger>
}

export function DialogContent({ children, className, ...props }: DialogPrimitive.DialogContentProps) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm data-[state=open]:animate-fade-in" />
      <DialogPrimitive.Content
        className={cn(
          'fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2',
          'bg-surface-raised border border-white/[0.06] rounded-xl p-6',
          'shadow-[0_16px_48px_rgba(0,0,0,0.4)]',
          'data-[state=open]:animate-slide-up',
          className
        )}
        {...props}
      >
        {children}
        <DialogPrimitive.Close className="absolute right-4 top-4 p-1 rounded-md text-text-muted hover:text-text-primary hover:bg-white/[0.04] transition-colors">
          <X className="h-4 w-4" />
        </DialogPrimitive.Close>
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  )
}

export function DialogHeader({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn('mb-5', className)}>{children}</div>
}

export function DialogTitle({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <DialogPrimitive.Title className={cn('text-[16px] font-bold text-text-primary', className)}>
      {children}
    </DialogPrimitive.Title>
  )
}

export function DialogDescription({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <DialogPrimitive.Description className={cn('mt-1.5 text-[13px] text-text-muted', className)}>
      {children}
    </DialogPrimitive.Description>
  )
}

export function DialogFooter({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn('mt-6 flex justify-end gap-3', className)}>{children}</div>
}

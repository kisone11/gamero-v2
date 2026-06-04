import { useToastStore } from '@/stores/toastStore'
import { X, CheckCircle, XCircle, AlertTriangle, Info } from 'lucide-react'

const icons = {
  success: CheckCircle,
  error: XCircle,
  warning: AlertTriangle,
  info: Info,
}

const colors = {
  success: 'border-success/30 bg-success/5',
  error: 'border-danger/30 bg-danger/5',
  warning: 'border-warning/30 bg-warning/5',
  info: 'border-cyan/30 bg-cyan/5',
}

const iconColors = {
  success: 'text-success',
  error: 'text-danger',
  warning: 'text-warning',
  info: 'text-cyan',
}

export function ToastContainer() {
  const { toasts, remove } = useToastStore()
  if (toasts.length === 0) return null

  return (
    <div className="fixed bottom-6 right-6 z-[9999] flex flex-col gap-2 max-w-sm">
      {toasts.map(toast => {
        const Icon = icons[toast.type]
        return (
          <div
            key={toast.id}
            className={`flex items-start gap-3 px-4 py-3 rounded-xl border backdrop-blur-xl animate-slide-up ${colors[toast.type]}`}
          >
            <Icon className={`w-4 h-4 mt-0.5 shrink-0 ${iconColors[toast.type]}`} />
            <p className="text-[13px] text-text-primary flex-1">{toast.message}</p>
            <button onClick={() => remove(toast.id)} className="text-text-muted hover:text-text-primary shrink-0">
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        )
      })}
    </div>
  )
}

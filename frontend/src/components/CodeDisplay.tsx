import { useEffect, useState, useRef } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'

interface CodeDisplayProps {
  code: string
  onClose: () => void
  duration?: number // 显示时长（毫秒），默认 10000ms
}

export function CodeDisplay({ code, onClose, duration = 10000 }: CodeDisplayProps) {
  const [timeLeft, setTimeLeft] = useState(duration / 1000)
  const containerRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    // 创建专用容器
    const container = document.createElement('div')
    container.id = 'code-display-portal'
    document.body.appendChild(container)
    containerRef.current = container

    return () => {
      // 清理容器
      if (containerRef.current && document.body.contains(containerRef.current)) {
        document.body.removeChild(containerRef.current)
      }
    }
  }, [])

  useEffect(() => {
    // 倒计时
    const timer = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          onClose()
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [onClose])

  // 等待容器创建完成
  if (!containerRef.current) return null

  // 使用 Portal 渲染到专用容器
  return createPortal(
    <div className="fixed bottom-6 left-6 z-50 animate-in slide-in-from-bottom-4 fade-in duration-300">
      <div className="bg-surface-elevated border border-white/[0.08] rounded-xl shadow-xl p-4 min-w-[280px]">
        <div className="flex items-start justify-between mb-2">
          <h3 className="text-[14px] font-semibold text-text-primary">
            开发模式：验证码
          </h3>
          <button
            onClick={onClose}
            className="text-text-muted hover:text-text-secondary transition-colors -mt-1 -mr-1"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="bg-amber/10 border border-amber/20 rounded-lg px-4 py-3 mb-2">
          <p className="text-[11px] text-text-muted mb-1">验证码</p>
          <p className="text-[28px] font-bold text-amber tracking-[0.3em] font-mono">
            {code}
          </p>
        </div>
        <div className="flex items-center justify-between text-[11px]">
          <span className="text-text-muted">
            {timeLeft} 秒后自动关闭
          </span>
          <button
            onClick={() => {
              navigator.clipboard.writeText(code)
              alert('验证码已复制到剪贴板')
            }}
            className="text-amber hover:text-amber/80 font-medium transition-colors"
          >
            复制
          </button>
        </div>
      </div>
    </div>,
    containerRef.current
  )
}

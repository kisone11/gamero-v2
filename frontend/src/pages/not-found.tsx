import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'

export default function NotFoundPage() {
  const navigate = useNavigate()
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const homePath = isAuthenticated ? '/feed' : '/projects'

  return (
    <div className="min-h-[70vh] flex items-center justify-center px-6">
      <div className="text-center max-w-md">
        {/* Large 404 */}
        <div className="text-[120px] leading-none font-bold text-amber/[0.12] select-none tracking-tighter">
          404
        </div>

        {/* Fun message */}
        <h1 className="text-[22px] font-bold text-text-primary -mt-4 mb-3">
          这个页面迷路了
        </h1>
        <p className="text-[14px] text-text-secondary mb-8 leading-relaxed">
          看起来你来到了一个不存在的地方。
          <br />
          也许是传送坐标错了，也许这个页面还在加载中……
          <br />
          要不，先回首页看看？
        </p>

        {/* Actions */}
        <div className="flex items-center justify-center gap-3">
          <Button onClick={() => navigate(homePath)}>
            返回首页
          </Button>
          <Button
            variant="outline"
            onClick={() => window.history.length > 1 ? navigate(-1) : navigate(homePath)}
          >
            返回上一页
          </Button>
        </div>
      </div>
    </div>
  )
}

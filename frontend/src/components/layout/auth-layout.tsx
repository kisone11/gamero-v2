import { Outlet, Link } from 'react-router-dom'

export function AuthLayout() {
  return (
    <div className="min-h-screen bg-surface-void flex flex-col items-center justify-center px-4">
      <Link to="/" className="font-brand text-h3 text-amber tracking-tight mb-8">
        GAMERO
      </Link>
      <div className="bg-surface-card border border-white/[0.04] rounded-xl p-6 w-full max-w-[400px] mx-4">
        <Outlet />
      </div>
      <p className="mt-6 text-small text-text-muted">
        游戏开发者社区
      </p>
    </div>
  )
}

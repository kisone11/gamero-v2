import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Settings, Shield, Wrench, Briefcase, UserCheck, Bell } from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import {
  ProfileTab,
  SecurityTab,
  SkillsTab,
  PortfolioTab,
  AvailabilityTab,
  NotificationTab,
} from '@/components/settings'

// ============================================================
// Constants
// ============================================================

const TABS = [
  { key: 'profile', label: '个人资料', icon: Settings },
  { key: 'security', label: '安全', icon: Shield },
  { key: 'skills', label: '技能', icon: Wrench },
  { key: 'portfolio', label: '作品集', icon: Briefcase },
  { key: 'availability', label: '合作状态', icon: UserCheck },
  { key: 'notifications', label: '通知偏好', icon: Bell },
] as const

// ============================================================
// SettingsPage
// ============================================================

export default function SettingsPage() {
  const user = useAuthStore((s) => s.user)
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('profile')

  if (!user) {
    return (
      <div className="max-w-[720px] mx-auto px-6 py-8 min-h-[60vh] flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Shield className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">请先登录</p>
          <p className="text-[13px] text-text-muted mb-6">需要登录才能访问设置页面</p>
          <Button variant="secondary" size="sm" onClick={() => navigate('/login')}>
            去登录
          </Button>
        </div>
      </div>
    )
  }

  const ActiveComponent = ({ tabKey }: { tabKey: string }) => {
    switch (tabKey) {
      case 'profile': return <ProfileTab />
      case 'security': return <SecurityTab />
      case 'skills': return <SkillsTab />
      case 'portfolio': return <PortfolioTab />
      case 'availability': return <AvailabilityTab />
      case 'notifications': return <NotificationTab />
      default: return null
    }
  }

  return (
    <div className="max-w-[720px] mx-auto px-6 py-8">
      <h1 className="text-[22px] font-bold text-text-primary mb-6">设置</h1>

      <div className="flex gap-8">
        {/* Sidebar tabs — vertical on desktop */}
        <aside className="hidden md:block w-44 shrink-0">
          <div className="sticky top-[88px] space-y-0.5">
            {TABS.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={cn(
                  'w-full text-left px-3 py-2 rounded-lg text-[13px] font-medium transition-colors flex items-center gap-2.5',
                  activeTab === tab.key
                    ? 'bg-white/[0.06] text-text-primary'
                    : 'text-text-muted hover:text-text-secondary hover:bg-white/[0.03]'
                )}
              >
                <tab.icon className="w-4 h-4" />
                {tab.label}
              </button>
            ))}
          </div>
        </aside>

        {/* Horizontal tabs on mobile */}
        <div className="md:hidden w-full overflow-x-auto mb-6">
          <div className="flex gap-1 pb-2">
            {TABS.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={cn(
                  'whitespace-nowrap px-4 py-2 rounded-lg text-[13px] font-medium transition-all duration-200',
                  activeTab === tab.key
                    ? 'bg-white/[0.06] text-text-primary'
                    : 'text-text-muted hover:text-text-secondary hover:bg-white/[0.02]'
                )}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        {/* Content */}
        <main className="flex-1 min-w-0">
          <div className="bg-surface-card border border-white/[0.04] rounded-xl p-6">
            <ActiveComponent tabKey={activeTab} />
          </div>
        </main>
      </div>
    </div>
  )
}

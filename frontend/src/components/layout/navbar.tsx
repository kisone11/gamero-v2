import { Link, useNavigate, useLocation } from 'react-router-dom'
import { Search, Bell, LogOut, User, Settings, Plus, Gamepad2, Flag, CheckCheck, UserPlus, CheckCircle, XCircle, AtSign, Heart, MessageSquare, ThumbsUp, Send, Ban, ClipboardCheck, RefreshCw, X, Shield } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { cn } from '@/lib/utils'
import { Avatar } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/stores/authStore'
import { useNotifStore } from '@/stores/notifStore'
import { useClickOutside } from '@/hooks/useClickOutside'
import { notificationApi } from '@/api/notification'
import { timeAgo } from '@/lib/time'
import { useState, useRef, useEffect } from 'react'
import type { Notification } from '@/types/api'

const NAV_LINKS = [
  { href: '/feed', label: '动态' },
  { href: '/projects', label: '发现' },
  { href: '/community', label: '社区' },
  { href: '/recruit', label: '招募' },
  { href: '/talent', label: '人才' },
]

// ============================================================
// Notification helpers
// ============================================================

const NOTIF_ICON_MAP: Record<string, React.ComponentType<{ className?: string }>> = {
  recruitment_applied: UserPlus,
  application_approved: CheckCircle,
  application_rejected: XCircle,
  mention: AtSign,
  new_follower: UserPlus,
  project_status_changed: RefreshCw,
  post_liked: Heart,
  post_commented: MessageSquare,
  log_liked: ThumbsUp,
  log_commented: MessageSquare,
  system: Bell,
  talent_invited: Send,
  invite_accepted: CheckCheck,
  invite_declined: X,
  project_banned: Ban,
  project_unbanned: CheckCircle,
  report_handled: ClipboardCheck,
}

function getNotificationUrl(type: string, metadata: any): string | null {
  const m = metadata || {}
  switch (type) {
    case 'post_liked': case 'post_commented': case 'mention':
      return m.post_id ? `/post/${m.post_id}` : null
    case 'log_liked': case 'log_commented':
      return m.log_id ? `/devlog/${m.log_id}` : null
    case 'new_follower':
      return m.follower_username ? `/u/${m.follower_username}` : m.follower_id ? `/u/${m.follower_id}` : null
    case 'project_status_changed': case 'project_banned': case 'project_unbanned':
      return m.project_slug ? `/p/${m.project_slug}` : m.project_id ? `/p/${m.project_id}` : null
    case 'recruitment_applied': case 'application_approved': case 'application_rejected':
      return m.recruitment_id ? `/recruit/${m.recruitment_id}` : null
    case 'talent_invited': case 'invite_accepted': case 'invite_declined':
      return m.invitation_id ? `/me/invitations` : null
    case 'report_handled':
      return '/me/reports'
    default:
      return null
  }
}

export function Navbar() {
  const { user, logout, isAuthenticated } = useAuthStore()
  const location = useLocation()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)
  const [bellOpen, setBellOpen] = useState(false)
  const [searchFocused, setSearchFocused] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const bellRef = useClickOutside<HTMLDivElement>({
    handler: () => setBellOpen(false),
    enabled: bellOpen,
  })

  const wsUnread = useNotifStore(s => s.unreadCount)

  const { data: unreadCount } = useQuery({
    queryKey: ['unread-count'],
    queryFn: () => notificationApi.getUnreadCount(),
    enabled: isAuthenticated && !!user,
    retry: false,
    refetchInterval: 30000,
  })

  const count = wsUnread > 0 ? wsUnread : ((unreadCount as any)?.unread_count ?? 0)
  const canAccessAdmin = user?.role === 'admin' || user?.role === 'superadmin'

  const { data: dropdownData } = useQuery({
    queryKey: ['notifications-dropdown'],
    queryFn: () => notificationApi.list(1, 5),
    enabled: isAuthenticated && !!user && bellOpen,
    retry: false,
  })

  const markAllReadDropdown = async () => {
    try {
      await notificationApi.markAllRead()
      useNotifStore.getState().setUnread(0)
      setBellOpen(false)
    } catch {}
  }

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const isActive = (href: string) => location.pathname.startsWith(href)

  return (
    <header className="sticky top-0 z-50 h-16 bg-surface-deep/85 backdrop-blur-2xl border-b border-white/[0.04]">
      <div className="h-full max-w-[1280px] mx-auto px-6 flex items-center">
        {/* Logo */}
        <Link to="/feed" className="flex items-center gap-2.5 mr-8 shrink-0 group">
          <div className="w-8 h-8 rounded-md bg-gradient-to-br from-amber to-amber-dim flex items-center justify-center shadow-[0_0_12px_rgba(245,166,35,0.2)]">
            <Gamepad2 className="w-4 h-4 text-surface-void" />
          </div>
          <span className="font-brand text-lg font-bold text-text-primary tracking-tight">
            GAMERO
          </span>
        </Link>

        {/* Nav Links */}
        <nav className="flex items-center h-full">
          {NAV_LINKS.map(link => (
            <Link
              key={link.href}
              to={link.href}
              className={cn(
                'relative h-full flex items-center px-4 text-[13px] font-medium transition-colors',
                isActive(link.href)
                  ? 'text-text-primary'
                  : 'text-text-muted hover:text-text-secondary'
              )}
            >
              {link.label}
              {isActive(link.href) && (
                <span className="absolute bottom-0 left-3 right-3 h-[2px] bg-amber rounded-full" />
              )}
            </Link>
          ))}
        </nav>

        <div className="flex-1" />

        {/* Search */}
        <div className={cn(
          'hidden md:flex items-center gap-2 h-9 rounded-lg border transition-all duration-200 mr-4',
          searchFocused
            ? 'w-72 border-amber/40 bg-surface-void/80 shadow-[0_0_0_3px_rgba(245,166,35,0.08)]'
            : 'w-56 border-transparent hover:border-white/[0.06] bg-white/[0.03]'
        )}>
          <Search className="w-4 h-4 text-text-muted ml-3 shrink-0" />
          <input
            placeholder="搜索项目、日志、开发者..."
            className="bg-transparent border-none outline-none text-[13px] text-text-primary placeholder:text-text-muted w-full pr-3"
            onFocus={() => setSearchFocused(true)}
            onBlur={() => setSearchFocused(false)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                const q = (e.target as HTMLInputElement).value.trim()
                if (q) navigate(`/search?q=${encodeURIComponent(q)}`)
              }
            }}
          />
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1">
          <Link to="/projects/new">
            <Button variant="primary" size="sm">
              <Plus className="w-3.5 h-3.5" />
              创建
            </Button>
          </Link>

          {isAuthenticated && user ? (
            <>
              <div className="relative" ref={bellRef}>
                <button
                  onClick={() => setBellOpen(!bellOpen)}
                  className="relative p-2 text-text-muted hover:text-text-primary transition-colors rounded-lg hover:bg-white/[0.04]"
                >
                  <Bell className="w-[18px] h-[18px]" />
                  {count > 0 && (
                    <span className="absolute -top-0.5 -right-0.5 min-w-[18px] h-[18px] flex items-center justify-center px-1 text-[10px] font-bold text-white bg-coral rounded-full ring-2 ring-surface-deep">
                      {count > 99 ? '99+' : count}
                    </span>
                  )}
                </button>

                {/* Bell dropdown */}
                {bellOpen && (
                  <>
                    <div className="fixed inset-0 z-40" onClick={() => setBellOpen(false)} />
                    <div className="absolute right-0 top-full mt-2 w-[360px] bg-surface-raised border border-white/[0.06] rounded-xl shadow-[0_16px_48px_rgba(0,0,0,0.4)] z-50 animate-fade-in overflow-hidden">
                      <div className="px-4 py-3 border-b border-white/[0.04] flex items-center justify-between">
                        <span className="text-[14px] font-semibold text-text-primary">通知</span>
                        <span className="text-[11px] text-text-muted font-mono">{count} 条未读</span>
                      </div>

                      <div className="max-h-[360px] overflow-y-auto">
                        {dropdownData?.list?.length ? (
                          dropdownData.list.map((notif: Notification) => {
                            const IconComp = NOTIF_ICON_MAP[notif.type] || Bell
                            return (
                              <div
                                key={notif.id}
                                className={cn(
                                  'flex items-start gap-3 px-4 py-3 cursor-pointer transition-colors',
                                  'hover:bg-white/[0.03]',
                                  !notif.is_read && 'bg-amber/[0.02]',
                                )}
                                onClick={() => {
                                  setBellOpen(false)
                                  const url = getNotificationUrl(notif.type, notif.metadata)
                                  if (url) navigate(url)
                                }}
                              >
                                <div className="w-8 h-8 rounded-lg bg-amber/10 flex items-center justify-center shrink-0">
                                  <IconComp className="w-4 h-4 text-amber" />
                                </div>
                                <div className="min-w-0 flex-1">
                                  <div className="flex items-center gap-2">
                                    <span
                                      className={cn(
                                        'text-[13px] truncate',
                                        !notif.is_read ? 'font-semibold text-text-primary' : 'text-text-secondary',
                                      )}
                                    >
                                      {notif.title}
                                    </span>
                                    {!notif.is_read && (
                                      <span className="w-1.5 h-1.5 rounded-full bg-amber shrink-0" />
                                    )}
                                  </div>
                                  <p className="text-[12px] text-text-muted truncate mt-0.5">
                                    {notif.content}
                                  </p>
                                  <span className="text-[10px] text-text-muted/60 font-mono mt-1 inline-block">
                                    {timeAgo(notif.created_at)}
                                  </span>
                                </div>
                              </div>
                            )
                          })
                        ) : (
                          <div className="px-4 py-10 text-center">
                            <Bell className="w-6 h-6 text-text-muted/40 mx-auto mb-2" />
                            <p className="text-[13px] text-text-muted">暂无通知</p>
                          </div>
                        )}
                      </div>

                      <div className="border-t border-white/[0.04] p-3 flex gap-2">
                        <Link
                          to="/notifications"
                          onClick={() => setBellOpen(false)}
                          className="flex-1 h-9 flex items-center justify-center text-[12px] font-medium text-text-secondary bg-transparent border border-white/[0.08] rounded-lg hover:text-amber hover:border-amber/30 hover:bg-amber/[0.04] transition-colors"
                        >
                          查看全部
                        </Link>
                        <button
                          onClick={markAllReadDropdown}
                          className="flex-1 h-9 flex items-center justify-center gap-1.5 text-[12px] font-medium text-text-secondary bg-transparent border border-white/[0.08] rounded-lg hover:text-amber hover:border-amber/30 hover:bg-amber/[0.04] transition-colors"
                        >
                          <CheckCheck className="w-3.5 h-3.5" />
                          全部已读
                        </button>
                      </div>
                    </div>
                  </>
                )}
              </div>

              <div className="relative ml-1" ref={menuRef}>
                <button
                  onClick={() => setMenuOpen(!menuOpen)}
                  className="flex items-center gap-2 p-0.5 rounded-lg transition-colors hover:bg-white/[0.04]"
                >
                  <Avatar src={user.avatar_url} name={user.nickname} size="sm" />
                </button>

                {menuOpen && (
                  <>
                    <div className="fixed inset-0 z-40" onClick={() => setMenuOpen(false)} />
                    <div className="absolute right-0 top-full mt-2 w-56 bg-surface-raised border border-white/[0.06] rounded-xl shadow-[0_16px_48px_rgba(0,0,0,0.4)] py-1.5 z-50 animate-fade-in overflow-hidden">
                      <div className="px-3 py-2.5 border-b border-white/[0.04]">
                        <p className="text-[13px] font-medium text-text-primary">{user.nickname}</p>
                        <p className="text-[11px] text-text-muted mt-0.5 font-mono">@{user.username}</p>
                      </div>
                      <Link to={`/u/${user.username || user.id}`} className="flex items-center gap-3 px-3 py-2.5 text-[13px] text-text-secondary hover:text-text-primary hover:bg-white/[0.03] transition-colors" onClick={() => setMenuOpen(false)}>
                        <User className="w-4 h-4 text-text-muted" />
                        个人主页
                      </Link>
                      <Link to="/settings" className="flex items-center gap-3 px-3 py-2.5 text-[13px] text-text-secondary hover:text-text-primary hover:bg-white/[0.03] transition-colors" onClick={() => setMenuOpen(false)}>
                        <Settings className="w-4 h-4 text-text-muted" />
                        设置
                      </Link>
                      <Link to="/me/reports" className="flex items-center gap-3 px-3 py-2.5 text-[13px] text-text-secondary hover:text-text-primary hover:bg-white/[0.03] transition-colors" onClick={() => setMenuOpen(false)}>
                        <Flag className="w-4 h-4 text-text-muted" />
                        我的举报
                      </Link>
                      {canAccessAdmin && (
                        <Link to="/admin" className="flex items-center gap-3 px-3 py-2.5 text-[13px] text-text-secondary hover:text-text-primary hover:bg-white/[0.03] transition-colors" onClick={() => setMenuOpen(false)}>
                          <Shield className="w-4 h-4 text-text-muted" />
                          管理后台
                        </Link>
                      )}
                      <div className="border-t border-white/[0.04] mt-1 pt-1">
                        <button onClick={() => { logout(); setMenuOpen(false); navigate('/login') }} className="flex items-center gap-3 px-3 py-2.5 text-[13px] text-text-secondary hover:text-danger hover:bg-white/[0.03] transition-colors w-full text-left">
                          <LogOut className="w-4 h-4 text-text-muted" />
                          退出登录
                        </button>
                      </div>
                    </div>
                  </>
                )}
              </div>
            </>
          ) : (
            <div className="flex items-center gap-2 ml-2">
              <Link to="/login">
                <Button variant="ghost" size="sm">登录</Button>
              </Link>
              <Link to="/register">
                <Button variant="primary" size="sm">注册</Button>
              </Link>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}

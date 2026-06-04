import { cn } from '@/lib/utils'

type Size = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

interface AvatarProps {
  src?: string | null
  name?: string
  userId?: number
  size?: Size
  className?: string
}

const sizeMap: Record<Size, { container: string; text: string }> = {
  xs:  { container: 'w-6 h-6', text: 'text-[9px]' },
  sm:  { container: 'w-8 h-8', text: 'text-[11px]' },
  md:  { container: 'w-10 h-10', text: 'text-[13px]' },
  lg:  { container: 'w-12 h-12', text: 'text-[15px]' },
  xl:  { container: 'w-16 h-16', text: 'text-[20px]' },
}

export function Avatar({ src, name, userId, size = 'md', className }: AvatarProps) {
  const initials = name
    ? name.slice(0, 2).toUpperCase()
    : userId
      ? String(userId).slice(0, 2)
      : '?'

  const colors = [
    'from-amber to-amber-dim',
    'from-coral to-coral-dim',
    'from-cyan to-cyan-dim',
    'from-magenta to-magenta-dim',
    'from-success to-success',
  ]

  const colorIndex = userId ? userId % colors.length : (name?.length ?? 0) % colors.length
  const gradient = colors[colorIndex]

  return (
    <div
      className={cn(
        'relative inline-flex shrink-0 overflow-hidden rounded-lg',
        sizeMap[size].container,
        !src && `bg-gradient-to-br ${gradient}`,
        className
      )}
    >
      {src ? (
        <img
          src={src}
          alt={name || ''}
          className="w-full h-full object-cover"
          onError={(e) => {
            (e.target as HTMLImageElement).style.display = 'none'
            ;(e.target as HTMLImageElement).nextElementSibling?.classList.remove('hidden')
          }}
        />
      ) : null}
      <span
        className={cn(
          'absolute inset-0 flex items-center justify-center font-bold text-white/90',
          sizeMap[size].text,
          src && 'hidden'
        )}
      >
        {initials}
      </span>
    </div>
  )
}

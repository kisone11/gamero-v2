import { formatDistanceToNow, format, parseISO } from 'date-fns'
import { zhCN } from 'date-fns/locale'

export function timeAgo(iso: string): string {
  try {
    return formatDistanceToNow(parseISO(iso), { addSuffix: true, locale: zhCN })
  } catch {
    return iso
  }
}

export function formatDate(iso: string, fmt = 'yyyy年M月d日'): string {
  try {
    return format(parseISO(iso), fmt, { locale: zhCN })
  } catch {
    return iso
  }
}

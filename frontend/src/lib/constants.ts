import type { ProjectGenre, ProjectStatus, MemberRole } from '@/types/api'
import type { BadgeVariant } from '@/components/ui/badge'

/** Genre display labels */
export const GENRE_LABELS: Record<ProjectGenre, string> = {
  action: '动作',
  rpg: 'RPG',
  strategy: '策略',
  simulator: '模拟',
  puzzle: '益智',
  horror: '恐怖',
  platform: '平台',
  other: '其他',
}

/** Project status display config */
export const PROJECT_STATUS_CONFIG: Record<ProjectStatus, { label: string; variant: BadgeVariant }> = {
  preparing: { label: '筹备中', variant: 'warning' },
  developing: { label: '开发中', variant: 'primary' },
  playable: { label: '可试玩', variant: 'success' },
  launched: { label: '已上线', variant: 'success' },
  paused: { label: '暂停', variant: 'warning' },
  abandoned: { label: '已弃坑', variant: 'danger' },
}

/** Member role display labels */
export const MEMBER_ROLE_LABELS: Record<MemberRole, string> = {
  owner: '创建者',
  lead_programmer: '主程',
  lead_artist: '主美',
  lead_designer: '主策划',
  sound: '音效',
  tester: '测试',
  member: '成员',
}

/** Genre options for select/combobox */
export const GENRE_OPTIONS: { value: ProjectGenre; label: string }[] = [
  { value: 'action', label: '动作' },
  { value: 'rpg', label: 'RPG' },
  { value: 'strategy', label: '策略' },
  { value: 'simulator', label: '模拟' },
  { value: 'puzzle', label: '益智' },
  { value: 'horror', label: '恐怖' },
  { value: 'platform', label: '平台' },
  { value: 'other', label: '其他' },
]

/** Status options for select/combobox */
export const STATUS_OPTIONS: { value: ProjectStatus; label: string }[] = [
  { value: 'preparing', label: '筹备中' },
  { value: 'developing', label: '开发中' },
  { value: 'playable', label: '可试玩' },
  { value: 'launched', label: '已上线' },
  { value: 'paused', label: '暂停' },
  { value: 'abandoned', label: '已弃坑' },
]

/** Portfolio type display labels */
export const PORTFOLIO_TYPE_LABELS: Record<string, string> = {
  game: '游戏作品',
  demo: '试玩',
  art: '美术作品',
  code: '代码作品',
}

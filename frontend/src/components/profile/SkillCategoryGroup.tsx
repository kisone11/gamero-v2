import { cn } from '@/lib/utils'
import type { SkillCategory } from '@/types/api'

function skillLevelPercent(level: string): number {
  switch (level) {
    case 'beginner': return 33
    case 'intermediate': return 66
    case 'advanced': return 100
    default: return 0
  }
}

function skillLevelLabel(level: string): string {
  switch (level) {
    case 'beginner': return '初级'
    case 'intermediate': return '中级'
    case 'advanced': return '高级'
    default: return level
  }
}

function categoryLabel(cat: SkillCategory | string): string {
  const map: Record<string, string> = {
    program: '编程',
    art: '美术',
    design: '设计',
    sound: '音频',
    custom: '自定义',
  }
  return map[cat] ?? cat
}

function categoryColor(cat: SkillCategory | string): string {
  const map: Record<string, string> = {
    program: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
    art: 'bg-pink-500/10 text-pink-400 border-pink-500/20',
    design: 'bg-purple-500/10 text-purple-400 border-purple-500/20',
    sound: 'bg-green-500/10 text-green-400 border-green-500/20',
    custom: 'bg-amber/10 text-amber border-amber/20',
  }
  return map[cat] ?? 'bg-surface-deep text-text-secondary border-border-subtle'
}

export function SkillCategoryGroup({
  category,
  skills,
}: {
  category: string
  skills: { id: number; name: string; level: string; category: string; description?: string }[]
}) {
  if (skills.length === 0) return null
  return (
    <div className="mb-5 last:mb-0">
      <span className={cn('inline-block px-2 py-0.5 text-meta font-mono uppercase tracking-wider border rounded-lg mb-3', categoryColor(category))}>
        {categoryLabel(category)}
      </span>
      <div className="space-y-3">
        {skills.map((skill) => (
          <div key={skill.id} className="bg-white/[0.02] border border-white/[0.03] rounded-lg p-3">
            <div className="flex justify-between mb-1.5">
              <span className="text-body font-semibold text-text-primary">{skill.name}</span>
              <span className="text-caption font-medium text-amber">{skillLevelLabel(skill.level)}</span>
            </div>
            <div className="h-1.5 bg-white/[0.04] rounded-full overflow-hidden mb-2">
              <div className="h-full bg-amber rounded-full transition-all duration-500 ease-out" style={{ width: `${skillLevelPercent(skill.level)}%` }} />
            </div>
            {skill.description && (
              <p className="text-small text-text-muted leading-relaxed mt-1">{skill.description}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

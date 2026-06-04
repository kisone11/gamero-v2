import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Wrench, Plus, X } from 'lucide-react'
import { userApi } from '@/api/user'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/stores/toastStore'
import type { UserSkill, SkillInput } from '@/types/api'
import type { SkillCategory, SkillLevel } from '@/types/enums'

// Preset skills from backend model (mirrors backend/internal/model/user.go PresetSkills)
const PRESET_SKILLS: Record<SkillCategory, string[]> = {
  program: ['Unity', 'UE', 'Godot', '原生开发', 'Web前端', '后端开发', 'C#', 'C++', 'Lua', 'Python'],
  art: ['像素风', '3D建模', 'UI设计', '概念设计', '角色设计', '场景设计', '2D动画', '特效', '技术美术'],
  design: ['系统策划', '数值策划', '文案写作', '关卡设计', '剧情设计', '战斗设计'],
  sound: ['音乐创作', '音效制作', '声音设计', '配音', '混音'],
  custom: [],
}

const SKILL_CATEGORIES: { value: SkillCategory; label: string }[] = [
  { value: 'program', label: '编程' },
  { value: 'art', label: '美术' },
  { value: 'design', label: '策划' },
  { value: 'sound', label: '音效' },
  { value: 'custom', label: '自定义' },
]

const SKILL_LEVELS: { value: SkillLevel; label: string }[] = [
  { value: 'beginner', label: '入门' },
  { value: 'intermediate', label: '中级' },
  { value: 'advanced', label: '高级' },
]

const CATEGORY_COLORS: Record<SkillCategory, string> = {
  program: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
  art: 'bg-pink-500/10 text-pink-400 border-pink-500/20',
  design: 'bg-purple-500/10 text-purple-400 border-purple-500/20',
  sound: 'bg-green-500/10 text-green-400 border-green-500/20',
  custom: 'bg-amber/10 text-amber border-amber/15',
}

const LEVEL_PERCENT: Record<SkillLevel, number> = { beginner: 33, intermediate: 66, advanced: 100 }

let skillTempIdCounter = 0

export function SkillsTab() {
  const queryClient = useQueryClient()
  const user = useAuthStore((s) => s.user)

  const [skills, setSkills] = useState<(UserSkill & { _desc?: string })[]>([])
  const [loaded, setLoaded] = useState(false)
  const [activeCategory, setActiveCategory] = useState<SkillCategory>('program')

  const profileQuery = useQuery({
    queryKey: ['user-profile', user?.username],
    queryFn: () => userApi.getProfile(user!.username),
    enabled: !!user?.username,
  })

  useEffect(() => {
    if (profileQuery.data && !loaded) {
      setSkills((profileQuery.data.skills ?? []).map(s => ({ ...s, _desc: s.description || '' })))
      setLoaded(true)
    }
  }, [profileQuery.data, loaded])

  const updateSkillsMut = useMutation({
    mutationFn: () =>
      userApi.updateSkills({
        skills: skills.map((s) => ({ category: s.category, name: s.name, level: s.level, description: s._desc || undefined })),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-profile', user?.username] })
      toast.success('技能已更新')
    },
    onError: (err: any) => toast.error(err?.message || '保存失败'),
  })

  const addPresetSkill = (cat: SkillCategory, name: string) => {
    const exists = skills.some((s) => s.category === cat && s.name === name)
    if (exists) {
      toast.error('该技能已存在')
      return
    }
    const newSkill: UserSkill & { _desc?: string } = {
      id: --skillTempIdCounter,
      user_id: user?.id ?? 0,
      category: cat,
      name,
      level: 'intermediate' as SkillLevel,
      is_custom: false,
      created_at: new Date().toISOString(),
      _desc: '',
    }
    setSkills((prev) => [...prev, newSkill])
  }

  const addCustomSkill = () => {
    const input = document.getElementById('custom-skill-input') as HTMLInputElement
    const name = input?.value?.trim()
    if (!name) { toast.error('请输入技能名称'); return }
    const cat = activeCategory as SkillCategory
    const exists = skills.some((s) => s.category === cat && s.name === name)
    if (exists) { toast.error('该技能已存在'); return }
    const newSkill: UserSkill & { _desc?: string } = {
      id: --skillTempIdCounter,
      user_id: user?.id ?? 0,
      category: cat,
      name,
      level: 'intermediate' as SkillLevel,
      is_custom: true,
      created_at: new Date().toISOString(),
      _desc: '',
    }
    setSkills((prev) => [...prev, newSkill])
    if (input) input.value = ''
  }

  const updateSkillLevel = (id: number, level: SkillLevel) => {
    setSkills((prev) => prev.map((s) => (s.id === id ? { ...s, level } : s)))
  }

  const updateSkillDesc = (id: number, desc: string) => {
    setSkills((prev) => prev.map((s) => (s.id === id ? { ...s, _desc: desc } : s)))
  }

  const handleRemoveSkill = (id: number) => {
    setSkills((prev) => prev.filter((s) => s.id !== id))
  }

  const categoryLabel = (cat: SkillCategory) => SKILL_CATEGORIES.find((c) => c.value === cat)?.label ?? cat
  const levelLabel = (lvl: SkillLevel) => SKILL_LEVELS.find((l) => l.value === lvl)?.label ?? lvl

  const presetNames = PRESET_SKILLS[activeCategory] || []
  const unpickedPresets = presetNames.filter((n) => !skills.some((s) => s.category === activeCategory && s.name === n))
  const activeSkills = skills.filter((s) => s.category === activeCategory)

  if (profileQuery.isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full rounded-xl" />
        <Skeleton className="h-32 w-full rounded-xl" />
        <Skeleton className="h-32 w-full rounded-xl" />
      </div>
    )
  }

  if (profileQuery.isError) {
    return (
      <div className="py-16 text-center">
        <p className="text-body text-text-secondary mb-3">加载失败</p>
        <Button variant="secondary" size="sm" onClick={() => profileQuery.refetch()}>重试</Button>
      </div>
    )
  }

  const totalSkills = skills.length

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-[16px] font-semibold text-text-primary">技能特长</h2>
        <p className="text-body text-text-muted mt-1">展示你的专业技能和能力水平</p>
      </div>

      {/* Category tabs */}
      <div className="flex gap-1.5 flex-wrap">
        {SKILL_CATEGORIES.map((c) => {
          const count = skills.filter((s) => s.category === c.value).length
          return (
            <button
              key={c.value}
              onClick={() => setActiveCategory(c.value as SkillCategory)}
              className={`px-3 py-2 text-body font-medium rounded-lg border transition-all ${
                activeCategory === c.value
                  ? 'bg-amber/10 text-amber border-amber/20'
                  : 'text-text-muted border-white/[0.04] hover:border-white/[0.08]'
              }`}
            >
              {c.label}{count > 0 && <span className="ml-1.5 text-caption opacity-60">{count}</span>}
            </button>
          )
        })}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left: Preset skills + custom input */}
        <div className="space-y-4">
          <h3 className="text-body font-semibold text-text-secondary flex items-center gap-2">
            <Wrench className="w-4 h-4 text-amber" />
            推荐技能
          </h3>

          {/* Preset chips */}
          {unpickedPresets.length > 0 && (
            <div className="flex flex-wrap gap-1.5">
              {unpickedPresets.map((name) => (
                <button
                  key={name}
                  type="button"
                  onClick={() => addPresetSkill(activeCategory as SkillCategory, name)}
                  className="px-3 py-1.5 text-body rounded-lg border border-white/[0.04] bg-surface-card
                             text-text-secondary hover:border-amber/30 hover:text-amber transition-all active:scale-95"
                >
                  <Plus className="w-3 h-3 inline mr-1" />
                  {name}
                </button>
              ))}
            </div>
          )}
          {unpickedPresets.length === 0 && (
            <p className="text-small text-text-muted">已添加全部推荐技能</p>
          )}

          {/* Custom skill input */}
          <div className="flex items-center gap-2 pt-2 border-t border-white/[0.04]">
            <input
              id="custom-skill-input"
              type="text"
              placeholder="输入自定义技能名称..."
              className="flex-1 h-10 px-3.5 bg-surface-card border border-white/[0.06] rounded-xl text-body text-text-primary placeholder:text-text-muted focus:outline-none focus:border-amber/40"
              onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addCustomSkill() } }}
            />
            <Button variant="secondary" size="sm" onClick={addCustomSkill}>
              <Plus className="w-4 h-4" />添加
            </Button>
          </div>
        </div>

        {/* Right: My skills in this category */}
        <div className="space-y-3">
          <h3 className="text-body font-semibold text-text-secondary flex items-center gap-2">
            已选技能{activeSkills.length > 0 && <span className="text-caption text-text-muted font-normal">({activeSkills.length})</span>}
          </h3>

          {activeSkills.length === 0 && (
            <div className="text-center py-8 bg-white/[0.02] border border-white/[0.04] rounded-xl">
              <p className="text-body text-text-muted">还没有{categoryLabel(activeCategory as SkillCategory)}技能，从左侧添加</p>
            </div>
          )}

          {activeSkills.map((skill) => (
            <Card key={skill.id} className="space-y-3">
              {/* Header: category badge + name + remove */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className={`px-2 py-0.5 text-meta font-semibold uppercase tracking-wider border rounded ${CATEGORY_COLORS[skill.category as SkillCategory] || CATEGORY_COLORS.custom}`}>
                    {categoryLabel(skill.category as SkillCategory)}
                  </span>
                  <span className="text-h4 text-text-primary">{skill.name}</span>
                  {skill.is_custom && (
                    <span className="text-meta text-text-muted border border-white/[0.04] rounded px-1.5 py-0.5">自定义</span>
                  )}
                </div>
                <button onClick={() => handleRemoveSkill(skill.id)} className="text-text-muted hover:text-danger transition-colors">
                  <X className="w-4 h-4" />
                </button>
              </div>

              {/* Level selector */}
              <div className="flex items-center gap-2">
                <span className="text-small text-text-muted shrink-0">熟练度</span>
                {(['beginner', 'intermediate', 'advanced'] as SkillLevel[]).map((lvl) => (
                  <button
                    key={lvl}
                    onClick={() => updateSkillLevel(skill.id, lvl)}
                    className={`px-3 py-1.5 text-small font-medium rounded-lg border transition-all ${
                      skill.level === lvl
                        ? 'bg-amber/10 text-amber border-amber/20'
                        : 'text-text-muted border-white/[0.04] hover:border-white/[0.08]'
                    }`}
                  >
                    {levelLabel(lvl)}
                  </button>
                ))}
              </div>

              {/* Level bar */}
              <div className="h-1.5 bg-white/[0.04] rounded-full overflow-hidden">
                <div
                  className="h-full bg-amber rounded-full transition-all duration-300"
                  style={{ width: `${LEVEL_PERCENT[skill.level as SkillLevel] || 0}%` }}
                />
              </div>

              {/* Description */}
              <input
                type="text"
                value={skill._desc || ''}
                onChange={(e) => updateSkillDesc(skill.id, e.target.value)}
                placeholder="技能描述（选填，如：3年Unity开发经验...）"
                maxLength={200}
                className="w-full bg-surface-deep border border-white/[0.04] rounded-lg px-3 py-2 text-small text-text-primary placeholder:text-text-muted focus:outline-none focus:border-amber/40"
              />
            </Card>
          ))}
        </div>
      </div>

      {/* Footer: count + save */}
      <div className="flex items-center justify-between pt-4 border-t border-white/[0.04]">
        <span className="text-body text-text-muted">
          共 {totalSkills} 个技能
          {totalSkills === 0 && <span className="ml-1 text-amber">— 至少添加一个技能让其他人了解你的能力</span>}
        </span>
        <Button loading={updateSkillsMut.isPending} onClick={() => updateSkillsMut.mutate()} disabled={totalSkills === 0}>
          保存技能
        </Button>
      </div>
    </div>
  )
}

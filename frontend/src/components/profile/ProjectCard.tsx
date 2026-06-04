import { Link } from 'react-router-dom'
import { Gamepad2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import type { ProjectListItem } from '@/types/api'

function projectStatusLabel(status: string): string {
  const map: Record<string, string> = {
    preparing: '筹备中',
    developing: '开发中',
    playable: '可试玩',
    launched: '已上线',
    paused: '暂停',
    abandoned: '已弃坑',
  }
  return map[status] ?? status
}

function projectStatusColor(status: string): string {
  const map: Record<string, string> = {
    preparing: 'warning',
    developing: 'primary',
    playable: 'success',
    launched: 'success',
    paused: 'warning',
    abandoned: 'danger',
  }
  return map[status] ?? 'default'
}

export function ProjectCard({ project }: { project: ProjectListItem }) {
  const projectHref = `/p/${project.slug || project.id}`

  return (
    <Link
      to={projectHref}
      className="block bg-surface-card border border-white/[0.04] rounded-xl overflow-hidden
                 hover:border-amber/12 hover:shadow-card-hover
                 hover:-translate-y-[1px] transition-all duration-200 group"
    >
      <div className="p-5">
        <div className="flex items-start gap-4">
          <div className="w-10 h-10 rounded-xl bg-amber/10 flex items-center justify-center shrink-0 ring-1 ring-amber/20">
            <Gamepad2 className="w-5 h-5 text-amber" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 mb-1.5">
              <h4 className="text-h3 text-text-primary truncate group-hover:text-amber transition-colors">{project.name}</h4>
              <Badge variant={projectStatusColor(project.status) as any} size="sm" className="shrink-0">
                {projectStatusLabel(project.status)}
              </Badge>
            </div>
            {project.description && (
              <p className="text-body text-text-secondary line-clamp-2">{project.description}</p>
            )}
          </div>
        </div>
      </div>
      {project.cover_url && (
        <div className="border-t border-white/[0.04]" onClick={(e) => e.preventDefault()}>
          <img src={project.cover_url} alt={project.name} className="w-full max-h-40 object-cover" />
        </div>
      )}
    </Link>
  )
}

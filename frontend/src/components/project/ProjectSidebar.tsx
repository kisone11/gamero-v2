import { Monitor, Globe, Calendar, Clock, Play, Download, ExternalLink } from 'lucide-react'
import { Card } from '@/components/ui'
import { formatDate } from '@/lib/time'
import type { ProjectDetail } from '@/types/api'

export function ProjectSidebar({ project }: { project: ProjectDetail }) {
  return (
    <div className="space-y-4">
      <Card padding="lg">
        <h3 className="text-small font-semibold text-text-muted uppercase tracking-wider mb-4">项目信息</h3>
        <dl className="space-y-3">
          {project.engine && (
            <div className="flex items-start gap-3">
              <Monitor className="h-4 w-4 text-text-muted mt-0.5 shrink-0" />
              <div className="min-w-0">
                <dt className="text-small text-text-muted">引擎</dt>
                <dd className="text-body text-text-primary font-mono">{project.engine}</dd>
              </div>
            </div>
          )}
          {project.platform && project.platform.length > 0 && (
            <div className="flex items-start gap-3">
              <Globe className="h-4 w-4 text-text-muted mt-0.5 shrink-0" />
              <div className="min-w-0">
                <dt className="text-small text-text-muted">平台</dt>
                <dd className="text-body text-text-primary font-mono">{project.platform.join(' / ')}</dd>
              </div>
            </div>
          )}
          <div className="flex items-start gap-3">
            <Calendar className="h-4 w-4 text-text-muted mt-0.5 shrink-0" />
            <div className="min-w-0">
              <dt className="text-small text-text-muted">创建时间</dt>
              <dd className="text-body text-text-primary font-mono">{formatDate(project.created_at)}</dd>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <Clock className="h-4 w-4 text-text-muted mt-0.5 shrink-0" />
            <div className="min-w-0">
              <dt className="text-small text-text-muted">更新时间</dt>
              <dd className="text-body text-text-primary font-mono">{formatDate(project.updated_at)}</dd>
            </div>
          </div>
        </dl>
      </Card>

      {(project.demo_url || project.store_url) && (
        <Card padding="lg">
          <h3 className="text-small font-semibold text-text-muted uppercase tracking-wider mb-4">链接</h3>
          <div className="space-y-2">
            {project.demo_url && (
              <a
                href={project.demo_url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 text-body text-amber hover:text-amber-light transition-colors"
              >
                <Play className="h-3.5 w-3.5" />
                在线试玩
                <ExternalLink className="h-3 w-3 ml-auto" />
              </a>
            )}
            {project.store_url && (
              <a
                href={project.store_url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 text-body text-amber hover:text-amber-light transition-colors"
              >
                <Download className="h-3.5 w-3.5" />
                下载页面
                <ExternalLink className="h-3 w-3 ml-auto" />
              </a>
            )}
          </div>
        </Card>
      )}
    </div>
  )
}

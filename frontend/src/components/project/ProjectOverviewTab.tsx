import { FileText, Play, Download, Film, ExternalLink } from 'lucide-react'
import { Button, EmptyState } from '@/components/ui'
import { ImageGallery } from '@/components/shared/ImageGallery'
import type { ProjectDetail } from '@/types/api'

export function ProjectOverviewTab({ project }: { project: ProjectDetail }) {
  const hasScreenshots = project.screenshot_urls && project.screenshot_urls.length > 0
  const hasVideos = project.video_urls && project.video_urls.length > 0
  const hasContent = !!project.description || hasScreenshots || hasVideos
  const hasLinks = project.demo_url || project.store_url || project.video_url

  if (!hasContent && !hasLinks) {
    return (
      <div className="py-10">
        <EmptyState icon={<FileText className="w-7 h-7 text-text-muted" />} title="暂无内容" description="项目尚未完善简介信息" />
      </div>
    )
  }

  return (
    <div className="space-y-6 py-6">
      {/* Screenshots */}
      {hasScreenshots && (
        <section>
          <h3 className="text-h3 text-text-primary mb-4">截图</h3>
          <ImageGallery images={project.screenshot_urls} thumbnail maxShow={6} className="justify-start" />
        </section>
      )}

      {/* Videos */}
      {hasVideos && (
        <section>
          <h3 className="text-h3 text-text-primary mb-4">宣传视频</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {project.video_urls!.map((url, i) => (
              <video key={i} src={url} controls className="w-full rounded-xl" />
            ))}
          </div>
        </section>
      )}

      {/* Description */}
      {project.description && (
        <section>
          <h3 className="text-h3 text-text-primary mb-4">项目介绍</h3>
          <div className="text-body text-text-secondary leading-relaxed whitespace-pre-wrap bg-surface-card border border-white/[0.04] rounded-xl p-5">
            {project.description}
          </div>
        </section>
      )}

      {/* External links */}
      {hasLinks && (
        <section>
          <h3 className="text-h3 text-text-primary mb-4">外部链接</h3>
          <div className="flex flex-wrap gap-3">
            {project.demo_url && (
              <a href={project.demo_url} target="_blank" rel="noopener noreferrer">
                <Button variant="secondary" size="sm">
                  <Play className="h-4 w-4" />
                  在线试玩
                  <ExternalLink className="h-3 w-3 ml-1" />
                </Button>
              </a>
            )}
            {project.store_url && (
              <a href={project.store_url} target="_blank" rel="noopener noreferrer">
                <Button variant="secondary" size="sm">
                  <Download className="h-4 w-4" />
                  下载页面
                  <ExternalLink className="h-3 w-3 ml-1" />
                </Button>
              </a>
            )}
            {project.video_url && (
              <a href={project.video_url} target="_blank" rel="noopener noreferrer">
                <Button variant="secondary" size="sm">
                  <Film className="h-4 w-4" />
                  宣传视频
                  <ExternalLink className="h-3 w-3 ml-1" />
                </Button>
              </a>
            )}
          </div>
        </section>
      )}
    </div>
  )
}

import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Eye, PenLine, AlertTriangle, Search, Upload, Video, X } from 'lucide-react'
import { communityApi } from '@/api/community'
import { Button, Input, Textarea, Skeleton } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Markdown } from '@/components/ui/markdown'
import { toast } from '@/stores/toastStore'
import { uploadFile } from '@/lib/upload'
import { ImageGallery } from '@/components/shared/ImageGallery'
import { cn } from '@/lib/utils'

export default function EditPostPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const postId = Number(id)

  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [topicId, setTopicId] = useState<number | null>(null)
  const [preview, setPreview] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [unsavedOpen, setUnsavedOpen] = useState(false)
  const [newImageFiles, setNewImageFiles] = useState<File[]>([])
  const [newVideoFiles, setNewVideoFiles] = useState<File[]>([])
  const [uploadedImages, setUploadedImages] = useState<string[]>([])
  const [uploadedVideos, setUploadedVideos] = useState<string[]>([])
  const [uploading, setUploading] = useState(false)
  const imageInputRef = useRef<HTMLInputElement>(null)
  const videoInputRef = useRef<HTMLInputElement>(null)

  // Fetch post
  const { data: post, isLoading, isError } = useQuery({
    queryKey: ['post', postId],
    queryFn: () => communityApi.get(postId),
    enabled: !!id,
  })

  // Fetch topics for selector
  const { data: topics } = useQuery({
    queryKey: ['community-topics'],
    queryFn: () => communityApi.listTopics(),
  })

  useEffect(() => {
    if (post) {
      setTitle(post.title)
      setContent(post.content)
      setTopicId(post.topic_ids?.[0] ?? post.topics?.[0]?.id ?? null)
    }
  }, [post])

  // Unsaved changes warning
  useEffect(() => {
    if (!dirty) return
    const handler = (e: BeforeUnloadEvent) => { e.preventDefault(); e.returnValue = '' }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [dirty])

  const markDirty = () => setDirty(true)

  const handleNewImageChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || [])
    if (files.length === 0) return
    setNewImageFiles((prev) => [...prev, ...files])
    if (imageInputRef.current) imageInputRef.current.value = ''
    // Auto-upload immediately
    setUploading(true)
    for (const file of files) {
      try {
        const key = await uploadFile('post-image', file, postId)
        await communityApi.saveImage(postId, key)
        qc.invalidateQueries({ queryKey: ['post', postId] })
        qc.invalidateQueries({ queryKey: ['community-post', postId] })
      } catch {
        toast.error(`${file.name} 上传失败`)
      }
    }
    setUploading(false)
  }

  const handleNewVideoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || [])
    setNewVideoFiles((prev) => [...prev, ...files])
    if (videoInputRef.current) videoInputRef.current.value = ''
  }

  const updateMut = useMutation({
    mutationFn: async () => {
      // 图片已在选择时自动上传，这里只需处理视频
      if (newVideoFiles.length > 0) {
        setUploading(true)
        for (const file of newVideoFiles) {
          const key = await uploadFile('post-video', file, postId)
          await communityApi.saveVideo(postId, key)
          setUploadedVideos(prev => [...prev, URL.createObjectURL(file)])
        }
        setUploading(false)
      }
      return communityApi.update(postId, { title, content, topic_ids: topicId ? [topicId] : [] })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['post', postId] })
      qc.invalidateQueries({ queryKey: ['community-post', postId] })
      setNewImageFiles([])
      setNewVideoFiles([])
      setDirty(false)
      toast.success('已保存')
      navigate(`/post/${id}`)
    },
    onError: () => toast.error('保存失败'),
  })

  const topicList = Array.isArray(topics) ? topics : (topics as any)?.list ?? []

  // Loading state
  if (isLoading) {
    return (
      <div className="max-w-2xl mx-auto px-6 py-8 space-y-4">
        <Skeleton className="h-5 w-28" />
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  // Error state
  if (isError) {
    return (
      <div className="max-w-2xl mx-auto px-6 py-16 text-center">
        <div className="w-12 h-12 mx-auto mb-4 rounded-xl bg-white/[0.03] flex items-center justify-center">
          <AlertTriangle className="w-6 h-6 text-text-muted" />
        </div>
        <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
        <p className="text-[13px] text-text-muted mb-6">无法加载帖子信息，请重试</p>
        <div className="flex gap-3 justify-center">
          <Button variant="secondary" size="sm" onClick={() => navigate('/community')}>返回社区</Button>
          <Button variant="primary" size="sm" onClick={() => window.location.reload()}>重新加载</Button>
        </div>
      </div>
    )
  }

  // Not found
  if (!post) {
    return (
      <div className="max-w-2xl mx-auto px-6 py-16 text-center">
        <Search className="w-8 h-8 text-text-muted mb-4 mx-auto" />
        <p className="text-[15px] text-text-muted mb-2">帖子不存在</p>
        <Link to="/community" className="text-amber text-[13px] hover:underline">返回社区</Link>
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto px-6 py-8">
      <Link to={`/post/${id}`}
        className="inline-flex items-center gap-1.5 text-[13px] text-text-muted hover:text-amber transition-colors mb-6">
        <ArrowLeft className="w-4 h-4" /> 返回帖子
      </Link>

      <h1 className="text-[22px] font-bold text-text-primary mb-6">编辑帖子 · {post.title}</h1>

      <div className="space-y-5">
        {/* Title */}
        <Input
          label="标题"
          value={title}
          onChange={(e) => { setTitle(e.target.value); markDirty() }}
          placeholder="输入帖子标题..."
        />

        {/* Topic selector */}
        {topicList.length > 0 && (
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">
              话题
            </label>
            <div className="flex flex-wrap gap-2">
              <button
                onClick={() => { setTopicId(null); markDirty() }}
                className={cn(
                  'px-3 py-1.5 text-[13px] rounded-lg border transition-all active:scale-95',
                  topicId === null
                    ? 'border-amber bg-amber/10 text-amber'
                    : 'border-white/[0.04] text-text-muted hover:border-white/[0.08]',
                )}
              >
                无话题
              </button>
              {topicList.map((t: any) => (
                <button
                  key={t.id}
                  onClick={() => { setTopicId(t.id); markDirty() }}
                  className={cn(
                    'px-3 py-1.5 text-[13px] rounded-lg border transition-all active:scale-95',
                    topicId === t.id
                      ? 'border-amber bg-amber/10 text-amber'
                      : 'border-white/[0.04] text-text-muted hover:border-white/[0.08]',
                  )}
                >
                  {t.icon_key && <span className="mr-1">{t.icon_key}</span>}
                  {t.name}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Content with edit/preview tabs */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">
              内容
            </label>
            <div className="flex gap-1">
              <button
                onClick={() => setPreview(false)}
                className={cn(
                  'flex items-center gap-1 px-3 py-1.5 text-[12px] font-medium rounded-lg transition-all',
                  !preview ? 'bg-white/[0.06] text-text-primary' : 'text-text-muted hover:text-text-secondary',
                )}
              >
                <PenLine className="w-3.5 h-3.5" />
                编辑
              </button>
              <button
                onClick={() => setPreview(true)}
                className={cn(
                  'flex items-center gap-1 px-3 py-1.5 text-[12px] font-medium rounded-lg transition-all',
                  preview ? 'bg-white/[0.06] text-text-primary' : 'text-text-muted hover:text-text-secondary',
                )}
              >
                <Eye className="w-3.5 h-3.5" />
                预览
              </button>
            </div>
          </div>

          {preview ? (
            <div className="min-h-[300px] bg-surface-card border border-white/[0.04] rounded-xl p-5">
              {content ? (
                <Markdown content={content} />
              ) : (
                <p className="text-text-muted text-[14px]">暂无内容</p>
              )}
            </div>
          ) : (
            <>
              <Textarea
                value={content}
                onChange={(e) => { setContent(e.target.value); markDirty() }}
                placeholder="使用 Markdown 格式编写帖子内容..."
                rows={16}
              />
              <p className="text-right text-[11px] text-text-muted mt-1">{content.length} 字</p>
            </>
          )}
        </div>

        {/* Existing + newly uploaded images */}
        {((post.image_urls?.length ?? 0) > 0) && (
          <div>
            <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider block mb-2">已上传图片</label>
            <div className="flex gap-2 flex-wrap">
              {post.image_urls?.map((url, i) => {
                const key = post.image_keys?.[i]
                return (
                  <div key={`existing-${i}`} className="relative w-20 h-20 rounded-xl overflow-hidden border border-white/[0.04] group">
                    <img src={url} alt="" className="w-full h-full object-cover" />
                    {key && (
                      <button
                        type="button"
                        onClick={async () => {
                          try {
                            await communityApi.deleteImage(postId, key)
                            qc.invalidateQueries({ queryKey: ['post', postId] })
                            toast.success('图片已删除')
                          } catch {
                            toast.error('删除失败')
                          }
                        }}
                        className="absolute top-0.5 right-0.5 p-0.5 bg-black/50 rounded-full text-white hover:bg-black/70 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Upload progress */}
        {uploading && (
          <div className="flex items-center gap-2 text-[13px] text-amber">
            <div className="w-4 h-4 border-2 border-amber/30 border-t-amber rounded-full animate-spin" />
            正在上传...
          </div>
        )}

        {/* Existing videos */}
        {(post.video_urls?.length ?? 0) > 0 && (
          <div>
            <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider block mb-2">已上传视频</label>
            <div className="flex flex-wrap gap-2">
              {post.video_urls!.map((url, i) => {
                const key = post.video_keys?.[i]
                return (
                  <div key={i} className="relative w-48 h-28 rounded-xl overflow-hidden border border-white/[0.04] group">
                    <video src={url} controls className="w-full h-full object-cover" />
                    {key && (
                      <button
                        type="button"
                        onClick={async () => {
                          try {
                            await communityApi.deleteVideo(postId, key)
                            qc.invalidateQueries({ queryKey: ['post', postId] })
                            toast.success('视频已删除')
                          } catch {
                            toast.error('删除失败')
                          }
                        }}
                        className="absolute top-1 right-1 p-0.5 bg-black/50 rounded-full text-white hover:bg-black/70 opacity-0 group-hover:opacity-100 transition-opacity z-10"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Add images */}
        <input ref={imageInputRef} type="file" accept="image/*" multiple onChange={handleNewImageChange} className="hidden" />
        <button
          type="button"
          onClick={() => imageInputRef.current?.click()}
          className="w-full h-12 flex items-center justify-center border-2 border-dashed border-white/[0.04] rounded-xl hover:border-amber transition-colors"
        >
          <Upload className="h-5 w-5 text-text-muted mr-2" />
          <span className="text-[13px] text-text-muted font-semibold">添加图片</span>
        </button>

        {/* Add videos */}
        <input ref={videoInputRef} type="file" accept="video/mp4,video/quicktime,video/webm" onChange={handleNewVideoChange} className="hidden" />
        <button
          type="button"
          onClick={() => videoInputRef.current?.click()}
          className="w-full h-12 flex items-center justify-center border-2 border-dashed border-white/[0.04] rounded-xl hover:border-amber transition-colors"
        >
          <Video className="h-5 w-5 text-text-muted mr-2" />
          <span className="text-[13px] text-text-muted font-semibold">添加视频</span>
        </button>

        {/* Actions */}
        <div className="flex gap-3 pt-4">
          <Button variant="secondary" onClick={() => { if (dirty) { setUnsavedOpen(true) } else { navigate(`/post/${id}`) } }}>
            取消
          </Button>
          <Button loading={updateMut.isPending} onClick={() => updateMut.mutate()}>
            保存修改
          </Button>
        </div>
      </div>

      {/* Unsaved changes dialog */}
      <Dialog open={unsavedOpen} onOpenChange={setUnsavedOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>未保存的更改</DialogTitle><DialogDescription>有未保存的更改，确定离开吗？</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setUnsavedOpen(false)}>继续编辑</Button>
            <Button variant="danger" onClick={() => { navigate(`/post/${id}`) }}>离开</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

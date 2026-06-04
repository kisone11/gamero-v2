import { useState, useRef, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Save, Send, Upload, X, Image as ImageIcon, Video, Zap, Eye, PenLine } from 'lucide-react'
import { logApi } from '@/api/log'
import { uploadFile } from '@/lib/upload'
import { Button, Input, Textarea, Skeleton } from '@/components/ui'
import { Markdown } from '@/components/ui/markdown'
import { toast } from '@/stores/toastStore'
import { cn } from '@/lib/utils'
import type { DevLogType, DevLogVisibility, DevLogStatus } from '@/types/enums'

// ============================================================
// Schema
// ============================================================

const formSchema = z.object({
  title: z.string().min(1, '请输入标题'),
  content: z.string().min(10, '内容至少 10 个字符'),
  log_type: z.string().min(1, '请选择日志类型'),
  version: z.string().optional(),
  visibility: z.string().min(1, '请选择可见性'),
})

type FormValues = z.infer<typeof formSchema>

// ============================================================
// EditDevLogPage
// ============================================================

export default function EditDevLogPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const logId = Number(id)
  const [formInitialized, setFormInitialized] = useState(false)

  const [imageFiles, setImageFiles] = useState<File[]>([])
  const [imagePreviews, setImagePreviews] = useState<string[]>([])
  const [videoFiles, setVideoFiles] = useState<File[]>([])
  const [videoNames, setVideoNames] = useState<string[]>([])
  const [preview, setPreview] = useState(false)
  const [dirty, setDirty] = useState(false)
  const imageInputRef = useRef<HTMLInputElement>(null)
  const videoInputRef = useRef<HTMLInputElement>(null)

  const {
    register,
    handleSubmit,
    formState: { errors, isDirty },
    watch,
    getValues,
    reset,
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      title: '',
      content: '',
      log_type: 'log',
      version: '',
      visibility: 'public',
    },
  })

  const logType = watch('log_type')
  const watchContent = watch('content')

  const logQuery = useQuery({
    queryKey: ['devlog', logId],
    queryFn: () => logApi.get(logId),
    enabled: !isNaN(logId),
  })

  const logData = logQuery.data

  useEffect(() => {
    if (!logData || formInitialized) return
    reset({
      title: logData.title,
      content: logData.content,
      log_type: logData.log_type,
      version: logData.version ?? '',
      visibility: logData.visibility,
    })
    setFormInitialized(true)
  }, [logData, formInitialized, reset])

  useEffect(() => {
    if (formInitialized && (isDirty || imageFiles.length > 0 || videoFiles.length > 0)) {
      setDirty(true)
    }
  }, [formInitialized, isDirty, imageFiles, videoFiles])

  useEffect(() => {
    if (!dirty) return
    const handler = (e: BeforeUnloadEvent) => { e.preventDefault(); e.returnValue = '' }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [dirty])

  const saveDraftMutation = useMutation({
    mutationFn: async (data: FormValues) => {
      await logApi.update(logId, {
        title: data.title,
        content: data.content,
        log_type: data.log_type as DevLogType,
        visibility: data.visibility as DevLogVisibility,
        version: data.version || undefined,
        status: 'draft' as DevLogStatus,
      })

      if (imageFiles.length > 0) {
        const keys = await Promise.all(imageFiles.map((file) => uploadFile('log-image', file, logId)))
        await logApi.uploadImages(logId, keys)
      }

      if (videoFiles.length > 0) {
        const keys = await Promise.all(videoFiles.map((file) => uploadFile('log-video', file, logId)))
        await logApi.uploadVideos(logId, keys)
      }
    },
    onSuccess: () => {
      toast.success('草稿已保存')
      queryClient.invalidateQueries({ queryKey: ['devlog', logId] })
    },
    onError: () => toast.error('保存失败，请重试'),
  })

  const publishMutation = useMutation({
    mutationFn: async (data: FormValues) => {
      await logApi.update(logId, {
        title: data.title,
        content: data.content,
        log_type: data.log_type as DevLogType,
        visibility: data.visibility as DevLogVisibility,
        version: data.version || undefined,
        status: 'published' as DevLogStatus,
      })

      const isDraft = logData?.status === 'draft'
      if (isDraft) await logApi.publish(logId)

      if (imageFiles.length > 0) {
        const keys = await Promise.all(imageFiles.map((file) => uploadFile('log-image', file, logId)))
        await logApi.uploadImages(logId, keys)
      }

      if (videoFiles.length > 0) {
        const keys = await Promise.all(videoFiles.map((file) => uploadFile('log-video', file, logId)))
        await logApi.uploadVideos(logId, keys)
      }
    },
    onSuccess: () => {
      toast.success('日志已发布')
      queryClient.invalidateQueries({ queryKey: ['devlog', logId] })
      navigate(`/devlog/${logId}`)
    },
    onError: () => toast.error('发布失败，请重试'),
  })

  const handleImageChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || [])
    const combined = [...imageFiles, ...files].slice(0, 6)
    setImageFiles(combined)
    setImagePreviews(combined.map((f) => URL.createObjectURL(f)))
    if (imageInputRef.current) imageInputRef.current.value = ''
  }

  const removeImage = (index: number) => {
    setImageFiles((prev) => prev.filter((_, i) => i !== index))
    setImagePreviews((prev) => {
      URL.revokeObjectURL(prev[index])
      return prev.filter((_, i) => i !== index)
    })
  }

  const handleVideoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || [])
    const combined = [...videoFiles, ...files].slice(0, 3)
    setVideoFiles(combined)
    setVideoNames(combined.map((f) => f.name))
    if (videoInputRef.current) videoInputRef.current.value = ''
  }

  const removeVideo = (index: number) => {
    setVideoFiles((prev) => prev.filter((_, i) => i !== index))
    setVideoNames((prev) => prev.filter((_, i) => i !== index))
  }

  const handleDeleteImage = async (key: string) => {
    try {
      await logApi.deleteImage(logId, key)
      toast.success('图片已删除')
      queryClient.invalidateQueries({ queryKey: ['devlog', logId] })
    } catch {
      toast.error('删除图片失败')
    }
  }

  const handleDeleteVideo = async (key: string) => {
    try {
      await logApi.deleteVideo(logId, key)
      toast.success('视频已删除')
      queryClient.invalidateQueries({ queryKey: ['devlog', logId] })
    } catch {
      toast.error('删除视频失败')
    }
  }

  const onSaveDraft = (data: FormValues) => saveDraftMutation.mutate(data)
  const onPublish = (data: FormValues) => publishMutation.mutate(data)

  useEffect(() => {
    return () => { imagePreviews.forEach((u) => URL.revokeObjectURL(u)) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (logQuery.isLoading) {
    return (
      <div className="max-w-3xl mx-auto px-6 py-8">
        <Skeleton className="h-10 w-48 mb-8" />
        <div className="space-y-6">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-40 w-full" />
          ))}
        </div>
      </div>
    )
  }

  if (logQuery.isError || !logData) {
    return (
      <div className="max-w-3xl mx-auto px-6 py-8 min-h-[60vh] flex items-center justify-center">
        <div className="flex flex-col items-center justify-center text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/[0.03] flex items-center justify-center">
            <Zap className="w-7 h-7 text-text-muted" />
          </div>
          <p className="text-[15px] font-semibold text-text-secondary mb-1">加载失败</p>
          <p className="text-[13px] text-text-muted mb-6">无法加载开发日志信息，请检查后重试</p>
          <Button variant="secondary" size="sm" onClick={() => logQuery.refetch()}>重新加载</Button>
        </div>
      </div>
    )
  }

  const isPending = saveDraftMutation.isPending || publishMutation.isPending

  return (
    <div className="max-w-3xl mx-auto px-6 py-8">
      <h1 className="text-[22px] font-bold text-text-primary mb-8">编辑日志 · {logData?.title || '...'}</h1>

      <form className="space-y-6" onSubmit={(e) => e.preventDefault()}>
        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">基本信息</h2>
          <div className="space-y-4">
            <Input label="标题" placeholder="输入日志标题" error={errors.title?.message} {...register('title')} fullWidth />
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">内容</label>
                <div className="flex gap-1">
                  <button
                    type="button"
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
                    type="button"
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
                  {watchContent ? (
                    <Markdown content={watchContent} />
                  ) : (
                    <p className="text-text-muted text-[14px]">暂无内容</p>
                  )}
                </div>
              ) : (
                <Textarea
                  placeholder="撰写日志内容（支持 Markdown，至少 10 个字符）"
                  error={errors.content?.message}
                  {...register('content')}
                  rows={14}
                  fullWidth
                />
              )}
            </div>
          </div>
        </div>

        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">日志设置</h2>
          <div className="space-y-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">日志类型</label>
              <select className="h-10 px-3 bg-surface-void border border-white/[0.04] rounded-xl text-[14px] text-text-primary focus:outline-none focus:border-amber" {...register('log_type')}>
                <option value="log">开发日志</option>
                <option value="release">发布</option>
              </select>
            </div>
            {logType === 'release' && (
              <Input label="版本号" placeholder="例如：1.0.0" {...register('version')} fullWidth />
            )}
          </div>
        </div>

        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">可见性</h2>
          <select className="h-10 px-3 bg-surface-void border border-white/[0.04] rounded-xl text-[14px] text-text-primary focus:outline-none focus:border-amber" {...register('visibility')}>
            <option value="public">公开</option>
            <option value="members_only">仅成员</option>
          </select>
        </div>

        {logData.image_urls && logData.image_urls.length > 0 && (
          <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
            <h2 className="text-[15px] font-semibold text-text-primary mb-4">已上传图片</h2>
            <div className="grid grid-cols-3 gap-3">
              {logData.image_urls.map((url, i) => (
                <div key={i} className="relative aspect-video rounded-xl overflow-hidden border border-white/[0.04] group">
                  <img src={url} alt="" className="w-full h-full object-cover" />
                  <button
                    type="button"
                    onClick={() => handleDeleteImage(logData.image_keys?.[i] || '')}
                    className="absolute top-1 right-1 p-0.5 bg-black/50 rounded-full text-white hover:bg-black/70 opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {logData.video_urls && logData.video_urls.length > 0 && (
          <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
            <h2 className="text-[15px] font-semibold text-text-primary mb-4">已上传视频</h2>
            <div className="space-y-3">
              {logData.video_urls.map((url, i) => (
                <div key={i} className="flex items-center gap-3 p-3 rounded-xl border border-white/[0.04] group">
                  <Video className="h-5 w-5 text-text-muted shrink-0" />
                  <span className="text-[14px] text-text-primary truncate flex-1">{url.split('/').pop() || url}</span>
                  <button
                    type="button"
                    onClick={() => handleDeleteVideo(logData.video_keys?.[i] || '')}
                    className="p-0.5 text-text-muted hover:text-danger opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">图片{imageFiles.length === 0 ? '（最多 6 张）' : ''}</h2>
          <input ref={imageInputRef} type="file" accept="image/*" multiple onChange={handleImageChange} className="hidden" />
          {imagePreviews.length > 0 && (
            <div className="grid grid-cols-3 gap-3 mb-4">
              {imagePreviews.map((preview, i) => (
                <div key={i} className="relative aspect-video rounded-xl overflow-hidden border border-white/[0.04]">
                  <img src={preview} alt="" className="w-full h-full object-cover" />
                  <button type="button" onClick={() => removeImage(i)} className="absolute top-1 right-1 p-0.5 bg-black/50 rounded-full text-white hover:bg-black/70">
                    <X className="h-3 w-3" />
                  </button>
                </div>
              ))}
            </div>
          )}
          {imageFiles.length < 6 && (
            <button type="button" onClick={() => imageInputRef.current?.click()} className="w-full h-24 flex items-center justify-center border-2 border-dashed border-white/[0.04] rounded-xl hover:border-amber transition-colors">
              <ImageIcon className="h-5 w-5 text-text-muted mr-2" />
              <span className="text-[13px] text-text-muted font-semibold">添加图片</span>
            </button>
          )}
          <p className="text-[12px] text-text-muted mt-2">{imageFiles.length}/6 张新图片</p>
        </div>

        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">视频（最多 3 个）</h2>
          <input ref={videoInputRef} type="file" accept="video/*" multiple onChange={handleVideoChange} className="hidden" />
          {videoNames.length > 0 && (
            <div className="space-y-3 mb-4">
              {videoNames.map((name, i) => (
                <div key={i} className="flex items-center gap-3 p-3 rounded-xl border border-white/[0.04]">
                  <Video className="h-5 w-5 text-text-muted shrink-0" />
                  <span className="text-[14px] text-text-primary truncate flex-1">{name}</span>
                  <button type="button" onClick={() => removeVideo(i)} className="p-0.5 text-text-muted hover:text-danger">
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
          {videoFiles.length < 3 && (
            <button type="button" onClick={() => videoInputRef.current?.click()} className="w-full h-24 flex items-center justify-center border-2 border-dashed border-white/[0.04] rounded-xl hover:border-amber transition-colors">
              <Video className="h-5 w-5 text-text-muted mr-2" />
              <span className="text-[13px] text-text-muted font-semibold">添加视频</span>
            </button>
          )}
          <p className="text-[12px] text-text-muted mt-2">{videoFiles.length}/3 个新视频</p>
        </div>

        <div className="flex items-center gap-3 justify-end">
          {(saveDraftMutation.isPending || publishMutation.isPending) && (
            <span className="text-[12px] text-text-muted animate-pulse mr-auto">自动保存中...</span>
          )}
          <Button type="button" variant="ghost" onClick={() => navigate(`/devlog/${logId}`)}>取消</Button>
          {(logData.status === 'draft') && (
            <Button type="button" variant="secondary" onClick={handleSubmit(onSaveDraft)} loading={saveDraftMutation.isPending}>
              <Save className="h-4 w-4" />保存草稿
            </Button>
          )}
          <Button type="button" onClick={handleSubmit(onPublish)} loading={publishMutation.isPending}>
            <Send className="h-4 w-4" />发布
          </Button>
        </div>
      </form>
    </div>
  )
}

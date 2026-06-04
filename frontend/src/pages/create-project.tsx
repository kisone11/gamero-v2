import { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, X, Upload } from 'lucide-react'
import { projectApi } from '@/api/project'
import { uploadFile } from '@/lib/upload'
import { Button, Input, Textarea, Badge } from '@/components/ui'
import { toast } from '@/stores/toastStore'
import type { ProjectGenre, ProjectStatus } from '@/types/enums'
import { GENRE_OPTIONS, STATUS_OPTIONS } from '@/lib/constants'

// ============================================================
// Schema
// ============================================================

const formSchema = z.object({
  name: z.string().min(1, '请输入项目名称'),
  description: z.string().min(10, '项目描述至少 10 个字符'),
  genre: z.string().min(1, '请选择项目类型'),
  status: z.string().min(1, '请选择项目状态'),
  engine: z.string().optional(),
  demo_url: z.string().optional(),
  store_url: z.string().optional(),
})

type FormValues = z.infer<typeof formSchema>

// ============================================================
// Constants
// ============================================================

const STYLE_TAG_OPTIONS = [
  { value: '2D', label: '2D' }, { value: '3D', label: '3D' },
  { value: 'pixel', label: '像素' }, { value: 'realist', label: '写实' },
  { value: 'cartoon', label: '卡通' }, { value: 'cyberpunk', label: '赛博朋克' },
  { value: 'fantasy', label: '奇幻' }, { value: 'scifi', label: '科幻' },
]

const ENGINE_OPTIONS = [
  { value: 'Unity', label: 'Unity' }, { value: 'Unreal Engine', label: 'Unreal Engine' },
  { value: 'Godot', label: 'Godot' }, { value: 'Cocos', label: 'Cocos' },
  { value: 'GameMaker', label: 'GameMaker' }, { value: 'RPG Maker', label: 'RPG Maker' },
  { value: '自研引擎', label: '自研引擎' }, { value: '其他', label: '其他' },
]

const PLATFORM_OPTIONS = ['PC', 'Mac', 'iOS', 'Android', 'Web', 'PS5', 'Xbox', 'Switch']

// ============================================================
// CreateProjectPage
// ============================================================

export default function CreateProjectPage() {
  const navigate = useNavigate()
  const [styleTags, setStyleTags] = useState<string[]>([])
  const [tagInput, setTagInput] = useState('')
  const [platforms, setPlatforms] = useState<string[]>([])
  const [coverFile, setCoverFile] = useState<File | null>(null)
  const [coverPreview, setCoverPreview] = useState<string>('')
  const [screenshotFiles, setScreenshotFiles] = useState<File[]>([])
  const [screenshotPreviews, setScreenshotPreviews] = useState<string[]>([])
  const [videoFile, setVideoFile] = useState<File | null>(null)
  const [videoPreview, setVideoPreview] = useState<string>('')
  const coverInputRef = useRef<HTMLInputElement>(null)
  const screenshotInputRef = useRef<HTMLInputElement>(null)
  const videoInputRef = useRef<HTMLInputElement>(null)
  const [dirty, setDirty] = useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors },
    watch,
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
      description: '',
      genre: '',
      status: '',
      engine: '',
      demo_url: '',
      store_url: '',
    },
  })

  const watchedName = watch('name')
  const watchedDesc = watch('description')

  useEffect(() => {
    if (watchedName || watchedDesc) setDirty(true)
  }, [watchedName, watchedDesc])

  useEffect(() => {
    if (coverFile || screenshotFiles.length > 0 || videoFile || styleTags.length > 0 || platforms.length > 0) {
      setDirty(true)
    }
  }, [coverFile, screenshotFiles, videoFile, styleTags, platforms])

  useEffect(() => {
    if (!dirty) return
    const handler = (e: BeforeUnloadEvent) => { e.preventDefault(); e.returnValue = '' }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [dirty])

  const createMutation = useMutation({
    mutationFn: async (data: FormValues) => {
      const project = await projectApi.create({
        name: data.name,
        description: data.description,
        genre: data.genre as ProjectGenre,
        status: data.status as ProjectStatus,
        style_tags: styleTags.length > 0 ? styleTags : undefined,
        platform: platforms.length > 0 ? platforms : undefined,
        engine: data.engine || undefined,
        demo_url: data.demo_url || undefined,
        store_url: data.store_url || undefined,
      })

      if (coverFile && project.id) {
        const coverKey = await uploadFile('cover', coverFile, project.id)
        await projectApi.uploadCover(project.id, coverKey)
      }

      if (screenshotFiles.length > 0 && project.id) {
        const keys = await Promise.all(
          screenshotFiles.map((file) => uploadFile('screenshot', file, project.id)),
        )
        await projectApi.uploadScreenshots(project.id, keys)
      }

      if (videoFile && project.id) {
        const videoKey = await uploadFile('project-video', videoFile, project.id)
        await projectApi.uploadVideo(project.id, videoKey)
      }

      return project
    },
    onSuccess: (project) => {
      toast.success('项目创建成功')
      navigate(`/p/${project.slug}`)
    },
    onError: () => {
      toast.error('创建项目失败，请重试')
    },
  })

  const togglePlatform = (p: string) => {
    setPlatforms((prev) => (prev.includes(p) ? prev.filter((x) => x !== p) : [...prev, p]))
  }

  const addTag = () => {
    const tag = tagInput.trim()
    if (!tag || styleTags.includes(tag) || styleTags.length >= 5) return
    setStyleTags([...styleTags, tag])
    setTagInput('')
  }
  const removeTag = (tag: string) => setStyleTags(styleTags.filter((t) => t !== tag))

  const handleCoverChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setCoverFile(file)
    setCoverPreview(URL.createObjectURL(file))
  }

  const removeCover = () => {
    if (coverPreview) URL.revokeObjectURL(coverPreview)
    setCoverFile(null)
    setCoverPreview('')
    if (coverInputRef.current) coverInputRef.current.value = ''
  }

  const handleScreenshotsChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || [])
    setScreenshotFiles((prev) => [...prev, ...files])
    setScreenshotPreviews((prev) => [...prev, ...files.map((f) => URL.createObjectURL(f))])
    if (screenshotInputRef.current) screenshotInputRef.current.value = ''
  }

  const handleVideoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setVideoFile(file)
    setVideoPreview(URL.createObjectURL(file))
  }

  const removeVideo = () => {
    if (videoPreview) URL.revokeObjectURL(videoPreview)
    setVideoFile(null)
    setVideoPreview('')
    if (videoInputRef.current) videoInputRef.current.value = ''
  }

  const removeScreenshot = (index: number) => {
    setScreenshotFiles((prev) => prev.filter((_, i) => i !== index))
    setScreenshotPreviews((prev) => {
      URL.revokeObjectURL(prev[index])
      return prev.filter((_, i) => i !== index)
    })
  }

  const onSubmit = (data: FormValues) => createMutation.mutate(data)

  return (
    <div className="max-w-3xl mx-auto px-6 py-8">
      <h1 className="text-[22px] font-bold text-text-primary mb-8">创建项目</h1>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Basic Info */}
        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">基本信息</h2>
          <div className="space-y-4">
            <Input
              label="项目名称"
              placeholder="输入项目名称"
              error={errors.name?.message}
              {...register('name')}
              fullWidth
            />
            <Textarea
              label="项目描述"
              placeholder="简要描述你的项目（至少 10 个字符）"
              error={errors.description?.message}
              {...register('description')}
              fullWidth
            />
          </div>
        </div>

        {/* Genre & Status */}
        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">分类与状态</h2>
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">
                项目类型
              </label>
              <select
                className="h-10 px-3 bg-surface-void border border-white/[0.04] rounded-xl text-[14px] text-text-primary focus:outline-none focus:border-amber"
                {...register('genre')}
              >
                <option value="">请选择</option>
                {GENRE_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
              {errors.genre && (
                <p className="text-[13px] text-danger">{errors.genre.message}</p>
              )}
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">
                项目状态
              </label>
              <select
                className="h-10 px-3 bg-surface-void border border-white/[0.04] rounded-xl text-[14px] text-text-primary focus:outline-none focus:border-amber"
                {...register('status')}
              >
                <option value="">请选择</option>
                {STATUS_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
              {errors.status && (
                <p className="text-[13px] text-danger">{errors.status.message}</p>
              )}
            </div>
          </div>
        </div>

        {/* Style Tags */}
        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">风格标签</h2>
          <div className="flex gap-2">
            <input
              type="text"
              placeholder="输入标签名，按回车添加..."
              className="flex-1 h-10 px-3.5 bg-surface-void border border-white/[0.06] rounded-xl text-[13px] text-text-primary placeholder:text-text-muted focus:outline-none focus:border-amber/40"
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addTag() } }}
            />
            <Button type="button" variant="secondary" size="sm" onClick={addTag}>
              <Plus className="h-4 w-4" />添加
            </Button>
          </div>
          {styleTags.length > 0 && (
            <div className="flex flex-wrap gap-2 mt-3">
              {styleTags.map((tag) => (
                <Badge key={tag} variant="info" size="md">
                  {tag}
                  <button type="button" onClick={() => removeTag(tag)} className="ml-1.5 hover:text-danger">
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              ))}
            </div>
          )}
          <p className="text-[12px] text-text-muted mt-2">{styleTags.length}/5 个标签</p>
        </div>

        {/* Platform & Engine */}
        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">平台与引擎</h2>
          <div className="space-y-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">
                目标平台
              </label>
              <div className="flex flex-wrap gap-2">
                {PLATFORM_OPTIONS.map((p) => (
                  <button
                    key={p}
                    type="button"
                    onClick={() => togglePlatform(p)}
                    className={`px-3 py-1.5 rounded-xl border text-[13px] font-semibold transition-colors ${
                      platforms.includes(p)
                        ? 'bg-amber text-surface-void border-amber'
                        : 'bg-surface-void text-text-secondary border-white/[0.04] hover:border-amber'
                    }`}
                  >
                    {p}
                  </button>
                ))}
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-text-secondary uppercase tracking-wider">
                开发引擎
              </label>
              <select
                className="h-10 px-3 bg-surface-void border border-white/[0.04] rounded-xl text-[14px] text-text-primary focus:outline-none focus:border-amber"
                {...register('engine')}
              >
                <option value="">不指定</option>
                {ENGINE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {/* Cover */}
        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">封面图片</h2>
          <input
            ref={coverInputRef}
            type="file"
            accept="image/*"
            onChange={handleCoverChange}
            className="hidden"
          />
          {coverPreview ? (
            <div className="relative w-full h-48 rounded-xl overflow-hidden border border-white/[0.04]">
              <img src={coverPreview} alt="" className="w-full h-full object-cover" />
              <button
                type="button"
                onClick={removeCover}
                className="absolute top-2 right-2 p-1 bg-black/50 rounded-full text-white hover:bg-black/70"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => coverInputRef.current?.click()}
              className="w-full h-48 flex flex-col items-center justify-center border-2 border-dashed border-white/[0.04] rounded-xl hover:border-amber transition-colors"
            >
              <Upload className="h-8 w-8 text-text-muted mb-2" />
              <span className="text-[13px] text-text-muted font-semibold">点击上传封面</span>
            </button>
          )}
        </div>

        {/* Screenshots */}
        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">游戏截图</h2>
          <input
            ref={screenshotInputRef}
            type="file"
            accept="image/*"
            multiple
            onChange={handleScreenshotsChange}
            className="hidden"
          />
          {screenshotPreviews.length > 0 && (
            <div className="grid grid-cols-3 gap-3 mb-4">
              {screenshotPreviews.map((preview, i) => (
                <div
                  key={i}
                  className="relative aspect-video rounded-xl overflow-hidden border border-white/[0.04]"
                >
                  <img src={preview} alt="" className="w-full h-full object-cover" />
                  <button
                    type="button"
                    onClick={() => removeScreenshot(i)}
                    className="absolute top-1 right-1 p-0.5 bg-black/50 rounded-full text-white hover:bg-black/70"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              ))}
            </div>
          )}
          <button
            type="button"
            onClick={() => screenshotInputRef.current?.click()}
            className="w-full h-24 flex items-center justify-center border-2 border-dashed border-white/[0.04] rounded-xl hover:border-amber transition-colors"
          >
            <Upload className="h-5 w-5 text-text-muted mr-2" />
            <span className="text-[13px] text-text-muted font-semibold">添加截图</span>
          </button>
        </div>

        {/* Video */}
        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">宣传视频</h2>
          <input
            ref={videoInputRef}
            type="file"
            accept="video/mp4,video/quicktime,video/webm"
            onChange={handleVideoChange}
            className="hidden"
          />
          {videoPreview ? (
            <div className="relative w-full rounded-xl overflow-hidden border border-white/[0.04]">
              <video src={videoPreview} controls className="w-full max-h-64" />
              <button
                type="button"
                onClick={removeVideo}
                className="absolute top-2 right-2 p-1 bg-black/50 rounded-full text-white hover:bg-black/70"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => videoInputRef.current?.click()}
              className="w-full h-24 flex items-center justify-center border-2 border-dashed border-white/[0.04] rounded-xl hover:border-amber transition-colors"
            >
              <Upload className="h-5 w-5 text-text-muted mr-2" />
              <span className="text-[13px] text-text-muted font-semibold">添加宣传视频</span>
            </button>
          )}
        </div>

        {/* Links */}
        <div className="bg-surface-card border border-white/[0.04] rounded-xl p-5">
          <h2 className="text-[15px] font-semibold text-text-primary mb-4">外部链接</h2>
          <div className="space-y-4">
            <Input
              label="试玩链接"
              placeholder="https://..."
              {...register('demo_url')}
              fullWidth
            />
            <Input
              label="商店链接"
              placeholder="https://..."
              {...register('store_url')}
              fullWidth
            />
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-3 justify-end">
          <Button type="button" variant="ghost" onClick={() => navigate(-1)}>
            取消
          </Button>
          <Button type="submit" loading={createMutation.isPending}>
            创建项目
          </Button>
        </div>
      </form>
    </div>
  )
}

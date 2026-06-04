import { useState, useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { X } from 'lucide-react'
import { communityApi } from '@/api/community'
import { uploadFile } from '@/lib/upload'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { toast } from '@/stores/toastStore'
import type { Topic } from '@/types/api'

export interface CreatePostModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  topics: Topic[]
}

export function CreatePostModal({ open, onOpenChange, topics }: CreatePostModalProps) {
  const queryClient = useQueryClient()
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [topicId, setTopicId] = useState<number | null>(null)
  const [imageFiles, setImageFiles] = useState<File[]>([])
  const [imagePreviews, setImagePreviews] = useState<string[]>([])
  const [videoFile, setVideoFile] = useState<File | null>(null)
  const [videoPreview, setVideoPreview] = useState<string>('')
  const [uploadingIndex, setUploadingIndex] = useState(-1)
  const imageInputRef = useRef<HTMLInputElement>(null)
  const videoInputRef = useRef<HTMLInputElement>(null)

  const handleImagesChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || [])
    setImageFiles((prev) => [...prev, ...files])
    setImagePreviews((prev) => [...prev, ...files.map((f) => URL.createObjectURL(f))])
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

  const createMutation = useMutation({
    mutationFn: async () => {
      const post = await communityApi.create({
        title,
        content,
        topic_ids: topicId ? [topicId] : [],
      })
      if (!post?.id) return post

      // 上传图片（逐个上传，失败则删帖回滚）
      for (let i = 0; i < imageFiles.length; i++) {
        const file = imageFiles[i]
        setUploadingIndex(i)
        try {
          const key = await uploadFile('post-image', file, post.id)
          await communityApi.saveImage(post.id, key)
        } catch (e: any) {
          // 图片上传失败，删除已创建的帖子
          try { await communityApi.delete(post.id) } catch {}
          throw new Error(e?.response?.data?.message || e?.message || '图片上传失败，帖子已取消')
        }
      }
      setUploadingIndex(-1)
      // 上传视频
      if (videoFile) {
        try {
          const key = await uploadFile('post-video', videoFile, post.id)
          await communityApi.saveVideo(post.id, key)
        } catch (e: any) {
          try { await communityApi.delete(post.id) } catch {}
          throw new Error(e?.response?.data?.message || e?.message || '视频上传失败，帖子已取消')
        }
      }
      return post
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['community-posts'] })
      setTitle('')
      setContent('')
      setTopicId(null)
      setImageFiles([])
      setImagePreviews([])
      setVideoFile(null)
      setVideoPreview('')
      onOpenChange(false)
      toast.success('发帖成功')
    },
    onError: (err: any) => toast.error(err?.message || '发帖失败'),
  })

  const canSubmit = title.trim().length > 0 && content.trim().length > 0

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>发布新帖子</DialogTitle>
          <DialogDescription>分享你的开发进度、想法或问题</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <Input
            label="标题"
            placeholder="帖子标题"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            fullWidth
          />
          <div className="flex flex-col gap-1.5">
            <label className="text-meta font-semibold text-text-muted uppercase tracking-wider">
              话题
            </label>
            <select
              value={topicId ?? ''}
              onChange={(e) => setTopicId(e.target.value ? Number(e.target.value) : null)}
              className="h-10 px-3 bg-surface-void border border-white/[0.04] rounded-xl text-text-primary text-body focus:outline-none focus:border-amber focus:shadow-[0_0_0_2px_rgba(245,166,35,0.2)]"
            >
              <option value="">选择话题（可选）</option>
              {topics.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} ({t.post_count})
                </option>
              ))}
            </select>
          </div>
          <Textarea
            label="内容"
            placeholder="分享你的想法……"
            value={content}
            onChange={(e) => setContent(e.target.value)}
            rows={6}
            fullWidth
          />

          {/* 图片上传 */}
          <input
            ref={imageInputRef}
            type="file"
            accept="image/*"
            multiple
            onChange={handleImagesChange}
            className="hidden"
          />
          {imagePreviews.length > 0 && (
            <div className="flex gap-2 flex-wrap">
              {imagePreviews.map((preview, i) => (
                <div key={i} className="relative w-20 h-20 rounded-xl overflow-hidden border border-white/[0.04]">
                  <img src={preview} alt="" className="w-full h-full object-cover" />
                  {uploadingIndex === i && (
                    <div className="absolute inset-0 bg-black/60 flex items-center justify-center">
                      <span className="text-caption text-white font-medium">上传中...</span>
                    </div>
                  )}
                  <button
                    type="button"
                    onClick={() => removeImage(i)}
                    className="absolute top-0.5 right-0.5 p-0.5 bg-black/50 rounded-full text-white hover:bg-black/70"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              ))}
            </div>
          )}
          <button
            type="button"
            onClick={() => imageInputRef.current?.click()}
            className="text-body text-amber hover:text-amber-light transition-colors"
          >
            + 添加图片
          </button>

          {/* 视频上传 */}
          <input
            ref={videoInputRef}
            type="file"
            accept="video/mp4,video/quicktime,video/webm"
            onChange={handleVideoChange}
            className="hidden"
          />
          {videoPreview ? (
            <div className="relative w-full rounded-xl overflow-hidden border border-white/[0.04]">
              <video src={videoPreview} controls className="w-full max-h-40" />
              <button
                type="button"
                onClick={removeVideo}
                className="absolute top-1 right-1 p-0.5 bg-black/50 rounded-full text-white hover:bg-black/70"
              >
                <X className="h-3 w-3" />
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => videoInputRef.current?.click()}
              className="text-body text-amber hover:text-amber-light transition-colors"
            >
              + 添加视频
            </button>
          )}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            loading={createMutation.isPending}
            disabled={!canSubmit}
            onClick={() => createMutation.mutate()}
          >
            发布
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

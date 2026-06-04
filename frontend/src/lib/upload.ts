// 统一文件上传（获取预签名 URL 后直传 COS）
export type UploadType =
  | 'avatar'
  | 'cover'
  | 'screenshot'
  | 'log-image'
  | 'log-video'
  | 'post-image'
  | 'post-video'
  | 'project-video'

const IMAGE_TYPES: UploadType[] = ['avatar', 'cover', 'screenshot', 'log-image', 'post-image']
const VIDEO_TYPES: UploadType[] = ['log-video', 'post-video', 'project-video']

export async function uploadFile(type: UploadType, file: File, refId?: number): Promise<string> {
  // 0. 文件大小校验
  if (IMAGE_TYPES.includes(type) && file.size > 10 * 1024 * 1024) {
    throw new Error('图片文件大小不能超过 10MB')
  }
  if (VIDEO_TYPES.includes(type) && file.size > 100 * 1024 * 1024) {
    throw new Error('视频文件大小不能超过 100MB')
  }

  // 1. 获取预签名上传 URL
  const http = (await import('@/lib/http')).default
  const res = await http.post<{ data: { upload_url: string; key: string; expires_at: number } }>('/upload/token', {
    type,
    filename: file.name,
    resource_id: refId,
    content_type: file.type,
  })
  const { upload_url, key } = res.data.data

  // 2. 直传 COS（需要 COS 配置 CORS 允许 PUT）
  const putRes = await fetch(upload_url, {
    method: 'PUT',
    body: file,
    headers: { 'Content-Type': file.type },
  })
  if (!putRes.ok) {
    throw new Error(`文件上传失败 (${putRes.status})`)
  }
  return key
}

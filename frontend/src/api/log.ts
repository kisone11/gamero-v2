import http, { extractData } from '@/lib/http'
import type {
  ApiResponse,
  PageData,
  DevLogDetail,
  LogComment,
  CreateDevLogReq,
  UpdateDevLogReq,
  CreateCommentReq,
} from '@/types/api'

export interface ListLogsParams {
  page?: number
  page_size?: number
  project_id?: number
  author_id?: number
  log_type?: string
  status?: string
  visibility?: string
  keyword?: string
  sort?: string
}

export const logApi = {
  list: (params?: ListLogsParams) => {
    const { project_id, author_id, ...rest } = params || {}
    if (project_id) {
      return http.get<ApiResponse<PageData<DevLogDetail>>>(`/projects/${project_id}/logs`, { params: rest }).then(extractData)
    }
    if (author_id) {
      // 查询特定用户的日志走 /me/logs（后端只有这个列表端点）
      return http.get<ApiResponse<PageData<DevLogDetail>>>('/me/logs', { params: { ...rest, page: params?.page, page_size: params?.page_size } }).then(extractData)
    }
    return http.get<ApiResponse<PageData<DevLogDetail>>>('/discover/logs', { params }).then(extractData)
  },

  get: (id: number) =>
    http.get<ApiResponse<DevLogDetail>>(`/logs/${id}`).then(extractData),

  create: (projectId: number, data: CreateDevLogReq) =>
    http
      .post<ApiResponse<DevLogDetail>>(`/projects/${projectId}/logs`, data)
      .then(extractData),

  update: (id: number, data: UpdateDevLogReq) =>
    http.patch<ApiResponse<DevLogDetail>>(`/logs/${id}`, data).then(extractData),

  delete: (id: number) =>
    http.delete<ApiResponse<null>>(`/logs/${id}`).then(extractData),

  uploadCover: (id: number, key: string) =>
    http.patch<ApiResponse<{ cover_url: string }>>(`/logs/${id}/cover`, { key }).then(extractData),

  uploadImages: (id: number, keys: string[]) =>
    Promise.all(keys.map((key) =>
      http.post<ApiResponse<{ url: string; key: string }>>(`/logs/${id}/images`, { key }).then(extractData)
    )),

  deleteImage: (id: number, key: string) =>
    http.delete<ApiResponse<null>>(`/logs/${id}/images`, { data: { key } }).then(extractData),

  uploadVideos: (id: number, keys: string[]) =>
    Promise.all(keys.map((key) =>
      http.put<ApiResponse<null>>(`/logs/${id}/videos`, { key }).then(extractData)
    )),

  deleteVideo: (id: number, key: string) =>
    http.delete<ApiResponse<null>>(`/logs/${id}/videos`, { data: { key } }).then(extractData),

  like: (id: number) =>
    http.post<ApiResponse<null>>(`/logs/${id}/like`).then(extractData),

  unlike: (id: number) =>
    http.delete<ApiResponse<null>>(`/logs/${id}/like`).then(extractData),

  collect: (id: number) =>
    http.post<ApiResponse<null>>(`/logs/${id}/collect`).then(extractData),

  uncollect: (id: number) =>
    http.delete<ApiResponse<null>>(`/logs/${id}/collect`).then(extractData),

  getComments: (id: number, page = 1, pageSize = 20) =>
    http
      .get<ApiResponse<PageData<LogComment>>>(`/logs/${id}/comments`, {
        params: { page, page_size: pageSize },
      })
      .then(extractData),

  createComment: (id: number, data: CreateCommentReq) =>
    http.post<ApiResponse<LogComment>>(`/logs/${id}/comments`, data).then(extractData),

  deleteComment: (id: number, commentId: number) =>
    http.delete<ApiResponse<null>>(`/logs/${id}/comments/${commentId}`).then(extractData),

  likeComment: (id: number, commentId: number) =>
    http.post<ApiResponse<null>>(`/logs/${id}/comments/${commentId}/like`).then(extractData),

  unlikeComment: (id: number, commentId: number) =>
    http.delete<ApiResponse<null>>(`/logs/${id}/comments/${commentId}/like`).then(extractData),

  // alias for getComments with sort param
  listComments: (id: number, page = 1, pageSize = 20, sort: 'latest' | 'hot' = 'latest') =>
    http
      .get<ApiResponse<PageData<LogComment>>>(`/logs/${id}/comments`, {
        params: { page, page_size: pageSize, sort },
      })
      .then(extractData),

  publish: (id: number) =>
    http.post<ApiResponse<DevLogDetail>>(`/logs/${id}/publish`).then(extractData),
}

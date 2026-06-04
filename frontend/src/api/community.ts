import http, { extractData } from '@/lib/http'
import type {
  ApiResponse,
  PageData,
  Post,
  PostComment,
  Topic,
  Report,
  CreatePostReq,
  UpdatePostReq,
  CreateCommentReq,
  CreateReportReq,
} from '@/types/api'

export interface ListPostsParams {
  page?: number
  page_size?: number
  topic_id?: number
  author_id?: number
  keyword?: string
  sort?: 'latest' | 'hot'
}

export const communityApi = {
  // 话题
  listTopics: () =>
    http.get<ApiResponse<Topic[]>>('/community/topics').then(extractData),

  // 帖子
  list: (params?: ListPostsParams) =>
    http.get<ApiResponse<PageData<Post>>>('/community/posts', { params }).then(extractData),

  getUserPosts: (userId: number, page = 1, pageSize = 10) =>
    http.get<ApiResponse<PageData<Post>>>(`/users/${userId}/posts`, { params: { page, page_size: pageSize } }).then(extractData),

  get: (id: number) =>
    http.get<ApiResponse<Post>>(`/community/posts/${id}`).then(extractData),

  create: (data: CreatePostReq) =>
    http.post<ApiResponse<Post>>('/community/posts', data).then(extractData),

  update: (id: number, data: UpdatePostReq) =>
    http.patch<ApiResponse<Post>>(`/community/posts/${id}`, data).then(extractData),

  delete: (id: number) =>
    http.delete<ApiResponse<null>>(`/community/posts/${id}`).then(extractData),

  saveImage: (postId: number, key: string) =>
    http.put<ApiResponse<null>>(`/community/posts/${postId}/images`, { key }).then(extractData),

  saveVideo: (postId: number, key: string) =>
    http.put<ApiResponse<null>>(`/community/posts/${postId}/videos`, { key }).then(extractData),

  deleteImage: (postId: number, key: string) =>
    http.delete<ApiResponse<null>>(`/community/posts/${postId}/images`, { data: { key } }).then(extractData),

  deleteVideo: (postId: number, key: string) =>
    http.delete<ApiResponse<null>>(`/community/posts/${postId}/videos`, { data: { key } }).then(extractData),

  like: (id: number) =>
    http.post<ApiResponse<null>>(`/community/posts/${id}/like`).then(extractData),

  unlike: (id: number) =>
    http.delete<ApiResponse<null>>(`/community/posts/${id}/like`).then(extractData),

  collect: (id: number) =>
    http.post<ApiResponse<null>>(`/community/posts/${id}/collect`).then(extractData),

  uncollect: (id: number) =>
    http.delete<ApiResponse<null>>(`/community/posts/${id}/collect`).then(extractData),

  // 评论
  getComments: (id: number, page = 1, pageSize = 20, sort?: 'latest' | 'hot') =>
    http
      .get<ApiResponse<PageData<PostComment>>>(`/community/posts/${id}/comments`, {
        params: { page, page_size: pageSize, sort },
      })
      .then(extractData),

  // 通用举报
  createReport: (data: CreateReportReq) =>
    http.post<ApiResponse<null>>('/reports', data).then(extractData),

  getMyReports: (page = 1, pageSize = 20) =>
    http.get<ApiResponse<PageData<Report>>>('/me/reports', { params: { page, page_size: pageSize } }).then(extractData),

  createComment: (id: number, data: CreateCommentReq) =>
    http.post<ApiResponse<PostComment>>(`/community/posts/${id}/comments`, data).then(extractData),

  updateComment: (postId: number, commentId: number, content: string) =>
    http
      .patch<ApiResponse<PostComment>>(`/community/posts/${postId}/comments/${commentId}`, { content })
      .then(extractData),

  deleteComment: (id: number, commentId: number) =>
    http
      .delete<ApiResponse<null>>(`/community/posts/${id}/comments/${commentId}`)
      .then(extractData),

  likeComment: (id: number, commentId: number) =>
    http
      .post<ApiResponse<null>>(`/community/posts/${id}/comments/${commentId}/like`)
      .then(extractData),

  unlikeComment: (id: number, commentId: number) =>
    http
      .delete<ApiResponse<null>>(`/community/posts/${id}/comments/${commentId}/like`)
      .then(extractData),
}

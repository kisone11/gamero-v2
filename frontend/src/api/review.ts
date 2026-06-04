import http, { extractData } from '@/lib/http'
import type {
  ApiResponse,
  PageData,
  ReviewItem,
  ReviewRatingSummary,
  CreateReviewReq,
} from '@/types/api'

export const reviewApi = {
  list: (projectId: number, page = 1, pageSize = 20) =>
    http
      .get<ApiResponse<PageData<ReviewItem>>>(`/projects/${projectId}/reviews`, {
        params: { page, page_size: pageSize },
      })
      .then(extractData),

  getSummary: (projectId: number) =>
    http
      .get<ApiResponse<ReviewRatingSummary>>(`/projects/${projectId}/reviews/summary`)
      .then(extractData),

  create: (projectId: number, data: CreateReviewReq) =>
    http
      .post<ApiResponse<ReviewItem>>(`/projects/${projectId}/reviews`, data)
      .then(extractData),

  delete: (projectId: number, reviewId: number) =>
    http
      .delete<ApiResponse<null>>(`/projects/${projectId}/reviews/${reviewId}`)
      .then(extractData),

  like: (projectId: number, reviewId: number) =>
    http
      .post<ApiResponse<null>>(`/projects/${projectId}/reviews/${reviewId}/like`)
      .then(extractData),

  unlike: (projectId: number, reviewId: number) =>
    http
      .delete<ApiResponse<null>>(`/projects/${projectId}/reviews/${reviewId}/like`)
      .then(extractData),

  // 开发者回复评测
  reply: (projectId: number, reviewId: number, content: string) =>
    http
      .post<ApiResponse<null>>(`/projects/${projectId}/reviews/${reviewId}/reply`, { content })
      .then(extractData),
}

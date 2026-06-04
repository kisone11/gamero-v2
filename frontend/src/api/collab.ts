import http, { extractData } from '@/lib/http'
import type { ApiResponse, PageData, CollabReview, CollabReviewReq } from '@/types/api'

export const collabApi = {
  getReviewTags: () =>
    http.get<ApiResponse<{ tags: string[] }>>('/review-tags').then(extractData),

  createReview: (data: CollabReviewReq) =>
    http.post<ApiResponse<CollabReview>>('/collab-reviews', data).then(extractData),

  getUserReviews: (userId: number, page = 1, pageSize = 20) =>
    http
      .get<ApiResponse<PageData<CollabReview>>>(`/users/${userId}/reviews`, {
        params: { page, page_size: pageSize },
      })
      .then(extractData),
}

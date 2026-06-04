import http, { extractData } from '@/lib/http'
import type {
  ApiResponse,
  PageData,
  UserProfileResponse,
  FollowUserItem,
  UserSkill,
  Portfolio,
  MyStats,
  UpdateProfileReq,
  UpdateSkillsReq,
  CreatePortfolioReq,
  UpdatePortfolioReq,
  SetAvailabilityReq,
  Post,
  DevLog,
  CollabReview,
} from '@/types/api'

export const userApi = {
  getProfile: (username: string) =>
    http.get<ApiResponse<UserProfileResponse>>(`/users/${username}`).then(extractData),

  updateProfile: (data: UpdateProfileReq) =>
    http.patch<ApiResponse<UserProfileResponse>>('/users/me/profile', data).then(extractData),

  updateAvatar: (key: string) =>
    http.patch<ApiResponse<{ avatar_url: string }>>('/users/me/avatar', { key }).then(extractData),

  updateCover: (key: string) =>
    http.patch<ApiResponse<null>>('/users/me/cover', { key }).then(extractData),

  updateSkills: (data: UpdateSkillsReq) =>
    http.put<ApiResponse<UserSkill[]>>('/users/me/skills', data).then(extractData),

  createPortfolio: (data: CreatePortfolioReq) =>
    http.post<ApiResponse<Portfolio>>('/users/me/portfolio', data).then(extractData),

  updatePortfolio: (id: number, data: UpdatePortfolioReq) =>
    http.patch<ApiResponse<Portfolio>>(`/users/me/portfolio/${id}`, data).then(extractData),

  deletePortfolio: (id: number) =>
    http.delete<ApiResponse<null>>(`/users/me/portfolio/${id}`).then(extractData),

  setAvailability: (data: SetAvailabilityReq) =>
    http.patch<ApiResponse<null>>('/users/me/availability', data).then(extractData),

  follow: (userId: number) =>
    http.post<ApiResponse<null>>(`/users/${userId}/follow`).then(extractData),

  unfollow: (userId: number) =>
    http.delete<ApiResponse<null>>(`/users/${userId}/follow`).then(extractData),

  getFollowers: (username: string, page = 1, pageSize = 20) =>
    http
      .get<ApiResponse<PageData<FollowUserItem>>>(`/users/${username}/followers`, {
        params: { page, page_size: pageSize },
      })
      .then(extractData),

  getFollowing: (username: string, page = 1, pageSize = 20) =>
    http
      .get<ApiResponse<PageData<FollowUserItem>>>(`/users/${username}/following`, {
        params: { page, page_size: pageSize },
      })
      .then(extractData),

  getMyStats: () =>
    http.get<ApiResponse<MyStats>>('/users/me/stats').then(extractData),

  searchUsers: (keyword: string, page = 1, pageSize = 20) =>
    http
      .get<ApiResponse<PageData<UserProfileResponse>>>('/users/search', {
        params: { q: keyword, page, page_size: pageSize },
      })
      .then(extractData),

  // 我收藏的帖子
  getMyCollectedPosts: (page = 1, pageSize = 20) =>
    http
      .get<ApiResponse<PageData<Post>>>('/me/collections/posts', {
        params: { page, page_size: pageSize },
      })
      .then(extractData),

  getUserLogs: (userId: number, page = 1, pageSize = 10) =>
    http
      .get<ApiResponse<PageData<DevLog>>>(`/users/${userId}/logs`, {
        params: { page, page_size: pageSize },
      })
      .then(extractData),

  getUserEndorsements: (userId: number, page = 1, pageSize = 10) =>
    http
      .get<ApiResponse<PageData<CollabReview>>>(`/users/${userId}/endorsements`, {
        params: { page, page_size: pageSize },
      })
      .then(extractData),
}

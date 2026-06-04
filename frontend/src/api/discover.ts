import http, { extractData } from '@/lib/http'
import type { ApiResponse, PageData, ProjectListItem, DevLog, Post } from '@/types/api'

export interface DiscoverParams {
  page?: number; page_size?: number
  q?: string; keyword?: string
  type?: 'all' | 'project' | 'log' | 'post' | 'user'
}

export interface FeedItem {
  id: string
  type: 'post' | 'devlog' | 'project' | 'release'
  author: { id: number; nickname: string; username: string; avatar_url?: string }
  post?: import('@/types/api').Post
  devlog?: import('@/types/api').DevLog
  project?: import('@/types/api').ProjectListItem
  created_at: string
}

export const discoverApi = {
  // Feed uses cursor-based pagination
  getFeed: (cursor?: number, limit = 20) =>
    http.get<ApiResponse<{ items: FeedItem[]; next_cursor: number }>>('/feed', {
      params: { cursor, limit }
    }).then(extractData),

  getRecommendedProjects: (page = 1, pageSize = 5) =>
    http.get<ApiResponse<PageData<ProjectListItem>>>('/discover/projects', { params: { page, page_size: pageSize } }).then(extractData),

  getTrendingLogs: (page = 1, pageSize = 12) =>
    http.get<ApiResponse<PageData<DevLog>>>('/discover/logs', { params: { page, page_size: pageSize } }).then(extractData),

  getTrendingPosts: (page = 1, pageSize = 12) =>
    http.get<ApiResponse<PageData<Post>>>('/discover/posts', { params: { page, page_size: pageSize } }).then(extractData),

  getRecommendedPosts: (page = 1, pageSize = 12) =>
    http.get<ApiResponse<PageData<Post>>>('/discover/recommended-posts', { params: { page, page_size: pageSize } }).then(extractData),

  search: (params: DiscoverParams) =>
    http.get<ApiResponse<PageData<ProjectListItem | DevLog | Post>>>('/search', { params }).then(extractData),
}

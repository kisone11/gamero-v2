import http, { extractData } from '@/lib/http'
import type {
  ApiResponse,
  PageData,
  PlatformStats,
  DailyStats,
  AdminDashboardOverview,
  Report,
  AuditLog,
  UserProfileResponse,
  ProjectListItem,
  Post,
  DevLog,
  Topic,
} from '@/types/api'

// 敏感词相关类型
export interface SensitiveWord {
  id: number
  word: string
  created_at: string
}

export interface SensitiveWordsResponse {
  words: string[]
  total: number
}

export interface UserDetail extends UserProfileResponse {
  email?: string
  is_banned: boolean
  ban_reason?: string
  last_login?: string
  recent_projects: ProjectListItem[]
  recent_posts: Post[]
  recent_logs: DevLog[]
}

export const adminApi = {
  // ===== 统计数据 =====
  getPlatformStats: () =>
    http.get<ApiResponse<PlatformStats>>('/admin/stats').then(extractData),

  getDashboardOverview: () =>
    http.get<ApiResponse<AdminDashboardOverview>>('/admin/dashboard').then(extractData),

  getDailyStats: (days = 30) =>
    http.get<ApiResponse<DailyStats[]>>('/admin/stats/daily', { params: { days } }).then(extractData),

  // ===== 用户管理 =====
  listUsers: (params?: {
    page?: number
    page_size?: number
    keyword?: string
    role?: string
    status?: 'all' | 'normal' | 'banned'
  }) =>
    http.get<ApiResponse<PageData<UserProfileResponse>>>('/admin/users', { params }).then(extractData),

  getUser: (userId: number) =>
    http.get<ApiResponse<UserDetail>>(`/admin/users/${userId}`).then(extractData),

  banUser: (userId: number, reason: string) =>
    http.post<ApiResponse<null>>(`/admin/users/${userId}/ban`, { banned: true, reason }).then(extractData),

  unbanUser: (userId: number) =>
    http.delete<ApiResponse<null>>(`/admin/users/${userId}/ban`).then(extractData),

  setUserRole: (userId: number, role: 'user' | 'moderator' | 'admin') =>
    http.patch<ApiResponse<null>>(`/admin/users/${userId}/role`, { role }).then(extractData),

  batchBanUsers: (userIds: number[], reason: string) =>
    http.post<ApiResponse<null>>('/admin/users/batch-ban', { user_ids: userIds, reason }).then(extractData),

  exportUsers: (params?: { keyword?: string; role?: string; status?: string }) =>
    http.get('/admin/users/export', { params, responseType: 'blob' }),

  // ===== 内容审核 =====
  listReports: (params?: {
    page?: number
    page_size?: number
    status?: 'pending' | 'handled' | 'rejected'
    target_type?: 'post' | 'log' | 'project' | 'comment' | 'user' | 'all'
  }) =>
    http.get<ApiResponse<PageData<Report>>>('/admin/reports', { params }).then(extractData),

  handleReport: (id: number, action: string, note?: string) => {
    const status = action === 'dismiss' ? 'rejected' : 'handled'
    return http.patch<ApiResponse<null>>(`/admin/reports/${id}`, { status, note }).then(extractData)
  },

  hidePost: (id: number) =>
    http.patch<ApiResponse<null>>(`/admin/posts/${id}/hide`).then(extractData),

  hideLog: (id: number) =>
    http.patch<ApiResponse<null>>(`/admin/logs/${id}/hide`).then(extractData),

  deletePost: (id: number) =>
    http.delete<ApiResponse<null>>(`/admin/posts/${id}`).then(extractData),

  closeRecruitment: (id: number) =>
    http.delete<ApiResponse<null>>(`/admin/recruitments/${id}`).then(extractData),

  deleteComment: (id: number) =>
    http.delete<ApiResponse<null>>(`/admin/comments/${id}`).then(extractData),

  banProject: (id: number, reason = '') =>
    http.put<ApiResponse<null>>(`/admin/projects/${id}/ban`, { reason }).then(extractData),

  deleteProject: (id: number) =>
    http.delete<ApiResponse<null>>(`/admin/projects/${id}`).then(extractData),

  listProjects: (params?: { page?: number; page_size?: number; keyword?: string }) =>
    http.get<ApiResponse<PageData<ProjectListItem>>>('/admin/projects', { params }).then(extractData),

  listPosts: (params?: { page?: number; page_size?: number; keyword?: string }) =>
    http.get<ApiResponse<PageData<Post>>>('/admin/posts', { params }).then(extractData),

  listLogs: (params?: { page?: number; page_size?: number; keyword?: string }) =>
    http.get<ApiResponse<PageData<DevLog>>>('/admin/logs', { params }).then(extractData),

  // ===== 话题管理 =====
  createTopic: (data: { name: string; description?: string; icon?: string }) =>
    http.post<ApiResponse<Topic>>('/admin/topics', data).then(extractData),

  updateTopic: (id: number, data: { name?: string; description?: string; icon?: string; sort_order?: number }) =>
    http.put<ApiResponse<Topic>>(`/admin/topics/${id}`, data).then(extractData),

  deleteTopic: (id: number) =>
    http.delete<ApiResponse<null>>(`/admin/topics/${id}`).then(extractData),

  saveTopicIcon: (id: number, icon: File) => {
    const formData = new FormData()
    formData.append('icon', icon)
    return http.put<ApiResponse<{ url: string }>>(`/admin/topics/${id}/icon`, formData).then(extractData)
  },

  // ===== 话题管理 - 列表 =====
  listTopics: () =>
    http.get<ApiResponse<Topic[]>>('/admin/topics').then(extractData),

  // ===== 敏感词管理 =====
  getSensitiveWords: () =>
    http.get<ApiResponse<SensitiveWordsResponse>>('/admin/sensitive-words').then(extractData),

  addSensitiveWords: (words: string[]) =>
    http.post<ApiResponse<null>>('/admin/sensitive-words', { words }).then(extractData),

  deleteSensitiveWord: (word: string) =>
    http.delete<ApiResponse<null>>(`/admin/sensitive-words/${encodeURIComponent(word)}`).then(extractData),

  // ===== 审计日志 =====
  getAuditLogs: (params?: {
    page?: number
    page_size?: number
    admin_id?: number
    action?: string
    start_time?: string
    end_time?: string
  }) =>
    http.get<ApiResponse<PageData<AuditLog>>>('/admin/audit-logs', { params }).then(extractData),
}

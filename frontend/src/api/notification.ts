import http, { extractData } from '@/lib/http'
import type {
  ApiResponse,
  PageData,
  Notification,
  NotificationCategory,
} from '@/types/api'

export const notificationApi = {
  list: (page = 1, pageSize = 20) =>
    http
      .get<ApiResponse<PageData<Notification>>>('/notifications', {
        params: { page, page_size: pageSize },
      })
      .then(extractData),

  getUnreadCount: () =>
    http.get<ApiResponse<{ unread_count: number }>>('/notifications/unread-count').then(extractData),

  getCategories: () =>
    http.get<ApiResponse<NotificationCategory[]>>('/notifications/categories').then(extractData),

  markRead: (id: number) =>
    http.patch<ApiResponse<null>>(`/notifications/${id}/read`).then(extractData),

  markAllRead: () =>
    http.patch<ApiResponse<null>>('/notifications/read-all').then(extractData),

  delete: (id: number) =>
    http.delete<ApiResponse<null>>(`/notifications/${id}`).then(extractData),

  // 删除所有已读通知
  deleteRead: () =>
    http.delete<ApiResponse<null>>('/notifications/read').then(extractData),

  getByType: (type: string, page = 1, pageSize = 20) =>
    http
      .get<ApiResponse<PageData<Notification>>>('/notifications/by-type', {
        params: { type, page, page_size: pageSize },
      })
      .then(extractData),

  getPreferences: () =>
    http
      .get<ApiResponse<{ preferences: { type: string; enabled: boolean }[] }>>('/notifications/preferences')
      .then(extractData),

  updatePreference: (type: string, enabled: boolean) =>
    http
      .patch<ApiResponse<null>>('/notifications/preferences', { type, enabled })
      .then(extractData),
}

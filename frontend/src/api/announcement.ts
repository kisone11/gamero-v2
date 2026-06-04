import http, { extractData } from '@/lib/http'
import type { Announcement, ApiResponse, PageData } from '@/types/api'

export interface AnnouncementReq {
  title: string
  content: string
  level?: 'info' | 'warning' | 'danger' | 'success'
  is_active?: boolean
  is_pinned?: boolean
  expire_days?: number
}

export const announcementApi = {
  listPublic: (limit = 5) =>
    http.get<ApiResponse<Announcement[]>>('/announcements', { params: { limit } }).then(extractData),

  adminList: (page = 1, pageSize = 20, active?: boolean) =>
    http.get<ApiResponse<PageData<Announcement>>>('/admin/announcements', { params: { page, page_size: pageSize, active } }).then(extractData),

  adminCreate: (data: AnnouncementReq) =>
    http.post<ApiResponse<Announcement>>('/admin/announcements', data).then(extractData),

  adminUpdate: (id: number, data: AnnouncementReq) =>
    http.put<ApiResponse<null>>(`/admin/announcements/${id}`, data).then(extractData),

  adminDelete: (id: number) =>
    http.delete<ApiResponse<null>>(`/admin/announcements/${id}`).then(extractData),
}

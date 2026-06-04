import http, { extractData } from '@/lib/http'
import type {
  ApiResponse, PageData,
  RecruitmentListItem, ApplicationDetail,
  CreateRecruitmentReq, UpdateRecruitmentReq, ApplyRecruitmentReq,
} from '@/types/api'
import type { RecruitmentPosition } from '@/types/enums'

export interface ListRecruitmentsParams {
  page?: number; page_size?: number; position?: RecruitmentPosition; keyword?: string; project_id?: number
}

export const recruitApi = {
  list: (params?: ListRecruitmentsParams) =>
    http.get<ApiResponse<PageData<RecruitmentListItem>>>('/recruitments', { params }).then(extractData),

  get: (id: number) =>
    http.get<ApiResponse<RecruitmentListItem>>(`/recruitments/${id}`).then(extractData),

  create: (data: CreateRecruitmentReq) =>
    http.post<ApiResponse<RecruitmentListItem>>('/recruitments', data).then(extractData),

  update: (id: number, data: UpdateRecruitmentReq) =>
    http.put<ApiResponse<RecruitmentListItem>>(`/recruitments/${id}`, data).then(extractData),

  close: (id: number) =>
    http.patch<ApiResponse<null>>(`/recruitments/${id}/close`).then(extractData),

  reopen: (id: number) =>
    http.post<ApiResponse<null>>(`/recruitments/${id}/reopen`).then(extractData),

  delete: (id: number) =>
    http.delete<ApiResponse<null>>(`/recruitments/${id}`).then(extractData),

  getApplications: (id: number) =>
    http.get<ApiResponse<PageData<ApplicationDetail>>>(`/recruitments/${id}/applications`).then(extractData),

  apply: (recruitmentId: number, data: ApplyRecruitmentReq) =>
    http.post<ApiResponse<ApplicationDetail>>(`/recruitments/${recruitmentId}/apply`, data).then(extractData),

  approveApplication: (appId: number) =>
    http.patch<ApiResponse<null>>(`/applications/${appId}/approve`).then(extractData),

  rejectApplication: (appId: number) =>
    http.patch<ApiResponse<null>>(`/applications/${appId}/reject`).then(extractData),

  withdrawApplication: (appId: number) =>
    http.patch<ApiResponse<null>>(`/applications/${appId}/withdraw`).then(extractData),

  listMyApplications: (page = 1, pageSize = 20) =>
    http.get<ApiResponse<PageData<ApplicationDetail>>>('/applications/mine', { params: { page, page_size: pageSize } }).then(extractData),
}

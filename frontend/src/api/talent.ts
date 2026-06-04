import http, { extractData } from '@/lib/http'
import type { ApiResponse, PageData, TalentListItem, InvitationDetail, InviteTalentReq } from '@/types/api'
import type { SkillCategory } from '@/types/enums'

export interface ListTalentsParams {
  page?: number; page_size?: number; category?: SkillCategory; only_available?: boolean
  keyword?: string; level?: string; coop_preference?: string
}

export const talentApi = {
  list: (params?: ListTalentsParams) =>
    http.get<ApiResponse<PageData<TalentListItem>>>('/talents', { params }).then(extractData),

  invite: (data: InviteTalentReq) =>
    http.post<ApiResponse<InvitationDetail>>('/invitations', data).then(extractData),

  acceptInvitation: (id: number) =>
    http.patch<ApiResponse<null>>(`/invitations/${id}/accept`).then(extractData),

  declineInvitation: (id: number) =>
    http.patch<ApiResponse<null>>(`/invitations/${id}/decline`).then(extractData),

  listMyInvitations: (page = 1, pageSize = 20) =>
    http.get<ApiResponse<PageData<InvitationDetail>>>('/invitations/mine', { params: { page, page_size: pageSize } }).then(extractData),
}

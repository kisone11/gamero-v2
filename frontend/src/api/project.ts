import http, { extractData } from '@/lib/http'
import type {
  ApiResponse,
  PageData,
  ProjectDetail,
  ProjectMilestone,
  ProjectMilestoneReq,
  ProjectResource,
  ProjectResourceReq,
  ProjectRisk,
  ProjectRiskReq,
  ProjectTask,
  ProjectTaskReq,
  ProjectListItem,
  TimelineEvent,
  ProjectStats,
  MemberDetail,
  CreateProjectReq,
  UpdateProjectReq,
  UpdateStatusReq,
  AddMemberReq,
  UpdateMemberReq,
} from '@/types/api'
import type { ProjectGenre, ProjectStatus } from '@/types/enums'

export interface ListProjectsParams {
  page?: number
  page_size?: number
  keyword?: string
  genre?: ProjectGenre
  status?: ProjectStatus
  owner_id?: number
  participant_id?: number
  sort?: 'latest' | 'popular' | 'following'
}

export const projectApi = {
  list: (params?: ListProjectsParams) =>
    http.get<ApiResponse<PageData<ProjectListItem>>>('/projects', { params }).then(extractData),

  getUserProjects: (userId: number, page = 1, pageSize = 9) =>
    http.get<ApiResponse<PageData<ProjectListItem>>>(`/users/${userId}/projects`, { params: { page, page_size: pageSize } }).then(extractData),

  get: (slug: string) => {
    if (/^\d+$/.test(slug)) {
      return http.get<ApiResponse<ProjectDetail>>(`/projects/${slug}`).then(extractData)
    }
    return http.get<ApiResponse<ProjectDetail>>(`/projects/slug/${slug}`).then(extractData)
  },

  create: (data: CreateProjectReq) =>
    http.post<ApiResponse<ProjectDetail>>('/projects', data).then(extractData),

  update: (id: number | string, data: UpdateProjectReq) =>
    http.patch<ApiResponse<ProjectDetail>>(`/projects/${id}`, data).then(extractData),

  delete: (id: number | string) =>
    http.delete<ApiResponse<null>>(`/projects/${id}`).then(extractData),

  updateStatus: (id: number | string, data: UpdateStatusReq) =>
    http.patch<ApiResponse<null>>(`/projects/${id}/status`, data).then(extractData),

  uploadCover: (id: number | string, key: string) =>
    http.patch<ApiResponse<{ cover_url: string }>>(`/projects/${id}/cover`, { key }).then(extractData),

  uploadScreenshots: (id: number | string, keys: string[]) =>
    Promise.all(
      keys.map((key) =>
        http.post<ApiResponse<{ url: string }>>(`/projects/${id}/screenshots`, { key }).then(extractData),
      ),
    ),

  deleteScreenshot: (id: number | string, key: string) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/screenshots`, { data: { key } }).then(extractData),

  uploadVideo: (id: number | string, key: string) =>
    http.post<ApiResponse<{ url: string; key: string }>>(`/projects/${id}/videos`, { key }).then(extractData),

  deleteVideo: (id: number | string, key: string) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/videos`, { data: { key } }).then(extractData),

  follow: (id: number | string) =>
    http.post<ApiResponse<null>>(`/projects/${id}/follow`).then(extractData),

  unfollow: (id: number | string) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/follow`).then(extractData),

  collect: (id: number | string) =>
    http.post<ApiResponse<null>>(`/projects/${id}/collect`).then(extractData),

  uncollect: (id: number | string) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/collect`).then(extractData),

  getTimeline: (id: number | string) =>
    http.get<ApiResponse<TimelineEvent[]>>(`/projects/${id}/timeline`).then(extractData),

  getStats: (id: number | string) =>
    http.get<ApiResponse<ProjectStats>>(`/projects/${id}/stats`).then(extractData),

  // 成员
  addMember: (id: number | string, data: AddMemberReq) =>
    http.post<ApiResponse<MemberDetail>>(`/projects/${id}/members`, data).then(extractData),

  updateMember: (id: number | string, userId: number, data: UpdateMemberReq) =>
    http.patch<ApiResponse<MemberDetail>>(`/projects/${id}/members/${userId}`, data).then(extractData),

  removeMember: (id: number | string, userId: number) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/members/${userId}`).then(extractData),

  listTasks: (id: number | string) =>
    http.get<ApiResponse<ProjectTask[]>>(`/projects/${id}/tasks`).then(extractData),

  createTask: (id: number | string, data: ProjectTaskReq) =>
    http.post<ApiResponse<ProjectTask>>(`/projects/${id}/tasks`, data).then(extractData),

  updateTask: (id: number | string, taskId: number, data: ProjectTaskReq) =>
    http.patch<ApiResponse<ProjectTask>>(`/projects/${id}/tasks/${taskId}`, data).then(extractData),

  deleteTask: (id: number | string, taskId: number) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/tasks/${taskId}`).then(extractData),

  listMilestones: (id: number | string) =>
    http.get<ApiResponse<ProjectMilestone[]>>(`/projects/${id}/milestones`).then(extractData),

  createMilestone: (id: number | string, data: ProjectMilestoneReq) =>
    http.post<ApiResponse<ProjectMilestone>>(`/projects/${id}/milestones`, data).then(extractData),

  updateMilestone: (id: number | string, milestoneId: number, data: ProjectMilestoneReq) =>
    http.patch<ApiResponse<ProjectMilestone>>(`/projects/${id}/milestones/${milestoneId}`, data).then(extractData),

  deleteMilestone: (id: number | string, milestoneId: number) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/milestones/${milestoneId}`).then(extractData),

  listResources: (id: number | string) =>
    http.get<ApiResponse<ProjectResource[]>>(`/projects/${id}/resources`).then(extractData),

  createResource: (id: number | string, data: ProjectResourceReq) =>
    http.post<ApiResponse<ProjectResource>>(`/projects/${id}/resources`, data).then(extractData),

  updateResource: (id: number | string, resourceId: number, data: ProjectResourceReq) =>
    http.patch<ApiResponse<ProjectResource>>(`/projects/${id}/resources/${resourceId}`, data).then(extractData),

  deleteResource: (id: number | string, resourceId: number) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/resources/${resourceId}`).then(extractData),

  listRisks: (id: number | string) =>
    http.get<ApiResponse<ProjectRisk[]>>(`/projects/${id}/risks`).then(extractData),

  createRisk: (id: number | string, data: ProjectRiskReq) =>
    http.post<ApiResponse<ProjectRisk>>(`/projects/${id}/risks`, data).then(extractData),

  updateRisk: (id: number | string, riskId: number, data: ProjectRiskReq) =>
    http.patch<ApiResponse<ProjectRisk>>(`/projects/${id}/risks/${riskId}`, data).then(extractData),

  deleteRisk: (id: number | string, riskId: number) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/risks/${riskId}`).then(extractData),

  leaveProject: (id: number | string) =>
    http.delete<ApiResponse<null>>(`/projects/${id}/members/me`).then(extractData),

}

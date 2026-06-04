import { http } from '@kit.NetworkKit';
import { fileIo } from '@kit.CoreFileKit';
import { photoAccessHelper } from '@kit.MediaLibraryKit';
import { ApiConfig } from './ApiConfig';

interface ApiResponse<T> {
  code: number;
  message: string;
  data?: T;
}

interface RequestHeaders {
  'Content-Type': string;
  Authorization?: string;
}

interface UploadHeaders {
  'Content-Type': string;
}

export interface AuthUser {
  id: number;
  username: string;
  nickname: string;
  email?: string;
  role?: string;
}

export interface UserSkillItem {
  id: number;
  user_id: number;
  category: string;
  name: string;
  level: string;
  description?: string;
  is_custom?: boolean;
  created_at?: string;
}

export interface PortfolioItem {
  id: number;
  user_id: number;
  name: string;
  type: string;
  link?: string;
  image_url?: string;
  description?: string;
  created_at?: string;
}

export interface TalentItem {
  id: number;
  username: string;
  nickname: string;
  bio?: string;
  skills: UserSkillItem[];
  project_count: number;
  is_available: boolean;
  coop_preference?: string;
  location?: string;
}

export interface UserProfileItem {
  id: number;
  username: string;
  nickname: string;
  email?: string;
  role?: string;
  bio?: string;
  skills: UserSkillItem[];
  portfolio?: PortfolioItem[];
  post_count: number;
  log_count: number;
  project_count: number;
  followers_count: number;
  following_count: number;
  is_following: boolean;
  is_available?: boolean;
  coop_preference?: string;
  location?: string;
  created_at: string;
}

export interface ReportItem {
  id: number;
  reporter_id: number;
  target_type: string;
  target_id: number;
  reason: string;
  status: string;
  note?: string;
  supplement?: string;
  handler_name?: string;
  created_at: string;
  updated_at?: string;
}

export interface SensitiveWordsData {
  words: string[];
  total: number;
}

export interface UploadTokenData {
  upload_url: string;
  key: string;
  expires_at: number;
}

interface UploadTokenReq {
  type: string;
  filename: string;
  resource_id: number;
  content_type: string;
}

interface MediaKeyReq {
  key: string;
}

interface SensitiveWordsReq {
  words: string[];
}

interface BanUserReq {
  banned: boolean;
  reason: string;
}

interface HandleReportReq {
  status: string;
  note: string;
}

interface BanProjectReq {
  reason: string;
}

interface UpdateProfileReq {
  nickname: string;
  bio: string;
  location: string;
}

interface AvailabilityReq {
  is_available: boolean;
  coop_preference: string;
}

interface SkillInputItem {
  category: string;
  name: string;
  level: string;
  description: string;
}

interface UpdateSkillsReq {
  skills: SkillInputItem[];
}

interface PortfolioReq {
  name: string;
  type: string;
  link: string;
  image_url: string;
  description: string;
}

export interface SkillsData {
  skills: UserSkillItem[];
}

export interface PortfolioData {
  portfolio: PortfolioItem[];
}

interface AddMemberReq {
  user_id: number;
  role: string;
  contribution: string;
}

interface UpdateMemberReq {
  role: string;
  contribution: string;
}

export interface AuthResponse {
  user: AuthUser;
  access_token: string;
  refresh_token: string;
}

export interface PageData<T> {
  list: T[];
  total: number;
  page: number;
  page_size: number;
  pages: number;
}

export interface ProjectItem {
  id: number;
  owner_id?: number;
  name: string;
  slug: string;
  description: string;
  genre: string;
  status: string;
  follower_count: number;
  like_count?: number;
  collect_count?: number;
  view_count?: number;
  member_count?: number;
  cover_url?: string;
}

export interface ProjectMemberItem {
  id?: number;
  user_id: number;
  role: string;
  nickname?: string;
  username?: string;
  joined_at?: string;
}

export interface ProjectDetailItem extends ProjectItem {
  style_tags?: string[];
  engine?: string;
  platform?: string[];
  visibility?: string;
  demo_url?: string;
  store_url?: string;
  video_url?: string;
  screenshot_urls?: string[];
  is_collected?: boolean;
  is_followed?: boolean;
  members?: ProjectMemberItem[];
  created_at?: string;
  updated_at?: string;
}

export interface RecruitItem {
  id: number;
  project_id: number;
  owner_id?: number;
  project_name: string;
  project_slug?: string;
  position: string;
  description?: string;
  cooperation_type: string;
  contact_type?: string;
  contact_info?: string;
  headcount: number;
  expire_at: string;
  status?: string;
  created_at?: string;
}

export interface ApplicationItem {
  id: number;
  recruitment_id: number;
  project_id?: number;
  project_name?: string;
  position: string;
  message?: string;
  status: string;
  created_at: string;
  applicant_id: number;
  applicant_username: string;
  applicant_nickname: string;
}

export interface FeedItem {
  id: string;
  type: string;
  created_at?: string;
  author?: AuthUser;
  post?: FeedPostDetail;
  devlog?: FeedDevLogDetail;
  project?: FeedProjectDetail;
}

export interface FeedPostDetail {
  id: number;
  title: string;
  content_snippet: string;
  like_count: number;
  comment_count: number;
  created_at?: string;
}

export interface FeedDevLogDetail {
  id: number;
  project_id: number;
  title: string;
  log_type: string;
  created_at?: string;
}

export interface FeedProjectDetail {
  id: number;
  name: string;
  description: string;
  status: string;
  created_at?: string;
}

export interface FeedResponse {
  items: FeedItem[];
  next_cursor: number;
}

export interface PlatformStats {
  total_users: number;
  active_users: number;
  total_projects: number;
  total_logs: number;
  total_posts: number;
  new_users_today: number;
  new_projects_today: number;
}

export interface ProjectTask {
  id: number;
  project_id: number;
  title: string;
  description?: string;
  status: string;
  priority: string;
  due_date?: string;
  assignee_nickname?: string;
}

interface ProjectTaskReq {
  title: string;
  description?: string;
  status: string;
  priority: string;
  due_date?: string;
}

interface ApplyRecruitmentReq {
  position: string;
  message: string;
}

interface RecruitmentReq {
  project_id?: number;
  position: string;
  headcount: number;
  description: string;
  cooperation_type: string;
  expire_days?: number;
}

export interface PostItem {
  id: number;
  title: string;
  content: string;
  content_snippet?: string;
  like_count: number;
  comment_count: number;
  view_count: number;
  created_at: string;
  author?: AuthUser;
}

export interface DevLogItem {
  id: number;
  project_id: number;
  title: string;
  content: string;
  log_type: string;
  like_count: number;
  comment_count: number;
  view_count: number;
  created_at: string;
  author_nickname?: string;
  project_name?: string;
}

export interface NotificationItem {
  id: number;
  type: string;
  title: string;
  content: string;
  is_read: boolean;
  created_at: string;
}

export interface NotificationPreferenceItem {
  type: string;
  enabled: boolean;
}

export interface NotificationPreferencesData {
  preferences: NotificationPreferenceItem[];
}

interface NotificationPreferenceReq {
  type: string;
  enabled: boolean;
}

export interface UnreadCountResponse {
  unread_count: number;
}

export interface SearchItem {
  id: number;
  title?: string;
  name?: string;
  content?: string;
  description?: string;
  created_at?: string;
  author?: AuthUser;
}

export interface CommentItem {
  id: number;
  content: string;
  like_count: number;
  is_liked?: boolean;
  created_at: string;
  author?: AuthUser;
  user_nickname?: string;
  user_username?: string;
}

interface CreateCommentReq {
  content: string;
}

interface CreatePostReq {
  title: string;
  content: string;
  images?: string[];
  videos?: string[];
}

interface CreateDevLogReq {
  title: string;
  content: string;
  log_type: string;
  version?: string;
  visibility: string;
  status: string;
}

interface CreateProjectReq {
  name: string;
  description: string;
  genre: string;
  status: string;
  visibility: string;
}

interface LoginCredentials {
  email: string;
  password: string;
}

interface RegisterCredentials {
  email: string;
  code: string;
  password: string;
  nickname: string;
}

interface SendEmailCodeReq {
  email: string;
  scene: string;
}

export class GameroApi {
  static accessToken: string = '';

  static async get<T>(path: string, token: string = GameroApi.accessToken): Promise<T> {
    return GameroApi.request<T>(http.RequestMethod.GET, path, undefined, token);
  }

  static async post<T>(path: string, body?: object, token: string = GameroApi.accessToken): Promise<T> {
    return GameroApi.request<T>(http.RequestMethod.POST, path, body, token);
  }

  static async patch<T>(path: string, body?: object, token: string = GameroApi.accessToken): Promise<T> {
    return GameroApi.request<T>('PATCH' as http.RequestMethod, path, body, token);
  }

  static async put<T>(path: string, body?: object, token: string = GameroApi.accessToken): Promise<T> {
    return GameroApi.request<T>('PUT' as http.RequestMethod, path, body, token);
  }

  static async delete<T>(path: string, token: string = GameroApi.accessToken): Promise<T> {
    return GameroApi.request<T>(http.RequestMethod.DELETE, path, undefined, token);
  }

  static async request<T>(method: http.RequestMethod, path: string, body?: object, token: string = ''): Promise<T> {
    const client = http.createHttp();
    const headers: RequestHeaders = {
      'Content-Type': 'application/json'
    };
    if (token.length > 0) {
      headers.Authorization = `Bearer ${token}`;
    }

    let payload: string | undefined = undefined;
    if (body) {
      try {
        payload = JSON.stringify(body);
      } catch {
        throw new Error('请求参数序列化失败');
      }
    }

    const response = await client.request(`${ApiConfig.baseUrl}${path}`, {
      method,
      header: headers,
      extraData: payload,
      expectDataType: http.HttpDataType.STRING,
      connectTimeout: 10000,
      readTimeout: 15000
    });
    client.destroy();

    const raw = response.result as string;
    let parsed: ApiResponse<T>;
    try {
      parsed = JSON.parse(raw) as ApiResponse<T>;
    } catch {
      throw new Error('接口响应解析失败');
    }
    if (parsed.code !== 0) {
      throw new Error(parsed.message || '请求失败');
    }
    return parsed.data as T;
  }

  static getUploadToken(type: string, filename: string, resourceId: number, contentType: string): Promise<UploadTokenData> {
    const payload: UploadTokenReq = { type, filename, resource_id: resourceId, content_type: contentType };
    return GameroApi.post<UploadTokenData>('/upload/token', payload);
  }

  static async pickAndUploadMedia(type: string, resourceId: number, filename: string, contentType: string, isVideo: boolean): Promise<string> {
    const picker = new photoAccessHelper.PhotoViewPicker();
    const options = new photoAccessHelper.PhotoSelectOptions();
    options.MIMEType = isVideo ? photoAccessHelper.PhotoViewMIMETypes.VIDEO_TYPE : photoAccessHelper.PhotoViewMIMETypes.IMAGE_TYPE;
    options.maxSelectNumber = 1;
    const result = await picker.select(options);
    if (result.photoUris.length === 0) {
      throw new Error('未选择文件');
    }

    const token = await GameroApi.getUploadToken(type, filename, resourceId, contentType);
    const file = fileIo.openSync(result.photoUris[0], fileIo.OpenMode.READ_ONLY);
    const stat = fileIo.statSync(file.fd);
    const buffer = new ArrayBuffer(stat.size);
    fileIo.readSync(file.fd, buffer);
    fileIo.closeSync(file);

    const client = http.createHttp();
    const uploadHeaders: UploadHeaders = { 'Content-Type': contentType };
    try {
      const uploadResult = await client.request(token.upload_url, {
        method: http.RequestMethod.PUT,
        header: uploadHeaders,
        extraData: buffer
      });
      if (uploadResult.responseCode < 200 || uploadResult.responseCode >= 300) {
        throw new Error('对象存储上传失败');
      }
      return token.key;
    } finally {
      client.destroy();
    }
  }

  static async login(email: string, password: string): Promise<AuthResponse> {
    const credentials: LoginCredentials = { email, password };
    const result = await GameroApi.post<AuthResponse>('/auth/login/email', credentials, '');
    GameroApi.accessToken = result.access_token;
    return result;
  }

  static async sendRegisterCode(email: string): Promise<void> {
    const payload: SendEmailCodeReq = { email, scene: 'register' };
    await GameroApi.post<void>('/auth/email/code', payload, '');
  }

  static async register(email: string, code: string, password: string, nickname: string): Promise<AuthResponse> {
    const credentials: RegisterCredentials = { email, code, password, nickname };
    const result = await GameroApi.post<AuthResponse>('/auth/register/email', credentials, '');
    GameroApi.accessToken = result.access_token;
    return result;
  }

  static logout(): void {
    GameroApi.accessToken = '';
  }

  static listProjects(): Promise<PageData<ProjectItem>> {
    return GameroApi.get<PageData<ProjectItem>>('/projects?page=1&page_size=10&sort=latest', '');
  }

  static createProject(name: string, description: string, genre: string, status: string): Promise<ProjectDetailItem> {
    const payload: CreateProjectReq = {
      name,
      description,
      genre,
      status,
      visibility: 'public'
    };
    return GameroApi.post<ProjectDetailItem>('/projects', payload);
  }

  static updateProject(id: number, name: string, description: string, genre: string, status: string): Promise<ProjectDetailItem> {
    const payload: CreateProjectReq = {
      name,
      description,
      genre,
      status,
      visibility: 'public'
    };
    return GameroApi.patch<ProjectDetailItem>(`/projects/${id}`, payload);
  }

  static deleteProject(id: number): Promise<void> {
    return GameroApi.delete<void>(`/projects/${id}`);
  }

  static saveProjectCover(id: number, key: string): Promise<void> {
    const payload: MediaKeyReq = { key };
    return GameroApi.patch<void>(`/projects/${id}/cover`, payload);
  }

  static saveProjectScreenshot(id: number, key: string): Promise<void> {
    const payload: MediaKeyReq = { key };
    return GameroApi.post<void>(`/projects/${id}/screenshots`, payload);
  }

  static saveProjectVideo(id: number, key: string): Promise<void> {
    const payload: MediaKeyReq = { key };
    return GameroApi.post<void>(`/projects/${id}/videos`, payload);
  }

  static saveLogImage(id: number, key: string): Promise<void> {
    const payload: MediaKeyReq = { key };
    return GameroApi.post<void>(`/logs/${id}/images`, payload);
  }

  static saveLogVideo(id: number, key: string): Promise<void> {
    const payload: MediaKeyReq = { key };
    return GameroApi.put<void>(`/logs/${id}/videos`, payload);
  }

  static getProject(id: number): Promise<ProjectDetailItem> {
    return GameroApi.get<ProjectDetailItem>(`/projects/${id}`);
  }

  static followProject(id: number): Promise<void> {
    return GameroApi.post<void>(`/projects/${id}/follow`);
  }

  static unfollowProject(id: number): Promise<void> {
    return GameroApi.delete<void>(`/projects/${id}/follow`);
  }

  static collectProject(id: number): Promise<void> {
    return GameroApi.post<void>(`/projects/${id}/collect`);
  }

  static uncollectProject(id: number): Promise<void> {
    return GameroApi.delete<void>(`/projects/${id}/collect`);
  }

  static addProjectMember(projectId: number, userId: number, role: string, contribution: string): Promise<ProjectMemberItem> {
    const payload: AddMemberReq = { user_id: userId, role, contribution };
    return GameroApi.post<ProjectMemberItem>(`/projects/${projectId}/members`, payload);
  }

  static updateProjectMember(projectId: number, userId: number, role: string, contribution: string): Promise<ProjectMemberItem> {
    const payload: UpdateMemberReq = { role, contribution };
    return GameroApi.patch<ProjectMemberItem>(`/projects/${projectId}/members/${userId}`, payload);
  }

  static removeProjectMember(projectId: number, userId: number): Promise<void> {
    return GameroApi.delete<void>(`/projects/${projectId}/members/${userId}`);
  }

  static leaveProject(projectId: number): Promise<void> {
    return GameroApi.delete<void>(`/projects/${projectId}/members/me`);
  }

  static listRecruitments(): Promise<PageData<RecruitItem>> {
    return GameroApi.get<PageData<RecruitItem>>('/recruitments?page=1&page_size=10', '');
  }

  static createRecruitment(projectId: number, position: string, headcount: number, description: string, cooperationType: string): Promise<RecruitItem> {
    const payload: RecruitmentReq = {
      project_id: projectId,
      position,
      headcount,
      description,
      cooperation_type: cooperationType,
      expire_days: 30
    };
    return GameroApi.post<RecruitItem>('/recruitments', payload);
  }

  static updateRecruitment(id: number, position: string, headcount: number, description: string, cooperationType: string): Promise<RecruitItem> {
    const payload: RecruitmentReq = {
      position,
      headcount,
      description,
      cooperation_type: cooperationType
    };
    return GameroApi.request<RecruitItem>('PUT' as http.RequestMethod, `/recruitments/${id}`, payload);
  }

  static closeRecruitment(id: number): Promise<void> {
    return GameroApi.patch<void>(`/recruitments/${id}/close`);
  }

  static reopenRecruitment(id: number): Promise<void> {
    return GameroApi.post<void>(`/recruitments/${id}/reopen`);
  }

  static deleteRecruitment(id: number): Promise<void> {
    return GameroApi.delete<void>(`/recruitments/${id}`);
  }

  static getRecruitment(id: number): Promise<RecruitItem> {
    return GameroApi.get<RecruitItem>(`/recruitments/${id}`);
  }

  static applyRecruitment(id: number, position: string, message: string): Promise<ApplicationItem> {
    const payload: ApplyRecruitmentReq = { position, message };
    return GameroApi.post<ApplicationItem>(`/recruitments/${id}/apply`, payload);
  }

  static listMyApplications(): Promise<PageData<ApplicationItem>> {
    return GameroApi.get<PageData<ApplicationItem>>('/applications/mine?page=1&page_size=20');
  }

  static getFeed(): Promise<FeedResponse> {
    return GameroApi.get<FeedResponse>('/feed?limit=20');
  }

  static getAdminStats(): Promise<PlatformStats> {
    return GameroApi.get<PlatformStats>('/admin/stats');
  }

  static listProjectTasks(projectId: number): Promise<ProjectTask[]> {
    return GameroApi.get<ProjectTask[]>(`/projects/${projectId}/tasks`);
  }

  static createProjectTask(projectId: number, title: string, description: string, priority: string): Promise<ProjectTask> {
    const payload: ProjectTaskReq = { title, description, status: 'todo', priority };
    return GameroApi.post<ProjectTask>(`/projects/${projectId}/tasks`, payload);
  }

  static updateProjectTask(projectId: number, taskId: number, title: string, description: string, status: string, priority: string): Promise<ProjectTask> {
    const payload: ProjectTaskReq = { title, description, status, priority };
    return GameroApi.patch<ProjectTask>(`/projects/${projectId}/tasks/${taskId}`, payload);
  }

  static deleteProjectTask(projectId: number, taskId: number): Promise<void> {
    return GameroApi.delete<void>(`/projects/${projectId}/tasks/${taskId}`);
  }

  static listPosts(): Promise<PageData<PostItem>> {
    return GameroApi.get<PageData<PostItem>>('/community/posts?page=1&page_size=10&sort=latest');
  }

  static createPost(title: string, content: string, images: string[] = [], videos: string[] = []): Promise<PostItem> {
    const payload: CreatePostReq = { title, content, images, videos };
    return GameroApi.post<PostItem>('/community/posts', payload);
  }

  static savePostImage(id: number, key: string): Promise<void> {
    const payload: MediaKeyReq = { key };
    return GameroApi.put<void>(`/community/posts/${id}/images`, payload);
  }

  static savePostVideo(id: number, key: string): Promise<void> {
    const payload: MediaKeyReq = { key };
    return GameroApi.put<void>(`/community/posts/${id}/videos`, payload);
  }

  static updatePost(id: number, title: string, content: string): Promise<PostItem> {
    const payload: CreatePostReq = { title, content };
    return GameroApi.patch<PostItem>(`/community/posts/${id}`, payload);
  }

  static deletePost(id: number): Promise<void> {
    return GameroApi.delete<void>(`/community/posts/${id}`);
  }

  static getPost(id: number): Promise<PostItem> {
    return GameroApi.get<PostItem>(`/community/posts/${id}`);
  }

  static listPostComments(id: number): Promise<PageData<CommentItem>> {
    return GameroApi.get<PageData<CommentItem>>(`/community/posts/${id}/comments?page=1&page_size=20&sort=latest`);
  }

  static createPostComment(id: number, content: string): Promise<CommentItem> {
    const payload: CreateCommentReq = { content };
    return GameroApi.post<CommentItem>(`/community/posts/${id}/comments`, payload);
  }

  static likePost(id: number): Promise<void> {
    return GameroApi.post<void>(`/community/posts/${id}/like`);
  }

  static unlikePost(id: number): Promise<void> {
    return GameroApi.delete<void>(`/community/posts/${id}/like`);
  }

  static listDevLogs(): Promise<PageData<DevLogItem>> {
    return GameroApi.get<PageData<DevLogItem>>('/discover/logs?page=1&page_size=10&sort=latest');
  }

  static createDevLog(projectId: number, title: string, content: string): Promise<DevLogItem> {
    const payload: CreateDevLogReq = {
      title,
      content,
      log_type: 'log',
      visibility: 'public',
      status: 'published'
    };
    return GameroApi.post<DevLogItem>(`/projects/${projectId}/logs`, payload);
  }

  static createRelease(projectId: number, version: string, title: string, content: string): Promise<DevLogItem> {
    const payload: CreateDevLogReq = {
      title,
      content,
      log_type: 'release',
      version,
      visibility: 'public',
      status: 'published'
    };
    return GameroApi.post<DevLogItem>(`/projects/${projectId}/logs`, payload);
  }

  static listReleases(projectId: number): Promise<PageData<DevLogItem>> {
    return GameroApi.get<PageData<DevLogItem>>(`/projects/${projectId}/releases?page=1&page_size=10`);
  }

  static updateDevLog(id: number, title: string, content: string): Promise<DevLogItem> {
    const payload: CreateDevLogReq = {
      title,
      content,
      log_type: 'log',
      visibility: 'public',
      status: 'published'
    };
    return GameroApi.patch<DevLogItem>(`/logs/${id}`, payload);
  }

  static deleteDevLog(id: number): Promise<void> {
    return GameroApi.delete<void>(`/logs/${id}`);
  }

  static listMyDevLogs(): Promise<PageData<DevLogItem>> {
    return GameroApi.get<PageData<DevLogItem>>('/me/logs?page=1&page_size=20');
  }

  static listCollectedLogs(): Promise<PageData<DevLogItem>> {
    return GameroApi.get<PageData<DevLogItem>>('/me/collections/logs?page=1&page_size=20');
  }

  static getDevLog(id: number): Promise<DevLogItem> {
    return GameroApi.get<DevLogItem>(`/logs/${id}`);
  }

  static listDevLogComments(id: number): Promise<PageData<CommentItem>> {
    return GameroApi.get<PageData<CommentItem>>(`/logs/${id}/comments?page=1&page_size=20&sort=latest`);
  }

  static createDevLogComment(id: number, content: string): Promise<CommentItem> {
    const payload: CreateCommentReq = { content };
    return GameroApi.post<CommentItem>(`/logs/${id}/comments`, payload);
  }

  static likeDevLog(id: number): Promise<void> {
    return GameroApi.post<void>(`/logs/${id}/like`);
  }

  static unlikeDevLog(id: number): Promise<void> {
    return GameroApi.delete<void>(`/logs/${id}/like`);
  }

  static listNotifications(): Promise<PageData<NotificationItem>> {
    return GameroApi.get<PageData<NotificationItem>>('/notifications?page=1&page_size=20');
  }

  static getNotificationPreferences(): Promise<NotificationPreferencesData> {
    return GameroApi.get<NotificationPreferencesData>('/notifications/preferences');
  }

  static updateNotificationPreference(type: string, enabled: boolean): Promise<void> {
    const payload: NotificationPreferenceReq = { type, enabled };
    return GameroApi.patch<void>('/notifications/preferences', payload);
  }

  static markNotificationRead(id: number): Promise<void> {
    return GameroApi.patch<void>(`/notifications/${id}/read`);
  }

  static markAllNotificationsRead(): Promise<void> {
    return GameroApi.patch<void>('/notifications/read-all');
  }

  static deleteNotification(id: number): Promise<void> {
    return GameroApi.delete<void>(`/notifications/${id}`);
  }

  static listMyReports(): Promise<PageData<ReportItem>> {
    return GameroApi.get<PageData<ReportItem>>('/me/reports?page=1&page_size=20');
  }

  static listCollectedPosts(): Promise<PageData<PostItem>> {
    return GameroApi.get<PageData<PostItem>>('/me/collections/posts?page=1&page_size=20');
  }

  static listCollectedProjects(): Promise<PageData<ProjectItem>> {
    return GameroApi.get<PageData<ProjectItem>>('/me/collections/projects?page=1&page_size=20');
  }

  static getUnreadCount(): Promise<UnreadCountResponse> {
    return GameroApi.get<UnreadCountResponse>('/notifications/unread-count');
  }

  static search(keyword: string, type: string = 'all'): Promise<PageData<SearchItem>> {
    return GameroApi.get<PageData<SearchItem>>(`/search?page=1&page_size=10&type=${type}&keyword=${encodeURIComponent(keyword)}`);
  }

  static listTalents(): Promise<PageData<TalentItem>> {
    return GameroApi.get<PageData<TalentItem>>('/talents?page=1&page_size=20');
  }

  static getUserProfile(userId: number): Promise<UserProfileItem> {
    return GameroApi.get<UserProfileItem>(`/users/${userId}`);
  }

  static getMyProfile(): Promise<UserProfileItem> {
    return GameroApi.get<UserProfileItem>('/users/me');
  }

  static updateMyProfile(nickname: string, bio: string, location: string): Promise<UserProfileItem> {
    const payload: UpdateProfileReq = { nickname, bio, location };
    return GameroApi.patch<UserProfileItem>('/users/me/profile', payload);
  }

  static setAvailability(isAvailable: boolean, coopPreference: string): Promise<void> {
    const payload: AvailabilityReq = { is_available: isAvailable, coop_preference: coopPreference };
    return GameroApi.patch<void>('/users/me/availability', payload);
  }

  static updateSkills(items: UserSkillItem[]): Promise<SkillsData> {
    const skills = items.map((item: UserSkillItem) => {
      const skill: SkillInputItem = {
        category: item.category,
        name: item.name,
        level: item.level,
        description: item.description || ''
      };
      return skill;
    });
    const payload: UpdateSkillsReq = { skills };
    return GameroApi.put<SkillsData>('/users/me/skills', payload);
  }

  static createPortfolio(name: string, type: string, link: string, description: string): Promise<PortfolioItem> {
    const payload: PortfolioReq = { name, type, link, image_url: '', description };
    return GameroApi.post<PortfolioItem>('/users/me/portfolio', payload);
  }

  static deletePortfolio(id: number): Promise<void> {
    return GameroApi.delete<void>(`/users/me/portfolio/${id}`);
  }

  static followUser(userId: number): Promise<void> {
    return GameroApi.post<void>(`/users/${userId}/follow`);
  }

  static unfollowUser(userId: number): Promise<void> {
    return GameroApi.delete<void>(`/users/${userId}/follow`);
  }

  static adminListUsers(): Promise<PageData<UserProfileItem>> {
    return GameroApi.get<PageData<UserProfileItem>>('/admin/users?page=1&page_size=20');
  }

  static adminBanUser(userId: number, reason: string): Promise<void> {
    const payload: BanUserReq = { banned: true, reason };
    return GameroApi.post<void>(`/admin/users/${userId}/ban`, payload);
  }

  static adminUnbanUser(userId: number): Promise<void> {
    return GameroApi.delete<void>(`/admin/users/${userId}/ban`);
  }

  static adminListReports(): Promise<PageData<ReportItem>> {
    return GameroApi.get<PageData<ReportItem>>('/admin/reports?page=1&page_size=20&status=pending');
  }

  static adminHandleReport(reportId: number, status: string, note: string): Promise<void> {
    const payload: HandleReportReq = { status, note };
    return GameroApi.patch<void>(`/admin/reports/${reportId}`, payload);
  }

  static adminHidePost(id: number): Promise<void> {
    return GameroApi.patch<void>(`/admin/posts/${id}/hide`);
  }

  static adminDeletePost(id: number): Promise<void> {
    return GameroApi.delete<void>(`/admin/posts/${id}`);
  }

  static adminHideLog(id: number): Promise<void> {
    return GameroApi.patch<void>(`/admin/logs/${id}/hide`);
  }

  static adminCloseRecruitment(id: number): Promise<void> {
    return GameroApi.delete<void>(`/admin/recruitments/${id}`);
  }

  static adminBanProject(id: number, reason: string): Promise<void> {
    const payload: BanProjectReq = { reason };
    return GameroApi.put<void>(`/admin/projects/${id}/ban`, payload);
  }

  static adminDeleteProject(id: number): Promise<void> {
    return GameroApi.delete<void>(`/admin/projects/${id}`);
  }

  static adminListSensitiveWords(): Promise<SensitiveWordsData> {
    return GameroApi.get<SensitiveWordsData>('/admin/sensitive-words');
  }

  static adminAddSensitiveWords(words: string[]): Promise<void> {
    const payload: SensitiveWordsReq = { words };
    return GameroApi.post<void>('/admin/sensitive-words', payload);
  }

  static adminDeleteSensitiveWord(word: string): Promise<void> {
    return GameroApi.delete<void>(`/admin/sensitive-words/${encodeURIComponent(word)}`);
  }
}

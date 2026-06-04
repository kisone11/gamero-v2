import type {
  ProjectStatus,
  ProjectGenre,
  MemberRole,
  DevLogType,
  DevLogVisibility,
  DevLogStatus,
  CooperationType,
  SkillCategory,
  SkillLevel,
  PortfolioType,
  UserRole,
  RecruitmentPosition,
  ApplicationStatus,
  InvitationStatus,
  ContactType,
  RecruitmentStatus,
} from '@/types/enums'

export type {
  ProjectStatus,
  ProjectGenre,
  MemberRole,
  DevLogType,
  DevLogVisibility,
  DevLogStatus,
  CooperationType,
  SkillCategory,
  SkillLevel,
  PortfolioType,
  UserRole,
  RecruitmentPosition,
  ApplicationStatus,
  InvitationStatus,
  ContactType,
  RecruitmentStatus,
} from '@/types/enums'

// ============================================================
// 通用
// ============================================================
export interface ApiResponse<T = null> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

// ============================================================
// Auth
// ============================================================
export interface UserInfo {
  id: number
  username: string
  nickname: string
  role: UserRole
  avatar_url: string | null
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  expires_at: string
  user: UserInfo
}

export interface SendEmailCodeReq {
  email: string
  scene?: 'register' | 'reset'
}
export interface RegisterByEmailReq {
  email: string
  code: string
  password: string
  nickname?: string
}
export interface LoginByEmailReq {
  email: string
  password: string
}
export interface RefreshTokenReq {
  refresh_token: string
}
export interface ChangePasswordReq {
  old_password: string
  new_password: string
}
export interface ResetPasswordByCodeReq {
  email: string
  code: string
  new_password: string
}
export interface DeleteAccountReq {
  confirmation: string
}

// ============================================================
// User
// ============================================================
export interface UserSkill {
  id: number
  user_id: number
  category: SkillCategory
  name: string
  level: SkillLevel
  description?: string
  is_custom: boolean
  created_at: string
}

export interface Portfolio {
  id: number
  user_id: number
  name: string
  type: PortfolioType
  link?: string
  image_url?: string
  description?: string
  created_at: string
}

export interface UserProfileProject {
  id: number
  name: string
  slug: string
  genre: ProjectGenre
  status: ProjectStatus
  cover_url?: string | null
  member_role: MemberRole
  contribution?: string
  joined_at: string
  left_at?: string | null
  is_active: boolean
}

export interface UserProfileResponse {
  id: number
  username: string
  nickname: string
  role?: UserRole           // admin 接口返回
  email?: string             // admin 接口返回
  bio?: string
  avatar_url?: string | null
  skills: UserSkill[]
  portfolio: Portfolio[]
  projects: UserProfileProject[]
  logs: DevLog[]
  posts: Post[]
  endorsements: CollabReview[]
  post_count: number
  log_count: number
  project_count: number
  followers_count: number
  following_count: number
  is_following: boolean
  is_banned?: boolean        // 仅 admin 接口返回
  created_at: string
  badges: string[]
  // 仅查看自己时返回
  is_available?: boolean
  coop_preference?: CooperationType
  location?: string
}

export interface UserSearchItem {
  id: number
  username: string
  nickname: string
  avatar_url: string
}

export interface FollowUserItem {
  id: number
  username: string
  nickname: string
  bio?: string
  avatar_url?: string | null
  followers_count: number
  following_count: number
  is_following: boolean
  followed_at: string
}

export interface UpdateProfileReq {
  nickname?: string
  bio?: string
  location?: string
}
export interface UpdateSkillsReq {
  skills: SkillInput[]
}
export interface SkillInput {
  category: SkillCategory
  name: string
  level: SkillLevel
  description?: string
}
export interface CreatePortfolioReq {
  name: string
  type: PortfolioType
  link?: string
  image_url?: string
  description?: string
}
export type UpdatePortfolioReq = Partial<CreatePortfolioReq>
export interface SetAvailabilityReq {
  is_available: boolean
  coop_preference?: CooperationType
  location?: string
}

// ============================================================
// Project
// ============================================================
export interface MemberDetail {
  id: number
  project_id: number
  user_id: number
  role: MemberRole
  contribution?: string
  joined_at: string
  left_at?: string | null
  username?: string
  nickname?: string
  avatar_url?: string | null
}

export interface ProjectDetail {
  id: number
  owner_id: number
  name: string
  slug: string
  description: string
  genre: ProjectGenre
  style_tags: string[]
  engine?: string | null
  platform?: string[] | null
  status: ProjectStatus
  visibility: 'public' | 'private'
  cover_url?: string | null
  screenshot_urls: string[]
  screenshot_keys?: string[]
  demo_url?: string | null
  store_url?: string | null
  video_url?: string | null
  video_urls?: string[]
  video_keys?: string[]
  follower_count: number
  like_count?: number
  collect_count?: number
  view_count?: number
  is_collected?: boolean
  is_followed?: boolean
  release_count?: number
  total_downloads?: number
  created_at: string
  updated_at: string
  members: MemberDetail[]
}

export interface ProjectListItem {
  id: number
  owner_id: number
  name: string
  slug: string
  description: string
  genre: ProjectGenre
  style_tags: string[]
  status: ProjectStatus
  cover_url?: string | null
  follower_count: number
  like_count?: number
  collect_count?: number
  view_count?: number
  member_count?: number
  created_at: string
  updated_at?: string
}

export interface TimelineEvent {
  id: number
  project_id: number
  event_type: string
  title?: string
  description: string
  metadata?: Record<string, unknown> | string
  created_at: string
}

export interface CreateProjectReq {
  name: string
  description?: string
  genre?: ProjectGenre
  style_tags?: string[]
  status?: ProjectStatus
  visibility?: 'public' | 'private'
  demo_url?: string
  store_url?: string
  engine?: string
  platform?: string[]
}
export type UpdateProjectReq = Partial<CreateProjectReq & { video_url?: string }>
export interface UpdateStatusReq {
  status: ProjectStatus
}
export interface AddMemberReq {
  user_id: number
  role: MemberRole
  contribution?: string
}
export interface UpdateMemberReq {
  role: MemberRole
  contribution?: string
}
// ============================================================
// DevLog
// ============================================================
export interface DevLogAuthor {
  id: number
  username: string
  nickname: string
  avatar_url?: string | null
}

export interface DevLog {
  id: number
  project_id: number
  author_id: number
  title: string
  content: string
  log_type: DevLogType
  version?: string
  cover_url?: string | null
  image_urls?: string[]
  image_keys?: string[]
  video_urls?: string[]
  video_keys?: string[]
  download_url?: string
  visibility: DevLogVisibility
  status: DevLogStatus
  like_count: number
  view_count: number
  comment_count: number
  collect_count: number
  is_liked?: boolean
  is_collected?: boolean
  download_count?: number
  created_at: string
  updated_at: string
}

export interface DevLogDetail extends DevLog {
  author: DevLogAuthor
  author_id: number
  author_nickname: string
  author_username: string
  author_avatar_url?: string | null
  project_name?: string
  project_slug?: string
  project_cover_url?: string | null
  screenshot_urls?: string[]
  like_count: number
  collect_count: number
  view_count: number
  comment_count: number
  is_liked?: boolean
  is_collected?: boolean
}

export interface LogComment {
  id: number
  log_id: number
  user_id: number
  content: string
  parent_id?: number | null
  parent_nickname?: string | null
  like_count: number
  is_liked: boolean
  is_deleted: boolean
  replies: LogComment[]
  created_at: string
  author?: {
    id: number
    username: string
    nickname: string
    avatar_url?: string | null
  }
  // legacy compat: backend nests author info under `author` object
  user_nickname?: string
  user_username?: string
  updated_at?: string
}

export interface CreateDevLogReq {
  title?: string
  content?: string
  log_type?: DevLogType
  version?: string
  visibility?: DevLogVisibility
  status?: DevLogStatus
}
export type UpdateDevLogReq = Partial<CreateDevLogReq & { download_url?: string }>
export interface CreateCommentReq {
  content: string
  reply_to_id?: number
  parent_id?: number
}

// ============================================================
// Community
// ============================================================
export interface PostAuthor {
  id: number
  username: string
  nickname: string
  avatar_url?: string | null
}

export interface Post {
  id: number
  author_id: number
  project_id?: number
  title: string
  content: string
  content_snippet?: string
  image_urls?: string[]
  video_urls?: string[]
  image_keys?: string[]
  video_keys?: string[]
  like_count: number
  comment_count: number
  collect_count: number
  view_count: number
  is_pinned: boolean
  is_liked?: boolean
  is_collected?: boolean
  author?: PostAuthor
  topic_ids?: number[]
  topics?: Topic[]
  topic?: Topic | null
  created_at: string
  updated_at: string
}

export interface Topic {
  id: number
  name: string
  description?: string
  icon_key?: string
  icon_url?: string | null
  post_count: number
  follower_count?: number
  is_default?: boolean
  created_at: string
}

export interface PostComment {
  id: number
  post_id: number
  author_id: number
  author: PostAuthor
  content: string
  reply_to_id?: number | null
  like_count: number
  is_liked: boolean
  is_deleted: boolean
  replies: PostComment[]
  created_at: string
}

export interface CreatePostReq {
  title: string
  content: string
  topic_ids?: number[]
  images?: string[]
}
export interface UpdatePostReq {
  title?: string
  content?: string
  topic_ids?: number[]
}

// ============================================================
// Team / Recruit
// ============================================================
export interface RecruitmentListItem {
  id: number
  project_id: number
  owner_id: number
  position: RecruitmentPosition
  headcount: number
  description?: string
  cooperation_type: CooperationType
  contact_type?: ContactType
  contact_info?: string | null
  expire_at: string
  status: RecruitmentStatus
  created_at: string
  project_name: string
  project_slug?: string
  cover_url?: string | null
}

export interface ApplicationDetail {
  id: number
  recruitment_id: number
  project_id?: number
  project_name?: string
  position: RecruitmentPosition
  message?: string
  status: ApplicationStatus
  created_at: string
  applicant_id: number
  applicant_username: string
  applicant_nickname: string
  applicant_avatar_url?: string | null
}

export interface TalentListItem {
  id: number
  username: string
  nickname: string
  avatar_url?: string | null
  bio?: string
  skills: UserSkill[]
  project_count: number
  is_available: boolean
  coop_preference?: CooperationType
  location?: string
}

export interface InvitationDetail {
  id: number
  project_id: number
  project_name: string
  inviter_id: number
  talent_id?: number
  inviter_nickname: string
  inviter_avatar_url?: string | null
  position: RecruitmentPosition
  message?: string
  status: InvitationStatus
  expire_at?: string
  created_at: string
}

export interface CreateRecruitmentReq {
  project_id: number
  position: RecruitmentPosition
  headcount: number
  description?: string
  cooperation_type: CooperationType
  contact_type?: ContactType
  contact_info?: string
  expire_days?: number
}

export type UpdateRecruitmentReq = Partial<Omit<CreateRecruitmentReq, 'project_id'>>

export interface ApplyRecruitmentReq {
  position: RecruitmentPosition
  message?: string
}

export interface InviteTalentReq {
  project_id: number
  talent_id: number
  position: RecruitmentPosition
  message?: string
}

// ============================================================
// Collab Reviews
// ============================================================
export interface CollabReview {
  id: number
  project_id: number
  reviewer_id: number
  reviewee_id: number
  rating: number
  comment: string
  tags: string[]
  supplement?: string | null
  created_at: string
  updated_at?: string
  reviewer_username?: string
  reviewer_nickname: string
  reviewer_avatar_url?: string | null
}

export interface CollabReviewReq {
  project_id: number
  reviewee_id: number
  rating: number
  comment: string
  tags?: string[]
}

// ============================================================
// Review（玩家评测）
// ============================================================
export interface ReviewItem {
  id: number
  project_id: number
  user_id: number
  user_nickname: string
  user_avatar_url?: string | null
  rating: number
  content: string
  like_count: number
  is_liked: boolean
  reply_content?: string
  replied_at?: string
  created_at: string
}

export interface ReviewRatingSummary {
  total: number
  average: number
  distribution: { rating: number; count: number }[]
}

export interface CreateReviewReq {
  rating: number
  content: string
  tags?: string[]
}

// ============================================================
// Notification
// ============================================================
export type NotificationType =
  | 'recruitment_applied'
  | 'application_approved'
  | 'application_rejected'
  | 'mention'
  | 'new_follower'
  | 'project_status_changed'
  | 'post_liked'
  | 'post_commented'
  | 'log_liked'
  | 'log_commented'
  | 'system'
  | 'talent_invited'
  | 'invite_accepted'
  | 'invite_declined'
  | 'project_banned'
  | 'project_unbanned'
  | 'report_handled'

export interface Notification {
  id: number
  user_id: number
  sender_id?: number
  type: NotificationType
  title: string
  content: string
  metadata?: Record<string, unknown>
  is_read: boolean
  created_at: string
}

export interface NotificationCategory {
  type_group: string
  unread_count: number
}

// ============================================================
// WebSocket 消息
// ============================================================
export interface WsNotificationMsg {
  type: 'notification'
  payload: Notification
}
export interface WsUnreadCountMsg {
  type: 'unread_count'
  payload: { count: number }
}
export type WsMessage = WsNotificationMsg | WsUnreadCountMsg

// ============================================================
// Stats / Admin
// ============================================================
export interface MyStats {
  total_projects: number
  total_logs: number
  total_posts: number
  total_followers: number
  total_following: number
  total_likes_received: number
  total_views: number
}

export interface ProjectStats {
  views: number
  likes: number
  collects: number
  followers: number
  log_count: number
}

export type ProjectTaskStatus = 'todo' | 'doing' | 'done'
export type ProjectTaskPriority = 'low' | 'medium' | 'high'

export interface ProjectTask {
  id: number
  project_id: number
  creator_id: number
  assignee_id?: number
  title: string
  description?: string
  status: ProjectTaskStatus
  priority: ProjectTaskPriority
  due_date?: string | null
  created_at: string
  updated_at: string
  creator_nickname?: string
  assignee_nickname?: string
}

export interface ProjectTaskReq {
  title?: string
  description?: string
  status?: ProjectTaskStatus
  priority?: ProjectTaskPriority
  assignee_id?: number
  due_date?: string
}


export interface PlatformStats {
  total_users: number
  active_users: number
  total_projects: number
  total_logs: number
  total_posts: number
  new_users_today: number
  new_projects_today: number
}

export interface DailyStats {
  date: string
  new_users: number
  active_users: number
  new_projects: number
  new_logs: number
  new_posts: number
}

export interface CreateReportReq {
  target_type: 'post' | 'log' | 'project' | 'comment' | 'user' | 'project_release'
  target_id: number
  reason: string
  supplement?: string
}

export interface Report {
  id: number
  reporter_id: number
  target_type: 'post' | 'log' | 'project' | 'comment' | 'user' | 'project_release'
  target_id: number
  reason: string
  status: 'pending' | 'escalated' | 'handled' | 'rejected'
  note?: string
  supplement?: string
  handler_name?: string
  created_at: string
  updated_at: string
}

export interface AuditLog {
  id: number
  admin_id: number
  action: string
  target_type: string
  target_id: number
  note?: string
  created_at: string
}

export interface Announcement {
  id: number
  title: string
  content: string
  level: 'info' | 'warning' | 'danger' | 'success'
  is_active: boolean
  is_pinned: boolean
  created_by: number
  published_at: string
  expires_at?: string | null
  created_at: string
  updated_at: string
}

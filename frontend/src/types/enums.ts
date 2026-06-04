// 项目状态 — matches backend model.ProjectStatus
export type ProjectStatus = 'preparing' | 'developing' | 'playable' | 'launched' | 'paused' | 'abandoned'

// 项目类型 — matches backend model.ProjectGenre
export type ProjectGenre = 'action' | 'rpg' | 'strategy' | 'simulator' | 'puzzle' | 'horror' | 'platform' | 'other'

// 成员角色 — matches backend model.ProjectMemberRole
export type MemberRole = 'owner' | 'lead_programmer' | 'lead_artist' | 'lead_designer' | 'sound' | 'tester' | 'member'

// 日志类型 — matches backend model.DevLogType
export type DevLogType = 'log' | 'release'

// 日志可见性 — matches backend model.DevLogVisibility
export type DevLogVisibility = 'public' | 'members_only'

// 日志状态 — matches backend model.DevLogStatus
export type DevLogStatus = 'draft' | 'published'

// 招募职位 — matches backend model.RecruitmentPosition
export type RecruitmentPosition = 'program' | 'art' | 'design' | 'sound'

// 合作类型 — matches backend model.CooperationType
export type CooperationType = 'online' | 'offline' | 'hybrid'

// 申请状态 — matches backend model.ApplicationStatus
export type ApplicationStatus = 'pending' | 'approved' | 'rejected' | 'withdrawn'

// 邀请状态 — matches backend model.InvitationStatus
export type InvitationStatus = 'pending' | 'accepted' | 'declined' | 'withdrawn' | 'expired'

// 技能分类 — matches backend model.SkillCategory
export type SkillCategory = 'program' | 'art' | 'design' | 'sound' | 'custom'

// 技能级别 — matches backend model.SkillLevel
export type SkillLevel = 'beginner' | 'intermediate' | 'advanced'

// 作品集类型 — matches backend model.PortfolioType
export type PortfolioType = 'game' | 'demo' | 'art' | 'code'

// 用户角色 — matches backend model.UserRole
export type UserRole = 'user' | 'creator' | 'moderator' | 'admin' | 'superadmin'

// 联系方式类型 — matches backend model.ContactType
export type ContactType = 'wechat' | 'qq' | 'platform'

// 招募状态 — matches backend model.RecruitmentStatus
export type RecruitmentStatus = 'open' | 'closed' | 'expired'

// 项目风格标签 — matches backend model.ProjectStyleTag
export type ProjectStyleTag = '2D' | '3D' | 'pixel' | 'realist' | 'cartoon' | 'cyberpunk' | 'fantasy' | 'scifi'

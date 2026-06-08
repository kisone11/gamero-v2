import { GameroTheme } from '../common/GameroTheme';
import { FeedCardModel, ProjectCardModel, RecruitCardModel } from '../models/GameroModels';
import { ApiConfig } from '../services/ApiConfig';
import { ApplicationItem, AuthUser, CommentItem, DevLogItem, FeedItem, GameroApi, NotificationItem, NotificationPreferenceItem, PlatformStats, PortfolioItem, PostItem, ProjectDetailItem, ProjectItem, ProjectMemberItem, ProjectTask, RecruitItem, ReportItem, SearchItem, TalentItem, UserProfileItem, UserSkillItem } from '../services/GameroApi';

@Entry
@Component
struct Index {
  @State currentTab: number = 0;
  @State backendUrl: string = ApiConfig.baseUrl;
  @State isLoading: boolean = false;
  @State errorText: string = '';
  @State successText: string = '';
  @State authMode: string = 'login';
  @State loginEmail: string = ApiConfig.adminEmail;
  @State loginPassword: string = ApiConfig.adminPassword;
  @State registerEmail: string = '';
  @State registerCode: string = '';
  @State registerPassword: string = '';
  @State registerNickname: string = '';
  @State featureView: string = 'menu';
  @State searchKeyword: string = '';
  @State talents: TalentItem[] = [];
  @State selectedUser: UserProfileItem | null = null;
  @State myProfile: UserProfileItem | null = null;
  @State profileNickname: string = '';
  @State profileBio: string = '';
  @State profileLocation: string = '';
  @State profileAvailable: boolean = false;
  @State profileCoopPreference: string = 'online';
  @State profileSkills: UserSkillItem[] = [];
  @State portfolioItems: PortfolioItem[] = [];
  @State skillCategory: string = 'program';
  @State skillName: string = '';
  @State skillLevel: string = 'intermediate';
  @State skillDescription: string = '';
  @State portfolioName: string = '';
  @State portfolioType: string = 'game';
  @State portfolioLink: string = '';
  @State portfolioDescription: string = '';
  @State memberUserIdText: string = '';
  @State memberRole: string = 'member';
  @State memberContribution: string = '';
  @State currentUser: AuthUser | null = null;
  @State feeds: FeedCardModel[] = [];
  @State projectItems: ProjectItem[] = [];
  @State collectedProjects: ProjectItem[] = [];
  @State selectedProject: ProjectDetailItem | null = null;
  @State projectName: string = '';
  @State projectDescription: string = '';
  @State projectGenre: string = 'other';
  @State projectStatus: string = 'preparing';
  @State isEditingProject: boolean = false;
  @State projects: ProjectCardModel[] = [];
  @State recruits: RecruitCardModel[] = [];
  @State recruitItems: RecruitItem[] = [];
  @State selectedRecruit: RecruitItem | null = null;
  @State applications: ApplicationItem[] = [];
  @State applyMessage: string = '';
  @State recruitProjectId: number = 0;
  @State recruitPosition: string = 'program';
  @State recruitHeadcount: number = 1;
  @State recruitDescription: string = '';
  @State recruitCooperationType: string = 'online';
  @State isEditingRecruit: boolean = false;
  @State tasks: ProjectTask[] = [];
  @State taskTitle: string = '';
  @State taskDescription: string = '';
  @State taskPriority: string = 'medium';
  @State posts: PostItem[] = [];
  @State collectedPosts: PostItem[] = [];
  @State devLogs: DevLogItem[] = [];
  @State myDevLogs: DevLogItem[] = [];
  @State collectedLogs: DevLogItem[] = [];
  @State myReports: ReportItem[] = [];
  @State selectedPost: PostItem | null = null;
  @State selectedDevLog: DevLogItem | null = null;
  @State postBackView: string = 'posts';
  @State devLogBackView: string = 'logs';
  @State comments: CommentItem[] = [];
  @State commentInput: string = '';
  @State postTitle: string = '';
  @State postContent: string = '';
  @State postImageKeys: string[] = [];
  @State postVideoKeys: string[] = [];
  @State isEditingPost: boolean = false;
  @State devLogTitle: string = '';
  @State devLogContent: string = '';
  @State isEditingDevLog: boolean = false;
  @State releases: DevLogItem[] = [];
  @State releaseVersion: string = '';
  @State releaseTitle: string = '';
  @State releaseContent: string = '';
  @State notifications: NotificationItem[] = [];
  @State notificationPreferences: NotificationPreferenceItem[] = [];
  @State searchResults: SearchItem[] = [];
  @State unreadCount: number = 0;
  @State stats: PlatformStats | null = null;
  @State adminUsers: UserProfileItem[] = [];
  @State adminReports: ReportItem[] = [];
  @State adminBanReason: string = '违反社区规则';
  @State adminHandleNote: string = '已处理';
  @State adminTargetIdText: string = '';
  @State sensitiveWords: string[] = [];
  @State sensitiveWordInput: string = '';
  @State homeLoadedOnce: boolean = false;

  aboutToAppear(): void {
    this.errorText = '';
    if (this.currentUser) {
      this.refreshHomeData(false);
    } else {
      this.loadPublicData();
    }
  }

  async loadPublicData(): Promise<void> {
    try {
      const projectPage = await GameroApi.listProjects();
      this.projectItems = projectPage.list;
      this.projects = projectPage.list.map((item: ProjectItem) => {
        let card: ProjectCardModel = {
          title: item.name,
          desc: item.description || '暂无项目简介',
          status: this.statusText(item.status),
          genre: `${this.genreText(item.genre)} · 项目`,
          followers: `${item.follower_count || 0} 关注`
        };
        return card;
      });

      const recruitPage = await GameroApi.listRecruitments();
      this.recruitItems = recruitPage.list;
      this.recruits = recruitPage.list.map((item: RecruitItem) => {
        let card: RecruitCardModel = {
          project: item.project_name || `项目 #${item.project_id}`,
          role: this.positionText(item.position),
          mode: this.coopText(item.cooperation_type),
          days: this.daysLeft(item.expire_at)
        };
        return card;
      });
    } catch {
      this.projects = [];
      this.projectItems = [];
      this.recruits = [];
      this.recruitItems = [];
      this.errorText = '项目或招募数据加载失败，请确认后端已启动。';
    }
  }

  async openRecruitDetail(id: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    this.devLogBackView = this.featureView === 'myContent' ? 'myContent' : this.featureView === 'collections' ? 'collections' : 'logs';
    try {
      this.selectedRecruit = await GameroApi.getRecruitment(id);
      this.applyMessage = '';
    } catch {
      this.errorText = '招募详情加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  closeRecruitDetail(): void {
    this.selectedRecruit = null;
    this.isEditingRecruit = false;
  }

  syncDefaultRecruitProject(): void {
    if (this.recruitProjectId === 0 && this.projectItems.length > 0) {
      this.recruitProjectId = this.projectItems[0].id;
    }
  }

  async createRecruitmentFromForm(): Promise<void> {
    this.syncDefaultRecruitProject();
    if (this.recruitProjectId === 0) {
      this.errorText = '请先创建一个项目。';
      return;
    }
    if (this.recruitDescription.trim().length === 0) {
      this.errorText = '请输入招募说明。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const created = await GameroApi.createRecruitment(
        this.recruitProjectId,
        this.recruitPosition,
        this.recruitHeadcount,
        this.recruitDescription.trim(),
        this.recruitCooperationType
      );
      await this.loadPublicData();
      await this.openRecruitDetail(created.id);
      this.recruitDescription = '';
      this.successText = '招募已创建。';
    } catch {
      this.errorText = '创建招募失败。';
    } finally {
      this.isLoading = false;
    }
  }

  startEditRecruit(): void {
    if (!this.selectedRecruit) {
      return;
    }
    this.recruitProjectId = this.selectedRecruit.project_id;
    this.recruitPosition = this.selectedRecruit.position;
    this.recruitHeadcount = this.selectedRecruit.headcount;
    this.recruitDescription = this.selectedRecruit.description || '';
    this.recruitCooperationType = this.selectedRecruit.cooperation_type;
    this.isEditingRecruit = true;
  }

  cancelEditRecruit(): void {
    this.recruitDescription = '';
    this.recruitPosition = 'program';
    this.recruitHeadcount = 1;
    this.recruitCooperationType = 'online';
    this.isEditingRecruit = false;
  }

  async saveRecruitEdit(): Promise<void> {
    if (!this.selectedRecruit) {
      return;
    }
    if (this.recruitDescription.trim().length === 0) {
      this.errorText = '请输入招募说明。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      this.selectedRecruit = await GameroApi.updateRecruitment(
        this.selectedRecruit.id,
        this.recruitPosition,
        this.recruitHeadcount,
        this.recruitDescription.trim(),
        this.recruitCooperationType
      );
      await this.loadPublicData();
      this.cancelEditRecruit();
      this.successText = '招募已更新。';
    } catch {
      this.errorText = '更新招募失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async closeSelectedRecruit(): Promise<void> {
    if (!this.selectedRecruit) {
      return;
    }
    this.isLoading = true;
    try {
      await GameroApi.closeRecruitment(this.selectedRecruit.id);
      this.selectedRecruit = await GameroApi.getRecruitment(this.selectedRecruit.id);
      await this.loadPublicData();
      this.successText = '招募已关闭。';
    } catch {
      this.errorText = '关闭招募失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async reopenSelectedRecruit(): Promise<void> {
    if (!this.selectedRecruit) {
      return;
    }
    this.isLoading = true;
    try {
      await GameroApi.reopenRecruitment(this.selectedRecruit.id);
      this.selectedRecruit = await GameroApi.getRecruitment(this.selectedRecruit.id);
      await this.loadPublicData();
      this.successText = '招募已重开。';
    } catch {
      this.errorText = '重开招募失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async deleteSelectedRecruit(): Promise<void> {
    if (!this.selectedRecruit) {
      return;
    }
    this.isLoading = true;
    try {
      await GameroApi.deleteRecruitment(this.selectedRecruit.id);
      this.selectedRecruit = null;
      await this.loadPublicData();
      this.successText = '招募已删除。';
    } catch {
      this.errorText = '删除招募失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async applySelectedRecruit(): Promise<void> {
    if (!this.selectedRecruit) {
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.applyRecruitment(this.selectedRecruit.id, this.selectedRecruit.position, this.applyMessage.trim());
      this.successText = '申请已提交。';
      this.applyMessage = '';
      await this.loadMyApplications();
    } catch {
      this.errorText = '申请失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadMyApplications(): Promise<void> {
    try {
      const page = await GameroApi.listMyApplications();
      this.applications = page.list;
    } catch {}
  }

  async openFeature(view: string): Promise<void> {
    if (view === 'home') {
      await this.selectTab(0);
      return;
    }
    if (view === 'projects') {
      await this.selectTab(1);
      return;
    }
    if (view === 'recruit') {
      await this.selectTab(2);
      return;
    }
    this.featureView = view;
    this.currentTab = 3;
    this.errorText = '';
    this.successText = '';
    if (view === 'posts') {
      await this.loadPosts();
    } else if (view === 'logs') {
      await this.loadDevLogs();
    } else if (view === 'notifications') {
      await this.loadNotifications();
    } else if (view === 'search') {
      this.searchResults = [];
    } else if (view === 'talents') {
      await this.loadTalents();
    } else if (view === 'settings') {
      await this.loadMyProfile();
      await this.loadNotificationPreferences();
    } else if (view === 'admin') {
      await this.loadAdminData();
    } else if (view === 'myContent') {
      await this.loadMyContent();
    } else if (view === 'collections') {
      await this.loadCollections();
    }
  }

  async selectTab(index: number): Promise<void> {
    this.currentTab = index;
    this.errorText = '';
    this.successText = '';
    if (index === 0 && this.currentUser) {
      await this.refreshHomeData(false);
    } else if (index === 1) {
      await this.loadPublicData();
    } else if (index === 2) {
      await this.loadPublicData();
      await this.loadMyApplications();
    } else if (index === 3) {
      if (this.featureView === 'posts') {
        await this.loadPosts();
      } else if (this.featureView === 'logs') {
        await this.loadDevLogs();
      } else if (this.featureView === 'notifications') {
        await this.loadNotifications();
      } else if (this.featureView === 'talents') {
        await this.loadTalents();
      }
    } else if (index === 4 && this.currentUser) {
      await this.loadMyProfile();
    }
  }

  async loadCollections(): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      const projectPage = await GameroApi.listCollectedProjects();
      const postPage = await GameroApi.listCollectedPosts();
      const logPage = await GameroApi.listCollectedLogs();
      this.collectedProjects = projectPage.list;
      this.collectedPosts = postPage.list;
      this.collectedLogs = logPage.list;
    } catch {
      this.collectedProjects = [];
      this.collectedPosts = [];
      this.collectedLogs = [];
      this.errorText = '收藏列表加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadNotificationPreferences(): Promise<void> {
    try {
      const data = await GameroApi.getNotificationPreferences();
      this.notificationPreferences = data.preferences;
    } catch {
      this.notificationPreferences = [];
      this.errorText = '通知偏好加载失败。';
    }
  }

  notificationEnabled(type: string): boolean {
    const found = this.notificationPreferences.find((item: NotificationPreferenceItem) => item.type === type);
    return found ? found.enabled : true;
  }

  async toggleNotificationPreference(type: string): Promise<void> {
    const next = !this.notificationEnabled(type);
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.updateNotificationPreference(type, next);
      await this.loadNotificationPreferences();
      this.successText = '通知偏好已更新。';
    } catch {
      this.errorText = '通知偏好更新失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadMyContent(): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      const logPage = await GameroApi.listMyDevLogs();
      const reportPage = await GameroApi.listMyReports();
      this.myDevLogs = logPage.list;
      this.myReports = reportPage.list;
    } catch {
      this.myDevLogs = [];
      this.myReports = [];
      this.errorText = '我的日志或举报加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadAdminData(): Promise<void> {
    if (!this.currentUser || (this.currentUser.role !== 'admin' && this.currentUser.role !== 'superadmin')) {
      this.errorText = '当前账号没有管理后台权限。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      this.stats = await GameroApi.getAdminStats();
      const userPage = await GameroApi.adminListUsers();
      const reportPage = await GameroApi.adminListReports();
      const sensitiveData = await GameroApi.adminListSensitiveWords();
      this.adminUsers = userPage.list;
      this.adminReports = reportPage.list;
      this.sensitiveWords = sensitiveData.words;
    } catch {
      this.adminUsers = [];
      this.adminReports = [];
      this.sensitiveWords = [];
      this.errorText = '管理后台数据加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminBanUser(userId: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.adminBanUser(userId, this.adminBanReason.trim() || '违反社区规则');
      this.successText = '用户已封禁。';
      await this.loadAdminData();
    } catch {
      this.errorText = '封禁用户失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminUnbanUser(userId: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.adminUnbanUser(userId);
      this.successText = '用户已解封。';
      await this.loadAdminData();
    } catch {
      this.errorText = '解封用户失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminRejectReport(reportId: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.adminHandleReport(reportId, 'rejected', this.adminHandleNote.trim() || '不予处理');
      this.successText = '举报已驳回。';
      await this.loadAdminData();
    } catch {
      this.errorText = '驳回举报失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminHandleReport(report: ReportItem): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      if (report.target_type === 'post') {
        await GameroApi.adminDeletePost(report.target_id);
      } else if (report.target_type === 'log' || report.target_type === 'project_release') {
        await GameroApi.adminHideLog(report.target_id);
      } else if (report.target_type === 'project') {
        await GameroApi.adminBanProject(report.target_id, this.adminHandleNote.trim() || '举报处理下架');
      } else if (report.target_type === 'recruitment') {
        await GameroApi.adminCloseRecruitment(report.target_id);
      }
      await GameroApi.adminHandleReport(report.id, 'handled', this.adminHandleNote.trim() || '已处理');
      this.successText = '举报已处理。';
      await this.loadAdminData();
    } catch {
      this.errorText = '处理举报失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminDeleteProjectFromReport(report: ReportItem): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.adminDeleteProject(report.target_id);
      await GameroApi.adminHandleReport(report.id, 'handled', this.adminHandleNote.trim() || '项目已删除');
      this.successText = '项目已删除。';
      await this.loadAdminData();
    } catch {
      this.errorText = '删除项目失败。';
    } finally {
      this.isLoading = false;
    }
  }

  adminTargetId(): number {
    const id = Number.parseInt(this.adminTargetIdText, 10);
    return Number.isFinite(id) && id > 0 ? id : 0;
  }

  async adminDeletePostById(): Promise<void> {
    const id = this.adminTargetId();
    if (id === 0) {
      this.errorText = '请输入有效 ID。';
      return;
    }
    this.isLoading = true;
    try {
      await GameroApi.adminDeletePost(id);
      this.successText = '帖子已删除。';
    } catch {
      this.errorText = '删除帖子失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminCloseRecruitmentById(): Promise<void> {
    const id = this.adminTargetId();
    if (id === 0) {
      this.errorText = '请输入有效 ID。';
      return;
    }
    this.isLoading = true;
    try {
      await GameroApi.adminCloseRecruitment(id);
      this.successText = '招募已关闭。';
    } catch {
      this.errorText = '关闭招募失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminBanProjectById(): Promise<void> {
    const id = this.adminTargetId();
    if (id === 0) {
      this.errorText = '请输入有效 ID。';
      return;
    }
    this.isLoading = true;
    try {
      await GameroApi.adminBanProject(id, this.adminHandleNote.trim() || '管理员下架');
      this.successText = '项目已下架。';
    } catch {
      this.errorText = '下架项目失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminDeleteProjectById(): Promise<void> {
    const id = this.adminTargetId();
    if (id === 0) {
      this.errorText = '请输入有效 ID。';
      return;
    }
    this.isLoading = true;
    try {
      await GameroApi.adminDeleteProject(id);
      this.successText = '项目已删除。';
    } catch {
      this.errorText = '删除项目失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminAddSensitiveWords(): Promise<void> {
    const words = this.sensitiveWordInput.split(',').map((word: string) => word.trim()).filter((word: string) => word.length > 0);
    if (words.length === 0) {
      this.errorText = '请输入敏感词，多个词用英文逗号分隔。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.adminAddSensitiveWords(words);
      this.sensitiveWordInput = '';
      this.successText = '敏感词已添加。';
      await this.loadAdminData();
    } catch {
      this.errorText = '添加敏感词失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async adminDeleteSensitiveWord(word: string): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.adminDeleteSensitiveWord(word);
      this.successText = '敏感词已删除。';
      await this.loadAdminData();
    } catch {
      this.errorText = '删除敏感词失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadTalents(): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      const page = await GameroApi.listTalents();
      this.talents = page.list;
    } catch {
      this.talents = [];
      this.errorText = '人才列表加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async openUserProfile(userId: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      this.selectedUser = await GameroApi.getUserProfile(userId);
      this.featureView = 'userProfile';
    } catch {
      this.errorText = '用户主页加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async toggleFollowSelectedUser(): Promise<void> {
    if (!this.selectedUser) {
      return;
    }
    try {
      if (this.selectedUser.is_following) {
        await GameroApi.unfollowUser(this.selectedUser.id);
        this.selectedUser.is_following = false;
        this.selectedUser.followers_count = Math.max(0, this.selectedUser.followers_count - 1);
        this.successText = '已取消关注用户。';
      } else {
        await GameroApi.followUser(this.selectedUser.id);
        this.selectedUser.is_following = true;
        this.selectedUser.followers_count = this.selectedUser.followers_count + 1;
        this.successText = '已关注用户。';
      }
    } catch {
      this.errorText = '关注用户失败。';
    }
  }

  async loadMyProfile(): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      this.myProfile = await GameroApi.getMyProfile();
      this.profileNickname = this.myProfile.nickname;
      this.profileBio = this.myProfile.bio || '';
      this.profileLocation = this.myProfile.location || '';
      this.profileAvailable = this.myProfile.is_available || false;
      this.profileCoopPreference = this.myProfile.coop_preference || 'online';
      this.profileSkills = this.myProfile.skills || [];
      this.portfolioItems = this.myProfile.portfolio || [];
    } catch {
      this.errorText = '个人资料加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async saveMyProfile(): Promise<void> {
    if (this.profileNickname.trim().length === 0) {
      this.errorText = '请输入昵称。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      this.myProfile = await GameroApi.updateMyProfile(this.profileNickname.trim(), this.profileBio.trim(), this.profileLocation.trim());
      await GameroApi.setAvailability(this.profileAvailable, this.profileCoopPreference);
      this.successText = '个人资料已保存。';
    } catch {
      this.errorText = '保存个人资料失败。';
    } finally {
      this.isLoading = false;
    }
  }

  addSkillDraft(): void {
    if (this.skillName.trim().length === 0) {
      this.errorText = '请输入技能名称。';
      return;
    }
    const skill: UserSkillItem = {
      id: -Date.now(),
      user_id: this.currentUser ? this.currentUser.id : 0,
      category: this.skillCategory,
      name: this.skillName.trim(),
      level: this.skillLevel,
      description: this.skillDescription.trim()
    };
    this.profileSkills = this.profileSkills.concat([skill]);
    this.skillName = '';
    this.skillDescription = '';
  }

  removeSkillDraft(id: number): void {
    this.profileSkills = this.profileSkills.filter((skill: UserSkillItem) => skill.id !== id);
  }

  async saveSkills(): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      const data = await GameroApi.updateSkills(this.profileSkills);
      this.profileSkills = data.skills;
      this.successText = '技能已保存。';
    } catch {
      this.errorText = '保存技能失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async createPortfolioItem(): Promise<void> {
    if (this.portfolioName.trim().length === 0) {
      this.errorText = '请输入作品名称。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const item = await GameroApi.createPortfolio(this.portfolioName.trim(), this.portfolioType, this.portfolioLink.trim(), this.portfolioDescription.trim());
      this.portfolioItems = this.portfolioItems.concat([item]);
      this.portfolioName = '';
      this.portfolioLink = '';
      this.portfolioDescription = '';
      this.successText = '作品已添加。';
    } catch {
      this.errorText = '添加作品失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async deletePortfolioItem(id: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.deletePortfolio(id);
      this.portfolioItems = this.portfolioItems.filter((item: PortfolioItem) => item.id !== id);
      this.successText = '作品已删除。';
    } catch {
      this.errorText = '删除作品失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async openProjectDetail(id: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    this.postBackView = this.featureView === 'collections' ? 'collections' : 'posts';
    try {
      this.selectedProject = await GameroApi.getProject(id);
      this.tasks = await GameroApi.listProjectTasks(id);
      const releasePage = await GameroApi.listReleases(id);
      this.releases = releasePage.list;
      this.currentTab = 1;
    } catch {
      this.errorText = '项目详情加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async createProjectFromForm(): Promise<void> {
    if (this.projectName.trim().length === 0) {
      this.errorText = '请输入项目名称。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    try {
      const created = await GameroApi.createProject(
        this.projectName.trim(),
        this.projectDescription.trim(),
        this.projectGenre,
        this.projectStatus
      );
      this.projectName = '';
      this.projectDescription = '';
      this.projectGenre = 'other';
      this.projectStatus = 'preparing';
      await this.loadPublicData();
      await this.openProjectDetail(created.id);
      this.successText = '项目已创建。';
    } catch {
      this.errorText = '创建项目失败。';
    } finally {
      this.isLoading = false;
    }
  }

  closeProjectDetail(): void {
    this.selectedProject = null;
    this.isEditingProject = false;
  }

  startEditProject(): void {
    if (!this.selectedProject) {
      return;
    }
    this.projectName = this.selectedProject.name;
    this.projectDescription = this.selectedProject.description || '';
    this.projectGenre = this.selectedProject.genre || 'other';
    this.projectStatus = this.selectedProject.status || 'preparing';
    this.isEditingProject = true;
  }

  cancelEditProject(): void {
    this.projectName = '';
    this.projectDescription = '';
    this.projectGenre = 'other';
    this.projectStatus = 'preparing';
    this.isEditingProject = false;
  }

  async saveProjectEdit(): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    if (this.projectName.trim().length === 0) {
      this.errorText = '请输入项目名称。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    try {
      this.selectedProject = await GameroApi.updateProject(
        this.selectedProject.id,
        this.projectName.trim(),
        this.projectDescription.trim(),
        this.projectGenre,
        this.projectStatus
      );
      await this.loadPublicData();
      this.cancelEditProject();
      this.successText = '项目已更新。';
    } catch {
      this.errorText = '更新项目失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async deleteSelectedProject(): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    try {
      await GameroApi.deleteProject(this.selectedProject.id);
      this.selectedProject = null;
      await this.loadPublicData();
      this.successText = '项目已删除。';
    } catch {
      this.errorText = '删除项目失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async uploadProjectCover(): Promise<void> {
    if (!this.selectedProject) {
      this.errorText = '请先进入项目详情。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const key = await GameroApi.pickAndUploadMedia('cover', this.selectedProject.id, 'cover.jpg', 'image/jpeg', false);
      await GameroApi.saveProjectCover(this.selectedProject.id, key);
      this.selectedProject = await GameroApi.getProject(this.selectedProject.id);
      this.successText = '项目封面已上传。';
    } catch {
      this.errorText = '上传封面失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async uploadProjectScreenshot(): Promise<void> {
    if (!this.selectedProject) {
      this.errorText = '请先进入项目详情。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const key = await GameroApi.pickAndUploadMedia('screenshot', this.selectedProject.id, 'screenshot.jpg', 'image/jpeg', false);
      await GameroApi.saveProjectScreenshot(this.selectedProject.id, key);
      this.selectedProject = await GameroApi.getProject(this.selectedProject.id);
      this.successText = '项目截图已上传。';
    } catch {
      this.errorText = '上传截图失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async uploadProjectVideo(): Promise<void> {
    if (!this.selectedProject) {
      this.errorText = '请先进入项目详情。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const key = await GameroApi.pickAndUploadMedia('project-video', this.selectedProject.id, 'project-video.mp4', 'video/mp4', true);
      await GameroApi.saveProjectVideo(this.selectedProject.id, key);
      this.selectedProject = await GameroApi.getProject(this.selectedProject.id);
      this.successText = '项目视频已上传。';
    } catch {
      this.errorText = '上传视频失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async toggleProjectFollow(): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    try {
      if (this.selectedProject.is_followed) {
        await GameroApi.unfollowProject(this.selectedProject.id);
        this.selectedProject.is_followed = false;
        this.selectedProject.follower_count = Math.max(0, this.selectedProject.follower_count - 1);
        this.successText = '已取消关注项目。';
      } else {
        await GameroApi.followProject(this.selectedProject.id);
        this.selectedProject.is_followed = true;
        this.selectedProject.follower_count = this.selectedProject.follower_count + 1;
        this.successText = '已关注项目。';
      }
    } catch {
      this.errorText = '关注操作失败。';
    }
  }

  async toggleProjectCollect(): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    try {
      if (this.selectedProject.is_collected) {
        await GameroApi.uncollectProject(this.selectedProject.id);
        this.selectedProject.is_collected = false;
        this.successText = '已取消收藏项目。';
      } else {
        await GameroApi.collectProject(this.selectedProject.id);
        this.selectedProject.is_collected = true;
        this.successText = '已收藏项目。';
      }
    } catch {
      this.errorText = '收藏操作失败。';
    }
  }

  async reloadSelectedProject(): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    this.selectedProject = await GameroApi.getProject(this.selectedProject.id);
    const releasePage = await GameroApi.listReleases(this.selectedProject.id);
    this.releases = releasePage.list;
  }

  async createReleaseForSelectedProject(): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    if (this.releaseVersion.trim().length === 0 || this.releaseTitle.trim().length === 0) {
      this.errorText = '请输入版本号和标题。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.createRelease(
        this.selectedProject.id,
        this.releaseVersion.trim(),
        this.releaseTitle.trim(),
        this.releaseContent.trim()
      );
      const page = await GameroApi.listReleases(this.selectedProject.id);
      this.releases = page.list;
      this.releaseVersion = '';
      this.releaseTitle = '';
      this.releaseContent = '';
      this.successText = '版本已发布。';
    } catch {
      this.errorText = '发布版本失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async addMemberToSelectedProject(): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    const userId = Number.parseInt(this.memberUserIdText, 10);
    if (!Number.isFinite(userId) || userId <= 0) {
      this.errorText = '请输入有效的用户 ID。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.addProjectMember(this.selectedProject.id, userId, this.memberRole, this.memberContribution.trim());
      await this.reloadSelectedProject();
      this.memberUserIdText = '';
      this.memberContribution = '';
      this.successText = '成员已添加。';
    } catch {
      this.errorText = '添加成员失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async updateMemberRole(userId: number, role: string): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.updateProjectMember(this.selectedProject.id, userId, role, '');
      await this.reloadSelectedProject();
      this.successText = '成员角色已更新。';
    } catch {
      this.errorText = '更新成员失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async removeMemberFromSelectedProject(userId: number): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.removeProjectMember(this.selectedProject.id, userId);
      await this.reloadSelectedProject();
      this.successText = '成员已移除。';
    } catch {
      this.errorText = '移除成员失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async leaveSelectedProject(): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.leaveProject(this.selectedProject.id);
      await this.reloadSelectedProject();
      this.successText = '已退出项目。';
    } catch {
      this.errorText = '退出项目失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async createTaskForSelectedProject(): Promise<void> {
    if (!this.selectedProject || this.taskTitle.trim().length === 0) {
      this.errorText = '请输入任务标题。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.createProjectTask(this.selectedProject.id, this.taskTitle.trim(), this.taskDescription.trim(), this.taskPriority);
      this.tasks = await GameroApi.listProjectTasks(this.selectedProject.id);
      this.taskTitle = '';
      this.taskDescription = '';
      this.taskPriority = 'medium';
      this.successText = '任务已创建。';
    } catch {
      this.errorText = '创建任务失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async moveTask(task: ProjectTask, status: string): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.updateProjectTask(
        this.selectedProject.id,
        task.id,
        task.title,
        task.description || '',
        status,
        task.priority
      );
      this.tasks = await GameroApi.listProjectTasks(this.selectedProject.id);
      this.successText = '任务状态已更新。';
    } catch {
      this.errorText = '更新任务失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async removeTask(taskId: number): Promise<void> {
    if (!this.selectedProject) {
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.deleteProjectTask(this.selectedProject.id, taskId);
      this.tasks = await GameroApi.listProjectTasks(this.selectedProject.id);
      this.successText = '任务已删除。';
    } catch {
      this.errorText = '删除任务失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadPosts(): Promise<void> {
    this.isLoading = true;
    try {
      const page = await GameroApi.listPosts();
      this.posts = page.list;
    } catch {
      this.posts = [];
      this.errorText = '社区帖子加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async createCommunityPost(): Promise<void> {
    if (this.postTitle.trim().length === 0 || this.postContent.trim().length === 0) {
      this.errorText = '请输入帖子标题和内容。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.createPost(this.postTitle.trim(), this.postContent.trim(), this.postImageKeys, this.postVideoKeys);
      this.postTitle = '';
      this.postContent = '';
      this.postImageKeys = [];
      this.postVideoKeys = [];
      await this.loadPosts();
      this.successText = '帖子已发布。';
    } catch {
      this.errorText = '发布帖子失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async uploadPostImageToSelectedPost(): Promise<void> {
    if (!this.selectedPost) {
      this.errorText = '请先进入帖子详情。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const key = await GameroApi.pickAndUploadMedia('post-image', this.selectedPost.id, 'post-image.jpg', 'image/jpeg', false);
      await GameroApi.savePostImage(this.selectedPost.id, key);
      this.selectedPost = await GameroApi.getPost(this.selectedPost.id);
      this.successText = '帖子图片已上传。';
    } catch {
      this.errorText = '上传帖子图片失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async uploadPostVideoToSelectedPost(): Promise<void> {
    if (!this.selectedPost) {
      this.errorText = '请先进入帖子详情。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const key = await GameroApi.pickAndUploadMedia('post-video', this.selectedPost.id, 'post-video.mp4', 'video/mp4', true);
      await GameroApi.savePostVideo(this.selectedPost.id, key);
      this.selectedPost = await GameroApi.getPost(this.selectedPost.id);
      this.successText = '帖子视频已上传。';
    } catch {
      this.errorText = '上传帖子视频失败。';
    } finally {
      this.isLoading = false;
    }
  }

  startEditPost(): void {
    if (!this.selectedPost) {
      return;
    }
    this.postTitle = this.selectedPost.title;
    this.postContent = this.selectedPost.content || this.selectedPost.content_snippet || '';
    this.isEditingPost = true;
  }

  cancelEditPost(): void {
    this.postTitle = '';
    this.postContent = '';
    this.isEditingPost = false;
  }

  async savePostEdit(): Promise<void> {
    if (!this.selectedPost || this.postTitle.trim().length === 0 || this.postContent.trim().length === 0) {
      this.errorText = '请输入帖子标题和内容。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      this.selectedPost = await GameroApi.updatePost(this.selectedPost.id, this.postTitle.trim(), this.postContent.trim());
      await this.loadPosts();
      this.cancelEditPost();
      this.successText = '帖子已更新。';
    } catch {
      this.errorText = '更新帖子失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async deleteSelectedPost(): Promise<void> {
    if (!this.selectedPost) {
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.deletePost(this.selectedPost.id);
      this.selectedPost = null;
      this.comments = [];
      this.featureView = 'posts';
      await this.loadPosts();
      this.successText = '帖子已删除。';
    } catch {
      this.errorText = '删除帖子失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async openPostDetail(id: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    try {
      this.selectedPost = await GameroApi.getPost(id);
      const page = await GameroApi.listPostComments(id);
      this.comments = page.list;
      this.commentInput = '';
      this.featureView = 'postDetail';
    } catch {
      this.errorText = '帖子详情加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async likeSelectedPost(): Promise<void> {
    if (!this.selectedPost) {
      return;
    }
    try {
      await GameroApi.likePost(this.selectedPost.id);
      this.selectedPost.like_count = this.selectedPost.like_count + 1;
      this.successText = '已点赞帖子。';
    } catch {
      this.errorText = '点赞失败。';
    }
  }

  async submitPostComment(): Promise<void> {
    if (!this.selectedPost || this.commentInput.trim().length === 0) {
      this.errorText = '请输入评论内容。';
      return;
    }
    this.isLoading = true;
    try {
      await GameroApi.createPostComment(this.selectedPost.id, this.commentInput.trim());
      const page = await GameroApi.listPostComments(this.selectedPost.id);
      this.comments = page.list;
      this.commentInput = '';
      this.successText = '评论已发布。';
    } catch {
      this.errorText = '发布评论失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadDevLogs(): Promise<void> {
    this.isLoading = true;
    try {
      const page = await GameroApi.listDevLogs();
      this.devLogs = page.list;
    } catch {
      this.devLogs = [];
      this.errorText = '开发日志加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async createDevLogForSelectedProject(): Promise<void> {
    if (!this.selectedProject) {
      this.errorText = '请先进入一个项目详情。';
      return;
    }
    if (this.devLogTitle.trim().length === 0 || this.devLogContent.trim().length === 0) {
      this.errorText = '请输入日志标题和内容。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.createDevLog(this.selectedProject.id, this.devLogTitle.trim(), this.devLogContent.trim());
      this.devLogTitle = '';
      this.devLogContent = '';
      await this.loadDevLogs();
      this.successText = '开发日志已发布。';
    } catch {
      this.errorText = '发布开发日志失败。';
    } finally {
      this.isLoading = false;
    }
  }

  startEditDevLog(): void {
    if (!this.selectedDevLog) {
      return;
    }
    this.devLogTitle = this.selectedDevLog.title;
    this.devLogContent = this.selectedDevLog.content || '';
    this.isEditingDevLog = true;
  }

  cancelEditDevLog(): void {
    this.devLogTitle = '';
    this.devLogContent = '';
    this.isEditingDevLog = false;
  }

  async saveDevLogEdit(): Promise<void> {
    if (!this.selectedDevLog || this.devLogTitle.trim().length === 0 || this.devLogContent.trim().length === 0) {
      this.errorText = '请输入日志标题和内容。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      this.selectedDevLog = await GameroApi.updateDevLog(this.selectedDevLog.id, this.devLogTitle.trim(), this.devLogContent.trim());
      await this.loadDevLogs();
      this.cancelEditDevLog();
      this.successText = '开发日志已更新。';
    } catch {
      this.errorText = '更新开发日志失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async deleteSelectedDevLog(): Promise<void> {
    if (!this.selectedDevLog) {
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.deleteDevLog(this.selectedDevLog.id);
      this.selectedDevLog = null;
      this.comments = [];
      this.featureView = 'logs';
      await this.loadDevLogs();
      this.successText = '开发日志已删除。';
    } catch {
      this.errorText = '删除开发日志失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async uploadDevLogImage(): Promise<void> {
    if (!this.selectedDevLog) {
      this.errorText = '请先进入日志详情。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const key = await GameroApi.pickAndUploadMedia('log-image', this.selectedDevLog.id, 'log-image.jpg', 'image/jpeg', false);
      await GameroApi.saveLogImage(this.selectedDevLog.id, key);
      this.selectedDevLog = await GameroApi.getDevLog(this.selectedDevLog.id);
      this.successText = '日志图片已上传。';
    } catch {
      this.errorText = '上传日志图片失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async uploadDevLogVideo(): Promise<void> {
    if (!this.selectedDevLog) {
      this.errorText = '请先进入日志详情。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const key = await GameroApi.pickAndUploadMedia('log-video', this.selectedDevLog.id, 'log-video.mp4', 'video/mp4', true);
      await GameroApi.saveLogVideo(this.selectedDevLog.id, key);
      this.selectedDevLog = await GameroApi.getDevLog(this.selectedDevLog.id);
      this.successText = '日志视频已上传。';
    } catch {
      this.errorText = '上传日志视频失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async openDevLogDetail(id: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    try {
      this.selectedDevLog = await GameroApi.getDevLog(id);
      const page = await GameroApi.listDevLogComments(id);
      this.comments = page.list;
      this.commentInput = '';
      this.featureView = 'logDetail';
    } catch {
      this.errorText = '开发日志详情加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async likeSelectedDevLog(): Promise<void> {
    if (!this.selectedDevLog) {
      return;
    }
    try {
      await GameroApi.likeDevLog(this.selectedDevLog.id);
      this.selectedDevLog.like_count = this.selectedDevLog.like_count + 1;
      this.successText = '已点赞开发日志。';
    } catch {
      this.errorText = '点赞失败。';
    }
  }

  async submitDevLogComment(): Promise<void> {
    if (!this.selectedDevLog || this.commentInput.trim().length === 0) {
      this.errorText = '请输入评论内容。';
      return;
    }
    this.isLoading = true;
    try {
      await GameroApi.createDevLogComment(this.selectedDevLog.id, this.commentInput.trim());
      const page = await GameroApi.listDevLogComments(this.selectedDevLog.id);
      this.comments = page.list;
      this.commentInput = '';
      this.successText = '评论已发布。';
    } catch {
      this.errorText = '发布评论失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadNotifications(): Promise<void> {
    this.isLoading = true;
    try {
      const page = await GameroApi.listNotifications();
      this.notifications = page.list;
      const count = await GameroApi.getUnreadCount();
      this.unreadCount = count.unread_count;
    } catch {
      this.notifications = [];
      this.unreadCount = 0;
      this.errorText = '通知加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadUnreadCount(): Promise<void> {
    try {
      const count = await GameroApi.getUnreadCount();
      this.unreadCount = count.unread_count;
    } catch {
      this.unreadCount = 0;
    }
  }

  async markNotificationRead(id: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.markNotificationRead(id);
      await this.loadNotifications();
      this.successText = '通知已标记为已读。';
    } catch {
      this.errorText = '标记已读失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async markAllNotificationsRead(): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.markAllNotificationsRead();
      await this.loadNotifications();
      this.successText = '全部通知已读。';
    } catch {
      this.errorText = '全部已读失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async deleteNotification(id: number): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    try {
      await GameroApi.deleteNotification(id);
      await this.loadNotifications();
      this.successText = '通知已删除。';
    } catch {
      this.errorText = '删除通知失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async runSearch(): Promise<void> {
    if (this.searchKeyword.trim().length === 0) {
      this.errorText = '请输入搜索关键词。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    try {
      const page = await GameroApi.search(this.searchKeyword.trim());
      this.searchResults = page.list;
    } catch {
      this.searchResults = [];
      this.errorText = '搜索失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loginAndLoadProtectedData(): Promise<void> {
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    try {
      const auth = await GameroApi.login(this.loginEmail, this.loginPassword);
      this.currentUser = auth.user;
      this.successText = '登录成功，已连接真实后端。';
      await this.refreshHomeData(false);
      this.currentTab = 0;
    } catch {
      this.errorText = '登录或受保护数据加载失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async sendRegisterCode(): Promise<void> {
    if (this.registerEmail.trim().length === 0) {
      this.errorText = '请输入注册邮箱。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    try {
      await GameroApi.sendRegisterCode(this.registerEmail.trim());
      this.successText = '验证码已发送，请查看邮箱或后端开发环境输出。';
    } catch {
      this.errorText = '验证码发送失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async registerAndLogin(): Promise<void> {
    if (this.registerEmail.trim().length === 0 || this.registerCode.trim().length === 0 || this.registerPassword.length === 0) {
      this.errorText = '请填写邮箱、验证码和密码。';
      return;
    }
    this.isLoading = true;
    this.errorText = '';
    this.successText = '';
    try {
      const auth = await GameroApi.register(
        this.registerEmail.trim(),
        this.registerCode.trim(),
        this.registerPassword,
        this.registerNickname.trim()
      );
      this.currentUser = auth.user;
      this.successText = '注册成功，已自动登录。';
      await this.refreshHomeData(false);
      this.currentTab = 0;
    } catch {
      this.errorText = '注册失败。';
    } finally {
      this.isLoading = false;
    }
  }

  async loadProtectedDataAfterAuth(): Promise<void> {
    await this.refreshHomeData(false);
  }

  async refreshHomeData(showLoading: boolean = true): Promise<void> {
    if (showLoading) {
      this.isLoading = true;
    }
    this.errorText = '';
    await this.loadPublicData();

    try {
      const feedPage = await GameroApi.getFeed();
      this.feeds = feedPage.items.map((item: FeedItem) => {
        let title = this.typeText(item.type);
        let desc = '暂无内容摘要';
        if (item.post) {
          title = item.post.title;
          desc = item.post.content_snippet || '暂无帖子摘要';
        } else if (item.devlog) {
          title = item.devlog.title;
          desc = `${this.typeText(item.devlog.log_type)} · 开发日志更新`;
        } else if (item.project) {
          title = item.project.name;
          desc = item.project.description || this.statusText(item.project.status);
        }
        let card: FeedCardModel = {
          author: item.author ? item.author.nickname : 'Gamero 用户',
          title,
          desc,
          meta: `${this.typeText(item.type)} · ${item.created_at ? item.created_at.slice(0, 10) : '刚刚'}`
        };
        return card;
      });
    } catch {
      this.feeds = [];
      this.errorText = '动态流加载失败，当前没有使用演示数据。';
    }

    if (this.currentUser && (this.currentUser.role === 'admin' || this.currentUser.role === 'superadmin')) {
      try {
        this.stats = await GameroApi.getAdminStats();
      } catch {}
    }

    try {
      const projectPage = await GameroApi.listProjects();
      if (projectPage.list.length > 0) {
        this.tasks = await GameroApi.listProjectTasks(projectPage.list[0].id);
      }
    } catch {}

    await this.loadMyApplications();
    await this.loadUnreadCount();
    this.homeLoadedOnce = true;
    if (showLoading) {
      this.isLoading = false;
    }
  }

  logout(): void {
    GameroApi.logout();
    this.currentUser = null;
    this.stats = null;
    this.tasks = [];
    this.posts = [];
    this.collectedPosts = [];
    this.devLogs = [];
    this.myDevLogs = [];
    this.collectedLogs = [];
    this.myReports = [];
    this.notifications = [];
    this.notificationPreferences = [];
    this.searchResults = [];
    this.collectedProjects = [];
    this.adminUsers = [];
    this.adminReports = [];
    this.sensitiveWords = [];
    this.unreadCount = 0;
    this.featureView = 'menu';
    this.successText = '已退出登录。';
    this.errorText = '';
    this.currentTab = 0;
  }

  statusText(status: string): string {
    switch (status) {
      case 'preparing': return '筹备中';
      case 'developing': return '开发中';
      case 'playable': return '可试玩';
      case 'launched': return '已上线';
      case 'paused': return '暂停';
      case 'abandoned': return '已弃坑';
      default: return status;
    }
  }

  genreText(genre: string): string {
    switch (genre) {
      case 'action': return '动作';
      case 'rpg': return 'RPG';
      case 'strategy': return '策略';
      case 'simulator': return '模拟';
      case 'puzzle': return '解谜';
      case 'horror': return '恐怖';
      case 'platform': return '平台';
      case 'other': return '其他';
      default: return genre;
    }
  }

  positionText(position: string): string {
    switch (position) {
      case 'program': return '程序';
      case 'art': return '美术';
      case 'design': return '策划';
      case 'sound': return '音效';
      default: return position;
    }
  }

  coopText(type: string): string {
    switch (type) {
      case 'online': return '线上合作';
      case 'offline': return '线下合作';
      case 'hybrid': return '混合模式';
      default: return type;
    }
  }

  typeText(type: string): string {
    switch (type) {
      case 'post': return '社区帖子';
      case 'devlog': return '开发日志';
      case 'project': return '项目动态';
      case 'release': return '版本发布';
      default: return type;
    }
  }

  reportTargetText(type: string): string {
    switch (type) {
      case 'post': return '帖子';
      case 'log': return '开发日志';
      case 'project': return '项目';
      case 'recruitment': return '招募';
      case 'comment': return '评论';
      case 'user': return '用户';
      case 'project_release': return '版本发布';
      default: return type;
    }
  }

  reportStatusText(status: string): string {
    switch (status) {
      case 'pending': return '待处理';
      case 'handled': return '已处理';
      case 'rejected': return '已驳回';
      case 'escalated': return '已升级';
      default: return status;
    }
  }

  daysLeft(iso: string): string {
    const end = new Date(iso).getTime();
    const diff = Math.ceil((end - Date.now()) / 86400000);
    return diff > 0 ? `${diff} 天` : '已过期';
  }

  tabIcon(index: number): string {
    switch (index) {
      case 0: return '动';
      case 1: return '项';
      case 2: return '招';
      case 3: return '能';
      default: return '我';
    }
  }

  statusColor(status: string): string {
    switch (status) {
      case 'developing': return GameroTheme.success;
      case 'playable': return '#64B5D8';
      case 'launched': return GameroTheme.amber;
      case 'paused': return GameroTheme.warning;
      case 'abandoned': return GameroTheme.danger;
      default: return GameroTheme.textSecondary;
    }
  }

  genreColor(genre: string): string {
    switch (genre) {
      case 'action': return GameroTheme.coral;
      case 'rpg': return '#C4A4E8';
      case 'strategy': return GameroTheme.success;
      case 'simulator': return GameroTheme.warning;
      case 'puzzle': return '#64B5D8';
      case 'horror': return '#8B6B7A';
      case 'platform': return '#F0B070';
      default: return '#8888A0';
    }
  }

  @Builder
  LogoHeader() {
    Row() {
      Stack() {
        Rect().width(36).height(36).fill(GameroTheme.amber).radius(8)
        Text('G').fontColor('#16161D').fontSize(20).fontWeight(FontWeight.Bold)
      }.width(40).height(40)

      Column() {
        Text('GAMERO').fontColor(GameroTheme.textPrimary).fontSize(19).fontWeight(FontWeight.Bold)
        Text(this.currentUser ? '游戏共创社区' : 'HarmonyOS 原生端').fontColor(GameroTheme.textMuted).fontSize(11)
      }.alignItems(HorizontalAlign.Start).margin({ left: 10 })

      Blank()
      Text(this.currentUser ? this.currentUser.role || 'user' : 'Harmony').fontColor(GameroTheme.amber).fontSize(11).fontWeight(FontWeight.Bold).padding({ left: 10, right: 10, top: 6, bottom: 6 })
        .backgroundColor('#2C2418').borderRadius(8)
    }
    .width('100%')
    .padding({ left: 18, right: 18, top: 14, bottom: 12 })
    .backgroundColor('#F21A1A22')
    .border({ width: { bottom: 1 }, color: GameroTheme.borderSubtle })
  }

  @Builder
  StatPill(label: string, value: string) {
    Column() {
      Text(value).fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold)
      Text(label).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 3 })
    }
    .padding({ top: 12, bottom: 12 })
    .layoutWeight(1)
    .backgroundColor('#2226313A')
    .borderRadius(8)
    .border({ width: 1, color: GameroTheme.borderSubtle })
  }

  @Builder
  Hero() {
    Column() {
      Row() {
        Text('LIVE STUDIO')
          .fontColor(GameroTheme.amber)
          .fontSize(10)
          .fontWeight(FontWeight.Bold)
          .padding({ left: 9, right: 9, top: 5, bottom: 5 })
          .backgroundColor('#2C2418')
          .borderRadius(6)
        Blank()
        Text(this.currentUser ? this.currentUser.nickname || this.currentUser.username : '未登录')
          .fontColor(GameroTheme.textSecondary)
          .fontSize(11)
      }.width('100%').margin({ bottom: 14 })
      Text('从灵感到组队上线')
        .fontColor(GameroTheme.textPrimary)
        .fontSize(27)
        .fontWeight(FontWeight.Bold)
        .lineHeight(34)
      Text('项目展示、开发日志、社区交流、人才招募、任务看板与后台治理一体化。')
        .fontColor(GameroTheme.textSecondary)
        .fontSize(14)
        .lineHeight(22)
        .margin({ top: 10 })
      Row({ space: 10 }) {
        this.StatPill('项目', this.stats ? `${this.stats.total_projects}` : `${this.projects.length}`)
        this.StatPill('动态', `${this.feeds.length}`)
        this.StatPill('招募', `${this.recruits.length}`)
      }.width('100%').margin({ top: 18 })
      Button(this.homeLoadedOnce ? '刷新首页数据' : '加载首页数据')
        .width('100%')
        .height(40)
        .fontColor('#16161D')
        .fontSize(13)
        .fontWeight(FontWeight.Bold)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(8)
        .margin({ top: 14 })
        .onClick(() => this.refreshHomeData(true))
      if (this.errorText.length > 0) {
        Text(this.errorText).fontColor(GameroTheme.coral).fontSize(11).lineHeight(16).margin({ top: 12 })
      }
    }
    .width('100%')
    .padding(18)
    .backgroundColor('#23232D')
    .borderRadius(8)
    .border({ width: 1, color: GameroTheme.borderVisible })
  }

  @Builder
  FeedCard(item: FeedCardModel) {
    Column() {
      Row() {
        Text(item.author.slice(0, 1)).fontColor('#16161D').fontSize(14).fontWeight(FontWeight.Bold)
          .width(34).height(34).textAlign(TextAlign.Center).backgroundColor(GameroTheme.amber).borderRadius(8)
        Column() {
          Text(item.title).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold)
          Text(item.meta).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 3 })
        }.alignItems(HorizontalAlign.Start).margin({ left: 10 }).layoutWeight(1)
      }
      Text(item.desc).fontColor(GameroTheme.textSecondary).fontSize(13).lineHeight(20).margin({ top: 12 })
    }.cardStyle()
  }

  @Builder
  ProjectCard(item: ProjectCardModel) {
    Column() {
      Row() {
        Column() {
          Text(item.title).fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
          Text(item.genre).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
        }.alignItems(HorizontalAlign.Start).layoutWeight(1)
        Text(item.status).fontColor(GameroTheme.amber).fontSize(11).padding({ left: 9, right: 9, top: 5, bottom: 5 })
          .backgroundColor('#241A0B').borderRadius(10)
      }
      Text(item.desc).fontColor(GameroTheme.textSecondary).fontSize(13).lineHeight(20).margin({ top: 12 })
      Text(item.followers).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 12 })
    }.cardStyle()
  }

  @Builder
  ProjectItemCard(item: ProjectItem) {
    Column() {
      Row() {
        Rect().width(4).height(42).fill(this.genreColor(item.genre)).radius(2).margin({ right: 10 })
        Column() {
          Text(item.name).fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
          Text(`${this.genreText(item.genre)} · ${item.member_count || 0} 成员`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
        }.alignItems(HorizontalAlign.Start).layoutWeight(1)
        Text(this.statusText(item.status)).fontColor(this.statusColor(item.status)).fontSize(11).fontWeight(FontWeight.Bold).padding({ left: 9, right: 9, top: 5, bottom: 5 })
          .backgroundColor('#262A34').borderRadius(6)
      }
      Text(item.description || '暂无项目简介').fontColor(GameroTheme.textSecondary).fontSize(13).lineHeight(20).margin({ top: 12 })
      Row({ space: 8 }) {
        this.MiniMetric('关注', `${item.follower_count || 0}`)
        this.MiniMetric('浏览', `${item.view_count || 0}`)
        this.MiniMetric('收藏', `${item.collect_count || 0}`)
      }.width('100%').margin({ top: 12 })
    }.cardStyle().onClick(() => this.openProjectDetail(item.id))
  }

  @Builder
  MiniMetric(label: string, value: string) {
    Text(`${value} ${label}`)
      .fontColor(GameroTheme.textMuted)
      .fontSize(11)
      .padding({ left: 8, right: 8, top: 5, bottom: 5 })
      .backgroundColor('#20202A')
      .borderRadius(6)
  }

  @Builder
  SectionHeader(title: string, action: string, view: string) {
    Row() {
      Text(title).fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold)
      Blank()
      Text(action)
        .fontColor(GameroTheme.amber)
        .fontSize(11)
        .fontWeight(FontWeight.Bold)
        .padding({ left: 9, right: 9, top: 6, bottom: 6 })
        .backgroundColor('#2C2418')
        .borderRadius(6)
        .onClick(() => this.openFeature(view))
    }.width('100%').margin({ top: 16, bottom: 10 })
  }

  @Builder
  QuickAction(title: string, desc: string, view: string) {
    Column() {
      Text(title).fontColor(GameroTheme.textPrimary).fontSize(14).fontWeight(FontWeight.Bold)
      Text(desc).fontColor(GameroTheme.textMuted).fontSize(11).lineHeight(15).margin({ top: 5 })
    }
    .layoutWeight(1)
    .height(74)
    .padding(10)
    .alignItems(HorizontalAlign.Start)
    .justifyContent(FlexAlign.Center)
    .backgroundColor('#20202A')
    .borderRadius(8)
    .border({ width: 1, color: GameroTheme.borderSubtle })
    .onClick(() => this.openFeature(view))
  }

  @Builder
  QuickWorkspace() {
    Column() {
      Row() {
        Text('快捷工作区').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
        Blank()
        Text(`${this.unreadCount} 未读`).fontColor(this.unreadCount > 0 ? GameroTheme.amber : GameroTheme.textMuted).fontSize(11)
      }.width('100%').margin({ bottom: 10 })
      Row({ space: 8 }) {
        this.QuickAction('社区', '帖子与评论', 'posts')
        this.QuickAction('日志', '开发进展', 'logs')
      }.width('100%')
      Row({ space: 8 }) {
        this.QuickAction('搜索', '跨内容发现', 'search')
        this.QuickAction('人才', '找协作者', 'talents')
      }.width('100%').margin({ top: 8 })
      Row({ space: 8 }) {
        this.QuickAction('通知', '消息中心', 'notifications')
        this.QuickAction('收藏', '我的沉淀', 'collections')
      }.width('100%').margin({ top: 8 })
    }.cardStyle()
  }

  @Builder
  HomeProjectPreview() {
    Column() {
      Row() {
        Text('最近项目').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
        Blank()
        Text('全部项目').fontColor(GameroTheme.amber).fontSize(11).fontWeight(FontWeight.Bold).onClick(() => this.selectTab(1))
      }.width('100%').margin({ bottom: 10 })
      if (this.projectItems.length > 0) {
        ForEach(this.projectItems.slice(0, 3), (item: ProjectItem) => {
          Row() {
            Rect().width(4).height(34).fill(this.genreColor(item.genre)).radius(2).margin({ right: 10 })
            Column() {
              Text(item.name).fontColor(GameroTheme.textPrimary).fontSize(14).fontWeight(FontWeight.Bold)
              Text(`${this.genreText(item.genre)} · ${this.statusText(item.status)} · ${item.follower_count || 0} 关注`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
            }.alignItems(HorizontalAlign.Start).layoutWeight(1)
            Text('详情').fontColor(GameroTheme.amber).fontSize(11)
          }
          .width('100%')
          .padding({ top: 10, bottom: 10 })
          .border({ width: { bottom: 1 }, color: GameroTheme.borderSubtle })
          .onClick(() => this.openProjectDetail(item.id))
        })
      } else {
        Text('暂无项目数据').fontColor(GameroTheme.textMuted).fontSize(12)
      }
    }.cardStyle()
  }

  @Builder
  HomeRecruitPreview() {
    Column() {
      Row() {
        Text('正在招募').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
        Blank()
        Text('全部招募').fontColor(GameroTheme.amber).fontSize(11).fontWeight(FontWeight.Bold).onClick(() => this.selectTab(2))
      }.width('100%').margin({ bottom: 10 })
      if (this.recruitItems.length > 0) {
        ForEach(this.recruitItems.slice(0, 3), (item: RecruitItem) => {
          Row() {
            Column() {
              Text(item.project_name || `项目 #${item.project_id}`).fontColor(GameroTheme.textPrimary).fontSize(14).fontWeight(FontWeight.Bold)
              Text(`${this.positionText(item.position)} · ${this.coopText(item.cooperation_type)} · ${this.daysLeft(item.expire_at)}`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
            }.alignItems(HorizontalAlign.Start).layoutWeight(1)
            Text(item.status === 'closed' ? '已关闭' : '申请').fontColor(item.status === 'closed' ? GameroTheme.textMuted : '#16161D').fontSize(11).fontWeight(FontWeight.Bold).padding({ left: 9, right: 9, top: 5, bottom: 5 }).backgroundColor(item.status === 'closed' ? '#262A34' : GameroTheme.amber).borderRadius(6)
          }
          .width('100%')
          .padding({ top: 10, bottom: 10 })
          .border({ width: { bottom: 1 }, color: GameroTheme.borderSubtle })
          .onClick(() => this.openRecruitDetail(item.id))
        })
      } else {
        Text('暂无招募数据').fontColor(GameroTheme.textMuted).fontSize(12)
      }
    }.cardStyle()
  }

  @Builder
  ProjectComposer() {
    Column() {
      Text('创建项目').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '项目名称', text: this.projectName })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.projectName = value)
      TextInput({ placeholder: '项目简介', text: this.projectDescription })
        .height(56)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.projectDescription = value)
      Row({ space: 8 }) {
        Button('动作').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectGenre === 'action' ? GameroTheme.amber : '#202634').fontColor(this.projectGenre === 'action' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectGenre = 'action')
        Button('RPG').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectGenre === 'rpg' ? GameroTheme.amber : '#202634').fontColor(this.projectGenre === 'rpg' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectGenre = 'rpg')
        Button('其他').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectGenre === 'other' ? GameroTheme.amber : '#202634').fontColor(this.projectGenre === 'other' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectGenre = 'other')
      }.width('100%').margin({ top: 10 })
      Row({ space: 8 }) {
        Button('筹备').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectStatus === 'preparing' ? GameroTheme.amber : '#202634').fontColor(this.projectStatus === 'preparing' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectStatus = 'preparing')
        Button('开发中').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectStatus === 'developing' ? GameroTheme.amber : '#202634').fontColor(this.projectStatus === 'developing' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectStatus = 'developing')
        Button('可试玩').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectStatus === 'playable' ? GameroTheme.amber : '#202634').fontColor(this.projectStatus === 'playable' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectStatus = 'playable')
      }.width('100%').margin({ top: 10 })
      Button('创建项目')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => this.createProjectFromForm())
    }.cardStyle()
  }

  @Builder
  ProjectDetailPage() {
    Column() {
      Button('返回上一页')
        .fontColor(GameroTheme.amber)
        .backgroundColor('#241A0B')
        .borderRadius(14)
        .margin({ bottom: 12 })
        .onClick(() => this.closeProjectDetail())
      this.MessagePanel()
      if (this.selectedProject) {
        Column() {
          Text(this.selectedProject.name).fontColor(GameroTheme.textPrimary).fontSize(24).fontWeight(FontWeight.Bold)
          Text(`${this.genreText(this.selectedProject.genre)} · ${this.statusText(this.selectedProject.status)} · ${this.selectedProject.visibility || 'public'}`)
            .fontColor(GameroTheme.textMuted)
            .fontSize(12)
            .margin({ top: 8 })
          Text(this.selectedProject.description || '暂无项目简介')
            .fontColor(GameroTheme.textSecondary)
            .fontSize(14)
            .lineHeight(22)
            .margin({ top: 14 })
          Row({ space: 8 }) {
            this.StatPill('关注', `${this.selectedProject.follower_count || 0}`)
            this.StatPill('浏览', `${this.selectedProject.view_count || 0}`)
            this.StatPill('收藏', `${this.selectedProject.collect_count || 0}`)
          }.width('100%').margin({ top: 14 })
          Row({ space: 8 }) {
            Button(this.selectedProject.is_followed ? '取消关注' : '关注项目')
              .layoutWeight(1)
              .height(42)
              .fontColor(GameroTheme.voidBg)
              .backgroundColor(GameroTheme.amber)
              .borderRadius(14)
              .onClick(() => this.toggleProjectFollow())
            Button(this.selectedProject.is_collected ? '取消收藏' : '收藏项目')
              .layoutWeight(1)
              .height(42)
              .fontColor(GameroTheme.amber)
              .backgroundColor('#241A0B')
              .borderRadius(14)
              .onClick(() => this.toggleProjectCollect())
          }.width('100%').margin({ top: 14 })
          Row({ space: 8 }) {
            Button(this.isEditingProject ? '取消编辑' : '编辑项目')
              .layoutWeight(1)
              .height(42)
              .fontColor(GameroTheme.amber)
              .backgroundColor('#241A0B')
              .borderRadius(14)
              .onClick(() => {
                if (this.isEditingProject) {
                  this.cancelEditProject()
                } else {
                  this.startEditProject()
                }
              })
            Button('删除项目')
              .layoutWeight(1)
              .height(42)
              .fontColor(GameroTheme.textPrimary)
              .backgroundColor('#2A1212')
              .borderRadius(14)
              .onClick(() => this.deleteSelectedProject())
          }.width('100%').margin({ top: 10 })
        }.cardStyle()

        if (this.isEditingProject) {
          this.ProjectEditForm()
        }

        this.ProjectMediaPanel()

        this.ProjectMembersPanel()

        this.ReleasePanel()
        this.DevLogComposer()
        this.TaskComposer()
        this.TaskBoard()
      } else {
        this.EmptyCard('项目详情为空')
      }
    }.width('100%')
  }

  @Builder
  ReleasePanel() {
    Column() {
      Text('项目版本').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '版本号，例如 v1.0.0', text: this.releaseVersion })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.releaseVersion = value)
      TextInput({ placeholder: '版本标题', text: this.releaseTitle })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.releaseTitle = value)
      TextInput({ placeholder: '更新说明', text: this.releaseContent })
        .height(56)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.releaseContent = value)
      Button('发布版本')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => this.createReleaseForSelectedProject())
      if (this.releases.length > 0) {
        ForEach(this.releases, (release: DevLogItem) => {
          Column() {
            Text(release.title).fontColor(GameroTheme.textPrimary).fontSize(13).fontWeight(FontWeight.Bold)
            Text(`${release.created_at ? release.created_at.slice(0, 10) : '刚刚'} · ${release.comment_count} 评论`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
            Text(release.content || '暂无更新说明').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
          }.padding(10).margin({ top: 10 }).backgroundColor('#202634').borderRadius(12)
        })
      } else {
        Text('暂无版本记录').fontColor(GameroTheme.textSecondary).fontSize(12).margin({ top: 10 })
      }
    }.cardStyle()
  }

  @Builder
  ProjectMediaPanel() {
    Column() {
      Text('项目媒体').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold)
      Text('选择本地图片/视频后，先获取上传凭证并直传对象存储，再把 key 绑定到项目。')
        .fontColor(GameroTheme.textMuted)
        .fontSize(11)
        .lineHeight(16)
        .margin({ top: 6 })
      Row({ space: 8 }) {
        Button('上传封面').layoutWeight(1).height(36).fontSize(11).fontColor(GameroTheme.voidBg).backgroundColor(GameroTheme.amber).borderRadius(12).onClick(() => this.uploadProjectCover())
        Button('上传截图').layoutWeight(1).height(36).fontSize(11).fontColor(GameroTheme.amber).backgroundColor('#241A0B').borderRadius(12).onClick(() => this.uploadProjectScreenshot())
      }.width('100%').margin({ top: 10 })
      Button('上传项目视频')
        .width('100%')
        .height(36)
        .fontSize(11)
        .fontColor(GameroTheme.textPrimary)
        .backgroundColor('#202634')
        .borderRadius(12)
        .margin({ top: 8 })
        .onClick(() => this.uploadProjectVideo())
      if (this.selectedProject && this.selectedProject.cover_url && this.selectedProject.cover_url.length > 0) {
        Text('封面已设置').fontColor(GameroTheme.success).fontSize(11).margin({ top: 10 })
      }
      if (this.selectedProject && this.selectedProject.screenshot_urls && this.selectedProject.screenshot_urls.length > 0) {
        Text(`截图 ${this.selectedProject.screenshot_urls.length} 张`).fontColor(GameroTheme.textSecondary).fontSize(11).margin({ top: 6 })
      }
      if (this.selectedProject && this.selectedProject.video_url && this.selectedProject.video_url.length > 0) {
        Text('项目视频已设置').fontColor(GameroTheme.textSecondary).fontSize(11).margin({ top: 6 })
      }
    }.cardStyle()
  }

  @Builder
  ProjectMembersPanel() {
    Column() {
      Text('项目成员').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '要添加的用户 ID', text: this.memberUserIdText })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.memberUserIdText = value)
      Row({ space: 8 }) {
        Button('成员').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.memberRole === 'member' ? GameroTheme.amber : '#202634').fontColor(this.memberRole === 'member' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.memberRole = 'member')
        Button('管理员').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.memberRole === 'admin' ? GameroTheme.amber : '#202634').fontColor(this.memberRole === 'admin' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.memberRole = 'admin')
      }.width('100%').margin({ top: 10 })
      TextInput({ placeholder: '贡献说明，可选', text: this.memberContribution })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.memberContribution = value)
      Button('添加成员')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => this.addMemberToSelectedProject())
      Button('退出项目')
        .width('100%')
        .height(40)
        .fontColor(GameroTheme.textPrimary)
        .backgroundColor('#2A1212')
        .borderRadius(12)
        .margin({ top: 10 })
        .onClick(() => this.leaveSelectedProject())
      if (this.selectedProject && this.selectedProject.members && this.selectedProject.members.length > 0) {
        ForEach(this.selectedProject.members, (member: ProjectMemberItem) => {
          Column() {
            Row() {
              Text(member.nickname || member.username || `用户 #${member.user_id}`).fontColor(GameroTheme.textPrimary).fontSize(13).layoutWeight(1)
              Text(member.role).fontColor(GameroTheme.amber).fontSize(11)
            }
            Row({ space: 8 }) {
              Button('设为成员').layoutWeight(1).height(32).fontSize(11).backgroundColor('#202634').fontColor(GameroTheme.textPrimary).borderRadius(10).onClick(() => this.updateMemberRole(member.user_id, 'member'))
              Button('设为管理员').layoutWeight(1).height(32).fontSize(11).backgroundColor('#241A0B').fontColor(GameroTheme.amber).borderRadius(10).onClick(() => this.updateMemberRole(member.user_id, 'admin'))
              Button('移除').layoutWeight(1).height(32).fontSize(11).backgroundColor('#2A1212').fontColor(GameroTheme.textPrimary).borderRadius(10).onClick(() => this.removeMemberFromSelectedProject(member.user_id))
            }.width('100%').margin({ top: 8 })
          }.padding(10).margin({ top: 8 }).backgroundColor('#202634').borderRadius(12)
        })
      } else {
        Text('暂无成员数据').fontColor(GameroTheme.textSecondary).fontSize(12).margin({ top: 10 })
      }
    }.cardStyle()
  }

  @Builder
  ProjectEditForm() {
    Column() {
      Text('编辑项目').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '项目名称', text: this.projectName })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.projectName = value)
      TextInput({ placeholder: '项目简介', text: this.projectDescription })
        .height(56)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.projectDescription = value)
      Row({ space: 8 }) {
        Button('动作').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectGenre === 'action' ? GameroTheme.amber : '#202634').fontColor(this.projectGenre === 'action' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectGenre = 'action')
        Button('RPG').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectGenre === 'rpg' ? GameroTheme.amber : '#202634').fontColor(this.projectGenre === 'rpg' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectGenre = 'rpg')
        Button('其他').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectGenre === 'other' ? GameroTheme.amber : '#202634').fontColor(this.projectGenre === 'other' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectGenre = 'other')
      }.width('100%').margin({ top: 10 })
      Row({ space: 8 }) {
        Button('筹备').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectStatus === 'preparing' ? GameroTheme.amber : '#202634').fontColor(this.projectStatus === 'preparing' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectStatus = 'preparing')
        Button('开发中').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectStatus === 'developing' ? GameroTheme.amber : '#202634').fontColor(this.projectStatus === 'developing' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectStatus = 'developing')
        Button('可试玩').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.projectStatus === 'playable' ? GameroTheme.amber : '#202634').fontColor(this.projectStatus === 'playable' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.projectStatus = 'playable')
      }.width('100%').margin({ top: 10 })
      Button('保存修改')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => this.saveProjectEdit())
    }.cardStyle()
  }

  @Builder
  DevLogComposer() {
    Column() {
      Text('发布开发日志').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '日志标题', text: this.devLogTitle })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.devLogTitle = value)
      TextInput({ placeholder: '日志内容', text: this.devLogContent })
        .height(56)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.devLogContent = value)
      Button('发布日志')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => this.createDevLogForSelectedProject())
    }.cardStyle()
  }

  @Builder
  RecruitCard(item: RecruitCardModel) {
    Column() {
      Row() {
        Column() {
          Text(item.project).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold)
          Text(item.role).fontColor(GameroTheme.amber).fontSize(13).margin({ top: 5 })
          Text(`${item.mode} · ${item.days}`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 6 })
        }.alignItems(HorizontalAlign.Start).layoutWeight(1)
        Text('申请').fontColor(GameroTheme.voidBg).fontSize(12).fontWeight(FontWeight.Bold).padding({ left: 14, right: 14, top: 8, bottom: 8 })
          .backgroundColor(GameroTheme.amber).borderRadius(14)
      }
    }.cardStyle()
  }

  @Builder
  RecruitItemCard(item: RecruitItem) {
    Column() {
      Row() {
        Stack() {
          Rect().width(38).height(38).fill('#2C2418').radius(8)
          Text(this.positionText(item.position).slice(0, 1)).fontColor(GameroTheme.amber).fontSize(15).fontWeight(FontWeight.Bold)
        }.width(42).height(42).margin({ right: 10 })
        Column() {
          Text(item.project_name || `项目 #${item.project_id}`).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold)
          Text(this.positionText(item.position)).fontColor(GameroTheme.amber).fontSize(13).margin({ top: 5 })
          Text(`${this.coopText(item.cooperation_type)} · ${item.headcount} 人 · ${this.daysLeft(item.expire_at)}`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 6 })
        }.alignItems(HorizontalAlign.Start).layoutWeight(1)
        Text(item.status === 'closed' ? '已关闭' : '招募中').fontColor(item.status === 'closed' ? GameroTheme.textMuted : '#16161D').fontSize(11).fontWeight(FontWeight.Bold).padding({ left: 10, right: 10, top: 6, bottom: 6 })
          .backgroundColor(item.status === 'closed' ? '#262A34' : GameroTheme.amber).borderRadius(6)
      }
      Text(item.description || '暂无岗位说明').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 10 })
    }.cardStyle().onClick(() => this.openRecruitDetail(item.id))
  }

  @Builder
  RecruitDetailPage() {
    Column() {
      Button('返回上一页')
        .fontColor(GameroTheme.amber)
        .backgroundColor('#241A0B')
        .borderRadius(14)
        .margin({ bottom: 12 })
        .onClick(() => this.closeRecruitDetail())
      this.MessagePanel()
      if (this.selectedRecruit) {
        Column() {
          Text(this.selectedRecruit.project_name || `项目 #${this.selectedRecruit.project_id}`).fontColor(GameroTheme.textPrimary).fontSize(22).fontWeight(FontWeight.Bold)
          Text(`${this.positionText(this.selectedRecruit.position)} · ${this.coopText(this.selectedRecruit.cooperation_type)} · ${this.selectedRecruit.headcount} 人`)
            .fontColor(GameroTheme.amber)
            .fontSize(13)
            .margin({ top: 8 })
          Text(this.selectedRecruit.description || '暂无岗位说明')
            .fontColor(GameroTheme.textSecondary)
            .fontSize(14)
            .lineHeight(22)
            .margin({ top: 14 })
          Text(`截止：${this.selectedRecruit.expire_at ? this.selectedRecruit.expire_at.slice(0, 10) : '未设置'} · 状态：${this.selectedRecruit.status || 'open'}`)
            .fontColor(GameroTheme.textMuted)
            .fontSize(11)
            .margin({ top: 12 })
        }.cardStyle()

        Row({ space: 8 }) {
          Button(this.isEditingRecruit ? '取消编辑' : '编辑招募')
            .layoutWeight(1)
            .height(40)
            .fontColor(GameroTheme.amber)
            .backgroundColor('#241A0B')
            .borderRadius(12)
            .onClick(() => {
              if (this.isEditingRecruit) {
                this.cancelEditRecruit()
              } else {
                this.startEditRecruit()
              }
            })
          Button(this.selectedRecruit.status === 'closed' ? '重开' : '关闭')
            .layoutWeight(1)
            .height(40)
            .fontColor(GameroTheme.textPrimary)
            .backgroundColor('#202634')
            .borderRadius(12)
            .onClick(() => {
              if (this.selectedRecruit && this.selectedRecruit.status === 'closed') {
                this.reopenSelectedRecruit()
              } else {
                this.closeSelectedRecruit()
              }
            })
          Button('删除')
            .layoutWeight(1)
            .height(40)
            .fontColor(GameroTheme.textPrimary)
            .backgroundColor('#2A1212')
            .borderRadius(12)
            .onClick(() => this.deleteSelectedRecruit())
        }.width('100%').margin({ bottom: 12 })

        if (this.isEditingRecruit) {
          this.RecruitmentForm('保存招募', true)
        }

        Column() {
          Text('申请这个岗位').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
          TextInput({ placeholder: '申请说明，介绍你的技能和合作时间', text: this.applyMessage })
            .height(46)
            .fontColor(GameroTheme.textPrimary)
            .placeholderColor(GameroTheme.textMuted)
            .backgroundColor('#202634')
            .borderRadius(14)
            .margin({ top: 10 })
            .onChange((value: string) => this.applyMessage = value)
          Button('提交申请')
            .width('100%')
            .height(42)
            .fontColor(GameroTheme.voidBg)
            .backgroundColor(GameroTheme.amber)
            .borderRadius(14)
            .margin({ top: 10 })
            .onClick(() => this.applySelectedRecruit())
        }.cardStyle()

        this.MyApplicationsPanel()
      } else {
        this.EmptyCard('招募详情为空')
      }
    }.width('100%')
  }

  @Builder
  RecruitmentForm(buttonText: string, isEdit: boolean) {
    Column() {
      Text(isEdit ? '编辑招募' : '创建招募').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      if (!isEdit) {
        Text(`默认项目：${this.projectItems.length > 0 ? this.projectItems[0].name : '暂无项目'}`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 6 })
      }
      Row({ space: 8 }) {
        Button('程序').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.recruitPosition === 'program' ? GameroTheme.amber : '#202634').fontColor(this.recruitPosition === 'program' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.recruitPosition = 'program')
        Button('美术').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.recruitPosition === 'art' ? GameroTheme.amber : '#202634').fontColor(this.recruitPosition === 'art' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.recruitPosition = 'art')
        Button('策划').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.recruitPosition === 'design' ? GameroTheme.amber : '#202634').fontColor(this.recruitPosition === 'design' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.recruitPosition = 'design')
      }.width('100%').margin({ top: 10 })
      Row({ space: 8 }) {
        Button('线上').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.recruitCooperationType === 'online' ? GameroTheme.amber : '#202634').fontColor(this.recruitCooperationType === 'online' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.recruitCooperationType = 'online')
        Button('线下').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.recruitCooperationType === 'offline' ? GameroTheme.amber : '#202634').fontColor(this.recruitCooperationType === 'offline' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.recruitCooperationType = 'offline')
        Button('混合').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.recruitCooperationType === 'hybrid' ? GameroTheme.amber : '#202634').fontColor(this.recruitCooperationType === 'hybrid' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.recruitCooperationType = 'hybrid')
      }.width('100%').margin({ top: 10 })
      Row({ space: 8 }) {
        Button('1人').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.recruitHeadcount === 1 ? GameroTheme.amber : '#202634').fontColor(this.recruitHeadcount === 1 ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.recruitHeadcount = 1)
        Button('2人').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.recruitHeadcount === 2 ? GameroTheme.amber : '#202634').fontColor(this.recruitHeadcount === 2 ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.recruitHeadcount = 2)
        Button('3人').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.recruitHeadcount === 3 ? GameroTheme.amber : '#202634').fontColor(this.recruitHeadcount === 3 ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.recruitHeadcount = 3)
      }.width('100%').margin({ top: 10 })
      TextInput({ placeholder: '招募说明', text: this.recruitDescription })
        .height(56)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.recruitDescription = value)
      Button(buttonText)
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => {
          if (isEdit) {
            this.saveRecruitEdit()
          } else {
            this.createRecruitmentFromForm()
          }
        })
    }.cardStyle()
  }

  @Builder
  MyApplicationsPanel() {
    Column() {
      Text('我的申请').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold)
      if (this.applications.length > 0) {
        ForEach(this.applications, (app: ApplicationItem) => {
          Row() {
            Column() {
              Text(app.project_name || `招募 #${app.recruitment_id}`).fontColor(GameroTheme.textPrimary).fontSize(13).fontWeight(FontWeight.Bold)
              Text(`${this.positionText(app.position)} · ${app.message || '无申请说明'}`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
            }.alignItems(HorizontalAlign.Start).layoutWeight(1)
            Text(app.status).fontColor(GameroTheme.amber).fontSize(11)
          }.padding(10).margin({ top: 8 }).backgroundColor('#202634').borderRadius(12)
        })
      } else {
        Text('暂无申请记录').fontColor(GameroTheme.textSecondary).fontSize(12).margin({ top: 10 })
      }
    }.cardStyle()
  }

  @Builder
  TaskBoard() {
    Column() {
      Row() {
        Text('任务看板').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold)
        Blank()
        Text('后端 CRUD').fontColor(GameroTheme.success).fontSize(11)
      }
      if (this.tasks.length > 0) {
        ForEach(this.tasks.slice(0, 3), (task: ProjectTask) => {
          this.TaskRow(task)
        })
      } else {
        Text('暂无真实任务数据')
          .fontColor(GameroTheme.textPrimary)
          .fontSize(14)
          .fontWeight(FontWeight.Bold)
          .margin({ top: 14 })
        Text('请确认当前账号属于某个项目，且该项目已创建任务。这里不会再显示示例任务。')
          .fontColor(GameroTheme.textSecondary)
          .fontSize(12)
          .lineHeight(18)
          .margin({ top: 8 })
      }
    }.cardStyle()
  }

  @Builder
  TaskRow(task: ProjectTask) {
    Column() {
      Row() {
        Column() {
          Text(task.title).fontColor(GameroTheme.textPrimary).fontSize(13).fontWeight(FontWeight.Bold)
          Text(`${task.status} · ${task.priority}`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
        }.alignItems(HorizontalAlign.Start).layoutWeight(1)
        Text(task.due_date ? task.due_date.slice(0, 10) : '无截止').fontColor(GameroTheme.amber).fontSize(11)
      }
      if (this.selectedProject) {
        Row({ space: 8 }) {
          Button('待办').layoutWeight(1).height(32).fontSize(11).backgroundColor('#202634').fontColor(GameroTheme.textPrimary).borderRadius(10).onClick(() => this.moveTask(task, 'todo'))
          Button('进行中').layoutWeight(1).height(32).fontSize(11).backgroundColor('#241A0B').fontColor(GameroTheme.amber).borderRadius(10).onClick(() => this.moveTask(task, 'doing'))
          Button('完成').layoutWeight(1).height(32).fontSize(11).backgroundColor('#0E2518').fontColor(GameroTheme.success).borderRadius(10).onClick(() => this.moveTask(task, 'done'))
          Button('删除').layoutWeight(1).height(32).fontSize(11).backgroundColor('#2A1212').fontColor(GameroTheme.textPrimary).borderRadius(10).onClick(() => this.removeTask(task.id))
        }.width('100%').margin({ top: 10 })
      }
    }.padding(10).margin({ top: 10 }).backgroundColor('#202634').borderRadius(14)
  }

  @Builder
  TaskComposer() {
    Column() {
      Text('新建任务').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '任务标题', text: this.taskTitle })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.taskTitle = value)
      TextInput({ placeholder: '任务描述，可选', text: this.taskDescription })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.taskDescription = value)
      Row({ space: 8 }) {
        Button('低').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.taskPriority === 'low' ? GameroTheme.amber : '#202634').fontColor(this.taskPriority === 'low' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.taskPriority = 'low')
        Button('中').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.taskPriority === 'medium' ? GameroTheme.amber : '#202634').fontColor(this.taskPriority === 'medium' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.taskPriority = 'medium')
        Button('高').layoutWeight(1).height(34).fontSize(11).backgroundColor(this.taskPriority === 'high' ? GameroTheme.amber : '#202634').fontColor(this.taskPriority === 'high' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.taskPriority = 'high')
      }.width('100%').margin({ top: 10 })
      Button('创建任务')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => this.createTaskForSelectedProject())
    }.cardStyle()
  }

  @Builder
  TaskColumn(title: string, task: string, bg: string) {
    Column() {
      Text(title).fontColor(GameroTheme.textMuted).fontSize(11)
      Text(task).fontColor(GameroTheme.textPrimary).fontSize(12).lineHeight(18).margin({ top: 8 })
    }.padding(10).layoutWeight(1).backgroundColor(bg).borderRadius(14)
  }

  @Builder
  AdminPanel() {
    Column() {
      Text('管理后台').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold)
      Text('统计、用户治理、举报处理与内容审核已接真实后端。').fontColor(GameroTheme.textSecondary).fontSize(13).lineHeight(20).margin({ top: 8 })
      Row({ space: 8 }) {
        this.StatPill('用户', this.stats ? `${this.stats.total_users}` : '0')
        this.StatPill('项目', this.stats ? `${this.stats.total_projects}` : '0')
        this.StatPill('帖子', this.stats ? `${this.stats.total_posts}` : '0')
      }.margin({ top: 14 })
      Button('刷新管理数据')
        .width('100%')
        .height(40)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 12 })
        .onClick(() => this.loadAdminData())

      Text('处理备注').fontColor(GameroTheme.textMuted).fontSize(12).margin({ top: 14, bottom: 6 })
      TextInput({ placeholder: '审核备注', text: this.adminHandleNote })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(12)
        .onChange((value: string) => this.adminHandleNote = value)

      Text('按 ID 快速治理').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold).margin({ top: 18, bottom: 10 })
      TextInput({ placeholder: '输入帖子 / 招募 / 项目 ID', text: this.adminTargetIdText })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(12)
        .onChange((value: string) => this.adminTargetIdText = value)
      Row({ space: 8 }) {
        Button('删帖').layoutWeight(1).height(34).fontSize(11).fontColor(GameroTheme.textPrimary).backgroundColor('#2A1212').borderRadius(10).onClick(() => this.adminDeletePostById())
        Button('关招募').layoutWeight(1).height(34).fontSize(11).fontColor(GameroTheme.textPrimary).backgroundColor('#2A1212').borderRadius(10).onClick(() => this.adminCloseRecruitmentById())
      }.width('100%').margin({ top: 10 })
      Row({ space: 8 }) {
        Button('下架项目').layoutWeight(1).height(34).fontSize(11).fontColor(GameroTheme.amber).backgroundColor('#241A0B').borderRadius(10).onClick(() => this.adminBanProjectById())
        Button('删除项目').layoutWeight(1).height(34).fontSize(11).fontColor(GameroTheme.textPrimary).backgroundColor('#2A1212').borderRadius(10).onClick(() => this.adminDeleteProjectById())
      }.width('100%').margin({ top: 8 })

      Text('待处理举报').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold).margin({ top: 18, bottom: 10 })
      if (this.adminReports.length > 0) {
        ForEach(this.adminReports, (report: ReportItem) => {
          Column() {
            Row() {
              Text(`#${report.id} ${this.reportTargetText(report.target_type)} #${report.target_id}`)
                .fontColor(GameroTheme.textPrimary)
                .fontSize(14)
                .fontWeight(FontWeight.Bold)
                .layoutWeight(1)
              Text(this.reportStatusText(report.status)).fontColor(GameroTheme.amber).fontSize(11)
            }
            Text(`举报人 #${report.reporter_id} · ${report.created_at ? report.created_at.slice(0, 10) : ''}`)
              .fontColor(GameroTheme.textMuted)
              .fontSize(11)
              .margin({ top: 5 })
            Text(report.reason || '未填写原因')
              .fontColor(GameroTheme.textSecondary)
              .fontSize(12)
              .lineHeight(18)
              .margin({ top: 8 })
            if (report.supplement && report.supplement.length > 0) {
              Text(report.supplement).fontColor(GameroTheme.textMuted).fontSize(11).lineHeight(16).margin({ top: 6 })
            }
            Row({ space: 8 }) {
              Button(report.target_type === 'post' ? '删帖并处理' : report.target_type === 'project' ? '下架并处理' : report.target_type === 'recruitment' ? '关闭并处理' : report.target_type === 'log' || report.target_type === 'project_release' ? '隐藏并处理' : '标记已处理')
                .layoutWeight(1)
                .height(34)
                .fontSize(11)
                .fontColor(GameroTheme.voidBg)
                .backgroundColor(GameroTheme.amber)
                .borderRadius(10)
                .onClick(() => this.adminHandleReport(report))
              if (report.target_type === 'project') {
                Button('删除项目')
                  .layoutWeight(1)
                  .height(34)
                  .fontSize(11)
                  .fontColor(GameroTheme.textPrimary)
                  .backgroundColor('#2A1212')
                  .borderRadius(10)
                  .onClick(() => this.adminDeleteProjectFromReport(report))
              }
              Button('驳回')
                .layoutWeight(1)
                .height(34)
                .fontSize(11)
                .fontColor(GameroTheme.textPrimary)
                .backgroundColor('#202634')
                .borderRadius(10)
                .onClick(() => this.adminRejectReport(report.id))
            }.margin({ top: 10 })
          }.padding(12).backgroundColor('#111722').borderRadius(14).margin({ bottom: 10 })
        })
      } else {
        Text('暂无待处理举报。').fontColor(GameroTheme.textMuted).fontSize(12).margin({ bottom: 8 })
      }

      Text('用户治理').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold).margin({ top: 16, bottom: 10 })
      TextInput({ placeholder: '封禁原因', text: this.adminBanReason })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(12)
        .margin({ bottom: 10 })
        .onChange((value: string) => this.adminBanReason = value)
      if (this.adminUsers.length > 0) {
        ForEach(this.adminUsers, (user: UserProfileItem) => {
          Column() {
            Row() {
              Column() {
                Text(user.nickname || user.username).fontColor(GameroTheme.textPrimary).fontSize(14).fontWeight(FontWeight.Bold)
                Text(`#${user.id} · ${user.role || 'user'} · 项目 ${user.project_count || 0}`)
                  .fontColor(GameroTheme.textMuted)
                  .fontSize(11)
                  .margin({ top: 4 })
              }.alignItems(HorizontalAlign.Start).layoutWeight(1)
              Button('封禁')
                .width(68)
                .height(32)
                .fontSize(11)
                .fontColor(GameroTheme.textPrimary)
                .backgroundColor('#2A1212')
                .borderRadius(10)
                .onClick(() => this.adminBanUser(user.id))
              Button('解封')
                .width(68)
                .height(32)
                .fontSize(11)
                .fontColor(GameroTheme.amber)
                .backgroundColor('#241A0B')
                .borderRadius(10)
                .margin({ left: 6 })
                .onClick(() => this.adminUnbanUser(user.id))
            }
          }.padding(12).backgroundColor('#111722').borderRadius(14).margin({ bottom: 10 })
        })
      } else {
        Text('暂无用户数据。').fontColor(GameroTheme.textMuted).fontSize(12)
      }

      Text('敏感词管理').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold).margin({ top: 16, bottom: 10 })
      Text('Web 端已有 /admin/sensitive-words 接口封装；这里直接接后端真实词库。')
        .fontColor(GameroTheme.textMuted)
        .fontSize(11)
        .lineHeight(16)
        .margin({ bottom: 8 })
      TextInput({ placeholder: '输入敏感词，多个用英文逗号分隔', text: this.sensitiveWordInput })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(12)
        .onChange((value: string) => this.sensitiveWordInput = value)
      Button('添加敏感词')
        .width('100%')
        .height(40)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10, bottom: 10 })
        .onClick(() => this.adminAddSensitiveWords())
      if (this.sensitiveWords.length > 0) {
        ForEach(this.sensitiveWords, (word: string) => {
          Row() {
            Text(word).fontColor(GameroTheme.textPrimary).fontSize(13).layoutWeight(1)
            Button('删除')
              .width(68)
              .height(30)
              .fontSize(11)
              .fontColor(GameroTheme.textPrimary)
              .backgroundColor('#2A1212')
              .borderRadius(10)
              .onClick(() => this.adminDeleteSensitiveWord(word))
          }.padding(10).backgroundColor('#111722').borderRadius(12).margin({ bottom: 8 })
        })
      } else {
        Text('暂无敏感词。').fontColor(GameroTheme.textMuted).fontSize(12)
      }
    }.cardStyle()
  }

  @Builder
  MessagePanel() {
    if (this.successText.length > 0) {
      Text(this.successText).fontColor(GameroTheme.success).fontSize(12).lineHeight(18).padding(10).backgroundColor('#14281D').borderRadius(8).margin({ bottom: 10 })
    }
    if (this.errorText.length > 0) {
      Text(this.errorText).fontColor(GameroTheme.coral).fontSize(12).lineHeight(18).padding(10).backgroundColor('#2A1718').borderRadius(8).margin({ bottom: 10 })
    }
    if (this.isLoading) {
      Text('正在连接真实后端...').fontColor(GameroTheme.textSecondary).fontSize(12).padding(10).backgroundColor('#20202A').borderRadius(8).margin({ bottom: 10 })
    }
  }

  @Builder
  AuthTabs() {
    Row({ space: 8 }) {
      Button('登录')
        .layoutWeight(1)
        .fontColor(this.authMode === 'login' ? GameroTheme.voidBg : GameroTheme.textSecondary)
        .backgroundColor(this.authMode === 'login' ? GameroTheme.amber : '#202634')
        .borderRadius(14)
        .onClick(() => this.authMode = 'login')
      Button('注册')
        .layoutWeight(1)
        .fontColor(this.authMode === 'register' ? GameroTheme.voidBg : GameroTheme.textSecondary)
        .backgroundColor(this.authMode === 'register' ? GameroTheme.amber : '#202634')
        .borderRadius(14)
        .onClick(() => this.authMode = 'register')
    }.width('100%').margin({ bottom: 16 })
  }

  @Builder
  LoginForm() {
    Column() {
      Text('登录 Gamero').fontColor(GameroTheme.textPrimary).fontSize(20).fontWeight(FontWeight.Bold)
      Text('使用真实后端 /auth/login/email 登录。').fontColor(GameroTheme.textMuted).fontSize(12).margin({ top: 6, bottom: 16 })
      TextInput({ placeholder: '邮箱', text: this.loginEmail })
        .type(InputType.Email)
        .height(46)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ bottom: 10 })
        .onChange((value: string) => this.loginEmail = value)
      TextInput({ placeholder: '密码', text: this.loginPassword })
        .type(InputType.Password)
        .height(46)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ bottom: 14 })
        .onChange((value: string) => this.loginPassword = value)
      Button('登录并加载真实数据')
        .width('100%')
        .height(46)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(16)
        .onClick(() => this.loginAndLoadProtectedData())
      Button('填入管理员账号')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.amber)
        .backgroundColor('#241A0B')
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => {
          this.loginEmail = ApiConfig.adminEmail;
          this.loginPassword = ApiConfig.adminPassword;
        })
    }.cardStyle()
  }

  @Builder
  RegisterForm() {
    Column() {
      Text('注册账号').fontColor(GameroTheme.textPrimary).fontSize(20).fontWeight(FontWeight.Bold)
      Text('先发送邮箱验证码，再使用 /auth/register/email 注册。').fontColor(GameroTheme.textMuted).fontSize(12).margin({ top: 6, bottom: 16 })
      TextInput({ placeholder: '邮箱', text: this.registerEmail })
        .type(InputType.Email)
        .height(46)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ bottom: 10 })
        .onChange((value: string) => this.registerEmail = value)
      Button('发送验证码')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ bottom: 10 })
        .onClick(() => this.sendRegisterCode())
      TextInput({ placeholder: '验证码', text: this.registerCode })
        .height(46)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ bottom: 10 })
        .onChange((value: string) => this.registerCode = value)
      TextInput({ placeholder: '昵称，可选', text: this.registerNickname })
        .height(46)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ bottom: 10 })
        .onChange((value: string) => this.registerNickname = value)
      TextInput({ placeholder: '密码', text: this.registerPassword })
        .type(InputType.Password)
        .height(46)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ bottom: 14 })
        .onChange((value: string) => this.registerPassword = value)
      Button('注册并登录')
        .width('100%')
        .height(46)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(16)
        .onClick(() => this.registerAndLogin())
    }.cardStyle()
  }

  @Builder
  AuthPanel() {
    Column() {
      this.AuthTabs()
      this.MessagePanel()
      if (this.authMode === 'login') {
        this.LoginForm()
      } else {
        this.RegisterForm()
      }
    }.width('100%')
  }

  @Builder
  AccountPanel() {
    Column() {
      Text(this.currentUser ? this.currentUser.nickname || this.currentUser.username : '未登录')
        .fontColor(GameroTheme.textPrimary)
        .fontSize(20)
        .fontWeight(FontWeight.Bold)
      Text(this.currentUser ? `${this.currentUser.role || 'user'} · ${this.currentUser.email || this.loginEmail}` : '请先登录')
        .fontColor(GameroTheme.textMuted)
        .fontSize(12)
        .margin({ top: 6 })
      Text(`后端地址：${this.backendUrl}`)
        .fontColor(GameroTheme.textSecondary)
        .fontSize(12)
        .lineHeight(18)
        .margin({ top: 12 })
      Button('退出登录')
        .width('100%')
        .height(44)
        .fontColor(GameroTheme.textPrimary)
        .backgroundColor('#2A1212')
        .borderRadius(14)
        .margin({ top: 16 })
        .onClick(() => this.logout())
    }.cardStyle()
  }

  @Builder
  FeatureCard(title: string, desc: string, status: string, view: string) {
    Column() {
      Row() {
        Text(title).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold).layoutWeight(1)
        Text(status)
          .fontColor(status === '已接真实接口' ? GameroTheme.success : GameroTheme.amber)
          .fontSize(11)
          .padding({ left: 8, right: 8, top: 4, bottom: 4 })
          .backgroundColor(status === '已接真实接口' ? '#0E2518' : '#241A0B')
          .borderRadius(10)
      }
      Text(desc).fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
    }.cardStyle().onClick(() => this.openFeature(view))
  }

  @Builder
  BackToFeatures() {
    Button('返回功能列表')
      .fontColor(GameroTheme.amber)
      .backgroundColor('#241A0B')
      .borderRadius(14)
      .margin({ bottom: 12 })
      .onClick(() => this.featureView = 'menu')
  }

  @Builder
  PostListPage() {
    Column() {
      this.BackToFeatures()
      Text('社区帖子').sectionTitle()
      this.MessagePanel()
      this.PostComposer()
      if (this.posts.length > 0) {
        ForEach(this.posts, (post: PostItem) => {
          Column() {
            Text(post.title).fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
            Text(post.content_snippet || post.content || '暂无内容').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
            Text(`${post.author ? post.author.nickname : 'Gamero 用户'} · ${post.like_count} 赞 · ${post.comment_count} 评论 · ${post.view_count} 浏览`)
              .fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 10 })
          }.cardStyle().onClick(() => this.openPostDetail(post.id))
        })
      } else {
        this.EmptyCard('暂无社区帖子真实数据')
      }
    }.width('100%')
  }

  @Builder
  PostComposer() {
    Column() {
      Text('发布帖子').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '帖子标题', text: this.postTitle })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.postTitle = value)
      TextInput({ placeholder: '帖子内容', text: this.postContent })
        .height(56)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.postContent = value)
      Button('发布帖子')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => this.createCommunityPost())
    }.cardStyle()
  }

  @Builder
  CommentList() {
    Text('评论').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold).margin({ top: 14, bottom: 10 })
    if (this.comments.length > 0) {
      ForEach(this.comments, (comment: CommentItem) => {
        Column() {
          Text(comment.author ? comment.author.nickname : comment.user_nickname || comment.user_username || 'Gamero 用户')
            .fontColor(GameroTheme.amber)
            .fontSize(12)
            .fontWeight(FontWeight.Bold)
          Text(comment.content).fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 6 })
          Text(`${comment.like_count} 赞 · ${comment.created_at ? comment.created_at.slice(0, 10) : '刚刚'}`)
            .fontColor(GameroTheme.textMuted)
            .fontSize(11)
            .margin({ top: 8 })
        }.cardStyle()
      })
    } else {
      this.EmptyCard('暂无评论')
    }
  }

  @Builder
  PostDetailPage() {
    Column() {
      Button('返回上一页')
        .fontColor(GameroTheme.amber)
        .backgroundColor('#241A0B')
        .borderRadius(14)
        .margin({ bottom: 12 })
        .onClick(() => this.featureView = this.postBackView)
      this.MessagePanel()
      if (this.selectedPost) {
        Column() {
          Text(this.selectedPost.title).fontColor(GameroTheme.textPrimary).fontSize(22).fontWeight(FontWeight.Bold)
          Text(`${this.selectedPost.author ? this.selectedPost.author.nickname : 'Gamero 用户'} · ${this.selectedPost.view_count} 浏览 · ${this.selectedPost.comment_count} 评论`)
            .fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 8 })
          Text(this.selectedPost.content || this.selectedPost.content_snippet || '暂无内容')
            .fontColor(GameroTheme.textSecondary)
            .fontSize(14)
            .lineHeight(22)
            .margin({ top: 14 })
          Button(`点赞 ${this.selectedPost.like_count}`)
            .fontColor(GameroTheme.voidBg)
            .backgroundColor(GameroTheme.amber)
            .borderRadius(14)
            .margin({ top: 14 })
            .onClick(() => this.likeSelectedPost())
          Row({ space: 8 }) {
            Button(this.isEditingPost ? '取消编辑' : '编辑帖子')
              .layoutWeight(1)
              .height(40)
              .fontColor(GameroTheme.amber)
              .backgroundColor('#241A0B')
              .borderRadius(12)
              .onClick(() => {
                if (this.isEditingPost) {
                  this.cancelEditPost()
                } else {
                  this.startEditPost()
                }
              })
            Button('删除帖子')
              .layoutWeight(1)
              .height(40)
              .fontColor(GameroTheme.textPrimary)
              .backgroundColor('#2A1212')
              .borderRadius(12)
              .onClick(() => this.deleteSelectedPost())
          }.width('100%').margin({ top: 10 })
          Row({ space: 8 }) {
            Button('上传图片')
              .layoutWeight(1)
              .height(38)
              .fontColor(GameroTheme.amber)
              .backgroundColor('#241A0B')
              .borderRadius(12)
              .onClick(() => this.uploadPostImageToSelectedPost())
            Button('上传视频')
              .layoutWeight(1)
              .height(38)
              .fontColor(GameroTheme.textPrimary)
              .backgroundColor('#202634')
              .borderRadius(12)
              .onClick(() => this.uploadPostVideoToSelectedPost())
          }.width('100%').margin({ top: 10 })
        }.cardStyle()
        if (this.isEditingPost) {
          this.PostEditForm()
        }
        this.CommentComposer('发布帖子评论', true)
        this.CommentList()
      } else {
        this.EmptyCard('帖子详情为空')
      }
    }.width('100%')
  }

  @Builder
  PostEditForm() {
    Column() {
      Text('编辑帖子').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '帖子标题', text: this.postTitle })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.postTitle = value)
      TextInput({ placeholder: '帖子内容', text: this.postContent })
        .height(56)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.postContent = value)
      Button('保存帖子')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => this.savePostEdit())
    }.cardStyle()
  }

  @Builder
  DevLogListPage() {
    Column() {
      this.BackToFeatures()
      Text('开发日志').sectionTitle()
      this.MessagePanel()
      if (this.devLogs.length > 0) {
        ForEach(this.devLogs, (log: DevLogItem) => {
          Column() {
            Text(log.title).fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
            Text(log.content || '暂无日志内容').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
            Text(`${log.project_name || '项目'} · ${log.log_type} · ${log.like_count} 赞 · ${log.comment_count} 评论`)
              .fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 10 })
          }.cardStyle().onClick(() => this.openDevLogDetail(log.id))
        })
      } else {
        this.EmptyCard('暂无开发日志真实数据')
      }
    }.width('100%')
  }

  @Builder
  DevLogDetailPage() {
    Column() {
      Button('返回上一页')
        .fontColor(GameroTheme.amber)
        .backgroundColor('#241A0B')
        .borderRadius(14)
        .margin({ bottom: 12 })
        .onClick(() => this.featureView = this.devLogBackView)
      this.MessagePanel()
      if (this.selectedDevLog) {
        Column() {
          Text(this.selectedDevLog.title).fontColor(GameroTheme.textPrimary).fontSize(22).fontWeight(FontWeight.Bold)
          Text(`${this.selectedDevLog.project_name || '项目'} · ${this.selectedDevLog.log_type} · ${this.selectedDevLog.view_count} 浏览 · ${this.selectedDevLog.comment_count} 评论`)
            .fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 8 })
          Text(this.selectedDevLog.content || '暂无日志内容')
            .fontColor(GameroTheme.textSecondary)
            .fontSize(14)
            .lineHeight(22)
            .margin({ top: 14 })
          Button(`点赞 ${this.selectedDevLog.like_count}`)
            .fontColor(GameroTheme.voidBg)
            .backgroundColor(GameroTheme.amber)
            .borderRadius(14)
            .margin({ top: 14 })
            .onClick(() => this.likeSelectedDevLog())
          Row({ space: 8 }) {
            Button(this.isEditingDevLog ? '取消编辑' : '编辑日志')
              .layoutWeight(1)
              .height(40)
              .fontColor(GameroTheme.amber)
              .backgroundColor('#241A0B')
              .borderRadius(12)
              .onClick(() => {
                if (this.isEditingDevLog) {
                  this.cancelEditDevLog()
                } else {
                  this.startEditDevLog()
                }
              })
            Button('删除日志')
              .layoutWeight(1)
              .height(40)
              .fontColor(GameroTheme.textPrimary)
              .backgroundColor('#2A1212')
              .borderRadius(12)
              .onClick(() => this.deleteSelectedDevLog())
          }.width('100%').margin({ top: 10 })
          Row({ space: 8 }) {
            Button('上传图片')
              .layoutWeight(1)
              .height(38)
              .fontColor(GameroTheme.amber)
              .backgroundColor('#241A0B')
              .borderRadius(12)
              .onClick(() => this.uploadDevLogImage())
            Button('上传视频')
              .layoutWeight(1)
              .height(38)
              .fontColor(GameroTheme.textPrimary)
              .backgroundColor('#202634')
              .borderRadius(12)
              .onClick(() => this.uploadDevLogVideo())
          }.width('100%').margin({ top: 10 })
        }.cardStyle()
        if (this.isEditingDevLog) {
          this.DevLogEditForm()
        }
        this.CommentComposer('发布日志评论', false)
        this.CommentList()
      } else {
        this.EmptyCard('开发日志详情为空')
      }
    }.width('100%')
  }

  @Builder
  DevLogEditForm() {
    Column() {
      Text('编辑开发日志').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '日志标题', text: this.devLogTitle })
        .height(42)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.devLogTitle = value)
      TextInput({ placeholder: '日志内容', text: this.devLogContent })
        .height(56)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ top: 10 })
        .onChange((value: string) => this.devLogContent = value)
      Button('保存日志')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ top: 10 })
        .onClick(() => this.saveDevLogEdit())
    }.cardStyle()
  }

  @Builder
  CommentComposer(buttonText: string, isPost: boolean) {
    Column() {
      TextInput({ placeholder: '写下你的评论', text: this.commentInput })
        .height(46)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ bottom: 10 })
        .onChange((value: string) => this.commentInput = value)
      Button(buttonText)
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .onClick(() => {
          if (isPost) {
            this.submitPostComment()
          } else {
            this.submitDevLogComment()
          }
        })
    }.cardStyle()
  }

  @Builder
  NotificationListPage() {
    Column() {
      this.BackToFeatures()
      Text(`通知中心 · 未读 ${this.unreadCount}`).sectionTitle()
      Button('全部标记为已读')
        .width('100%')
        .height(42)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ bottom: 12 })
        .onClick(() => this.markAllNotificationsRead())
      this.MessagePanel()
      if (this.notifications.length > 0) {
        ForEach(this.notifications, (item: NotificationItem) => {
          Column() {
            Row() {
              Text(item.title).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold).layoutWeight(1)
              Text(item.is_read ? '已读' : '未读').fontColor(item.is_read ? GameroTheme.textMuted : GameroTheme.amber).fontSize(11)
            }
            Text(item.content).fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
            Text(`${item.type} · ${item.created_at ? item.created_at.slice(0, 10) : '刚刚'}`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 10 })
            Row({ space: 8 }) {
              if (!item.is_read) {
                Button('标记已读')
                  .layoutWeight(1)
                  .height(38)
                  .fontColor(GameroTheme.amber)
                  .backgroundColor('#241A0B')
                  .borderRadius(12)
                  .onClick(() => this.markNotificationRead(item.id))
              }
              Button('删除')
                .layoutWeight(1)
                .height(38)
                .fontColor(GameroTheme.textPrimary)
                .backgroundColor('#2A1212')
                .borderRadius(12)
                .onClick(() => this.deleteNotification(item.id))
            }.width('100%').margin({ top: 12 })
          }.cardStyle()
        })
      } else {
        this.EmptyCard('暂无通知真实数据')
      }
    }.width('100%')
  }

  @Builder
  SearchPage() {
    Column() {
      this.BackToFeatures()
      Text('搜索发现').sectionTitle()
      TextInput({ placeholder: '搜索项目、帖子、日志', text: this.searchKeyword })
        .height(46)
        .fontColor(GameroTheme.textPrimary)
        .placeholderColor(GameroTheme.textMuted)
        .backgroundColor('#202634')
        .borderRadius(14)
        .margin({ bottom: 10 })
        .onChange((value: string) => this.searchKeyword = value)
      Button('搜索')
        .width('100%')
        .height(44)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ bottom: 12 })
        .onClick(() => this.runSearch())
      this.MessagePanel()
      if (this.searchResults.length > 0) {
        ForEach(this.searchResults, (item: SearchItem) => {
          Column() {
            Text(item.title || item.name || `结果 #${item.id}`).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold)
            Text(item.content || item.description || '暂无摘要').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
          }.cardStyle()
        })
      } else {
        this.EmptyCard('请输入关键词搜索真实后端数据')
      }
    }.width('100%')
  }

  @Builder
  TalentPage() {
    Column() {
      this.BackToFeatures()
      Text('人才市场').sectionTitle()
      this.MessagePanel()
      if (this.talents.length > 0) {
        ForEach(this.talents, (item: TalentItem) => {
          Column() {
            Row() {
              Column() {
                Text(item.nickname || item.username).fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
                Text(`${item.is_available ? '可合作' : '暂不可合作'} · ${this.coopText(item.coop_preference || 'online')} · ${item.project_count} 项目`)
                  .fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
              }.alignItems(HorizontalAlign.Start).layoutWeight(1)
              Text('主页').fontColor(GameroTheme.voidBg).fontSize(11).fontWeight(FontWeight.Bold).padding({ left: 12, right: 12, top: 7, bottom: 7 })
                .backgroundColor(GameroTheme.amber).borderRadius(14)
            }
            Text(item.bio || '暂无个人简介').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 10 })
            if (item.skills.length > 0) {
              Text(`技能：${item.skills[0].name} · ${item.skills[0].level}`).fontColor(GameroTheme.amber).fontSize(11).margin({ top: 10 })
            }
          }.cardStyle().onClick(() => this.openUserProfile(item.id))
        })
      } else {
        this.EmptyCard('暂无真实人才数据')
      }
    }.width('100%')
  }

  @Builder
  UserProfilePage() {
    Column() {
      Button('返回上一页')
        .fontColor(GameroTheme.amber)
        .backgroundColor('#241A0B')
        .borderRadius(14)
        .margin({ bottom: 12 })
        .onClick(() => this.featureView = 'talents')
      this.MessagePanel()
      if (this.selectedUser) {
        Column() {
          Text(this.selectedUser.nickname || this.selectedUser.username).fontColor(GameroTheme.textPrimary).fontSize(24).fontWeight(FontWeight.Bold)
          Text(`${this.selectedUser.is_available ? '可合作' : '暂不可合作'} · ${this.coopText(this.selectedUser.coop_preference || 'online')} · ${this.selectedUser.location || '未填写地区'}`)
            .fontColor(GameroTheme.textMuted).fontSize(12).margin({ top: 8 })
          Text(this.selectedUser.bio || '暂无个人简介')
            .fontColor(GameroTheme.textSecondary).fontSize(14).lineHeight(22).margin({ top: 14 })
          Row({ space: 8 }) {
            this.StatPill('项目', `${this.selectedUser.project_count}`)
            this.StatPill('日志', `${this.selectedUser.log_count}`)
            this.StatPill('粉丝', `${this.selectedUser.followers_count}`)
          }.width('100%').margin({ top: 14 })
          Button(this.selectedUser.is_following ? '取消关注' : '关注用户')
            .width('100%')
            .height(42)
            .fontColor(GameroTheme.voidBg)
            .backgroundColor(GameroTheme.amber)
            .borderRadius(14)
            .margin({ top: 14 })
            .onClick(() => this.toggleFollowSelectedUser())
        }.cardStyle()
        Column() {
          Text('技能').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold)
          if (this.selectedUser.skills.length > 0) {
            ForEach(this.selectedUser.skills, (skill: UserSkillItem) => {
              Text(`${skill.name} · ${skill.level}`).fontColor(GameroTheme.textSecondary).fontSize(12).margin({ top: 8 })
            })
          } else {
            Text('暂无技能').fontColor(GameroTheme.textSecondary).fontSize(12).margin({ top: 8 })
          }
        }.cardStyle()
      } else {
        this.EmptyCard('用户主页为空')
      }
    }.width('100%')
  }

  @Builder
  SettingsPage() {
    Column() {
      this.BackToFeatures()
      Text('账号设置').sectionTitle()
      this.MessagePanel()
      Column() {
        Text('个人资料').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
        TextInput({ placeholder: '昵称', text: this.profileNickname })
          .height(42).fontColor(GameroTheme.textPrimary).placeholderColor(GameroTheme.textMuted).backgroundColor('#202634').borderRadius(14).margin({ top: 10 })
          .onChange((value: string) => this.profileNickname = value)
        TextInput({ placeholder: '简介', text: this.profileBio })
          .height(56).fontColor(GameroTheme.textPrimary).placeholderColor(GameroTheme.textMuted).backgroundColor('#202634').borderRadius(14).margin({ top: 10 })
          .onChange((value: string) => this.profileBio = value)
        TextInput({ placeholder: '地区', text: this.profileLocation })
          .height(42).fontColor(GameroTheme.textPrimary).placeholderColor(GameroTheme.textMuted).backgroundColor('#202634').borderRadius(14).margin({ top: 10 })
          .onChange((value: string) => this.profileLocation = value)
        Row({ space: 8 }) {
          Button(this.profileAvailable ? '可合作' : '暂不可合作')
            .layoutWeight(1).height(36).fontSize(11).backgroundColor(this.profileAvailable ? GameroTheme.amber : '#202634').fontColor(this.profileAvailable ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10)
            .onClick(() => this.profileAvailable = !this.profileAvailable)
          Button('线上').layoutWeight(1).height(36).fontSize(11).backgroundColor(this.profileCoopPreference === 'online' ? GameroTheme.amber : '#202634').fontColor(this.profileCoopPreference === 'online' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.profileCoopPreference = 'online')
          Button('混合').layoutWeight(1).height(36).fontSize(11).backgroundColor(this.profileCoopPreference === 'hybrid' ? GameroTheme.amber : '#202634').fontColor(this.profileCoopPreference === 'hybrid' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.profileCoopPreference = 'hybrid')
        }.width('100%').margin({ top: 10 })
        Button('保存资料')
          .width('100%').height(42).fontColor(GameroTheme.voidBg).backgroundColor(GameroTheme.amber).borderRadius(14).margin({ top: 10 })
          .onClick(() => this.saveMyProfile())
      }.cardStyle()
      this.SkillManagerPanel()
      this.PortfolioManagerPanel()
      this.NotificationPreferencePanel()
    }.width('100%')
  }

  @Builder
  SkillManagerPanel() {
    Column() {
      Text('技能管理').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      Row({ space: 8 }) {
        Button('程序').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.skillCategory === 'program' ? GameroTheme.amber : '#202634').fontColor(this.skillCategory === 'program' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.skillCategory = 'program')
        Button('美术').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.skillCategory === 'art' ? GameroTheme.amber : '#202634').fontColor(this.skillCategory === 'art' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.skillCategory = 'art')
        Button('策划').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.skillCategory === 'design' ? GameroTheme.amber : '#202634').fontColor(this.skillCategory === 'design' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.skillCategory = 'design')
        Button('音效').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.skillCategory === 'sound' ? GameroTheme.amber : '#202634').fontColor(this.skillCategory === 'sound' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.skillCategory = 'sound')
      }.width('100%').margin({ top: 10 })
      TextInput({ placeholder: '技能名称，例如 Unity / 像素风 / 数值策划', text: this.skillName })
        .height(42).fontColor(GameroTheme.textPrimary).placeholderColor(GameroTheme.textMuted).backgroundColor('#202634').borderRadius(14).margin({ top: 10 })
        .onChange((value: string) => this.skillName = value)
      Row({ space: 8 }) {
        Button('入门').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.skillLevel === 'beginner' ? GameroTheme.amber : '#202634').fontColor(this.skillLevel === 'beginner' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.skillLevel = 'beginner')
        Button('中级').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.skillLevel === 'intermediate' ? GameroTheme.amber : '#202634').fontColor(this.skillLevel === 'intermediate' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.skillLevel = 'intermediate')
        Button('高级').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.skillLevel === 'advanced' ? GameroTheme.amber : '#202634').fontColor(this.skillLevel === 'advanced' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.skillLevel = 'advanced')
      }.width('100%').margin({ top: 10 })
      TextInput({ placeholder: '技能描述，可选', text: this.skillDescription })
        .height(46).fontColor(GameroTheme.textPrimary).placeholderColor(GameroTheme.textMuted).backgroundColor('#202634').borderRadius(14).margin({ top: 10 })
        .onChange((value: string) => this.skillDescription = value)
      Row({ space: 8 }) {
        Button('添加技能').layoutWeight(1).height(40).fontColor(GameroTheme.amber).backgroundColor('#241A0B').borderRadius(14).onClick(() => this.addSkillDraft())
        Button('保存技能').layoutWeight(1).height(40).fontColor(GameroTheme.voidBg).backgroundColor(GameroTheme.amber).borderRadius(14).onClick(() => this.saveSkills())
      }.width('100%').margin({ top: 10 })
      if (this.profileSkills.length > 0) {
        ForEach(this.profileSkills, (skill: UserSkillItem) => {
          Row() {
            Column() {
              Text(`${skill.name} · ${skill.level}`).fontColor(GameroTheme.textPrimary).fontSize(13).fontWeight(FontWeight.Bold)
              Text(`${skill.category}${skill.description ? ` · ${skill.description}` : ''}`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
            }.alignItems(HorizontalAlign.Start).layoutWeight(1)
            Button('移除').width(64).height(30).fontSize(11).fontColor(GameroTheme.textPrimary).backgroundColor('#2A1212').borderRadius(10).onClick(() => this.removeSkillDraft(skill.id))
          }.padding(10).backgroundColor('#111722').borderRadius(12).margin({ top: 8 })
        })
      } else {
        Text('暂无技能，添加后会提升人才库展示质量。').fontColor(GameroTheme.textMuted).fontSize(12).margin({ top: 10 })
      }
    }.cardStyle()
  }

  @Builder
  PortfolioManagerPanel() {
    Column() {
      Text('作品集管理').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      TextInput({ placeholder: '作品名称', text: this.portfolioName })
        .height(42).fontColor(GameroTheme.textPrimary).placeholderColor(GameroTheme.textMuted).backgroundColor('#202634').borderRadius(14).margin({ top: 10 })
        .onChange((value: string) => this.portfolioName = value)
      Row({ space: 8 }) {
        Button('游戏').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.portfolioType === 'game' ? GameroTheme.amber : '#202634').fontColor(this.portfolioType === 'game' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.portfolioType = 'game')
        Button('Demo').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.portfolioType === 'demo' ? GameroTheme.amber : '#202634').fontColor(this.portfolioType === 'demo' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.portfolioType = 'demo')
        Button('美术').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.portfolioType === 'art' ? GameroTheme.amber : '#202634').fontColor(this.portfolioType === 'art' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.portfolioType = 'art')
        Button('代码').layoutWeight(1).height(32).fontSize(11).backgroundColor(this.portfolioType === 'code' ? GameroTheme.amber : '#202634').fontColor(this.portfolioType === 'code' ? GameroTheme.voidBg : GameroTheme.textPrimary).borderRadius(10).onClick(() => this.portfolioType = 'code')
      }.width('100%').margin({ top: 10 })
      TextInput({ placeholder: '作品链接，可选', text: this.portfolioLink })
        .height(42).fontColor(GameroTheme.textPrimary).placeholderColor(GameroTheme.textMuted).backgroundColor('#202634').borderRadius(14).margin({ top: 10 })
        .onChange((value: string) => this.portfolioLink = value)
      TextInput({ placeholder: '作品说明，可选', text: this.portfolioDescription })
        .height(52).fontColor(GameroTheme.textPrimary).placeholderColor(GameroTheme.textMuted).backgroundColor('#202634').borderRadius(14).margin({ top: 10 })
        .onChange((value: string) => this.portfolioDescription = value)
      Button('添加作品')
        .width('100%').height(40).fontColor(GameroTheme.voidBg).backgroundColor(GameroTheme.amber).borderRadius(14).margin({ top: 10 })
        .onClick(() => this.createPortfolioItem())
      if (this.portfolioItems.length > 0) {
        ForEach(this.portfolioItems, (item: PortfolioItem) => {
          Row() {
            Column() {
              Text(`${item.name} · ${item.type}`).fontColor(GameroTheme.textPrimary).fontSize(13).fontWeight(FontWeight.Bold)
              Text(item.description || item.link || '暂无说明').fontColor(GameroTheme.textMuted).fontSize(11).lineHeight(16).margin({ top: 4 })
            }.alignItems(HorizontalAlign.Start).layoutWeight(1)
            Button('删除').width(64).height(30).fontSize(11).fontColor(GameroTheme.textPrimary).backgroundColor('#2A1212').borderRadius(10).onClick(() => this.deletePortfolioItem(item.id))
          }.padding(10).backgroundColor('#111722').borderRadius(12).margin({ top: 8 })
        })
      } else {
        Text('暂无作品集。').fontColor(GameroTheme.textMuted).fontSize(12).margin({ top: 10 })
      }
    }.cardStyle()
  }

  @Builder
  NotificationPreferencePanel() {
    Column() {
      Text('通知偏好').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
      this.PreferenceButton('提及我', 'mention')
      this.PreferenceButton('新粉丝', 'new_follower')
      this.PreferenceButton('帖子被赞', 'post_liked')
      this.PreferenceButton('帖子被评论', 'post_commented')
      this.PreferenceButton('开发日志被赞', 'log_liked')
      this.PreferenceButton('开发日志被评论', 'log_commented')
      this.PreferenceButton('招募申请', 'recruitment_applied')
      this.PreferenceButton('项目邀请', 'talent_invited')
      this.PreferenceButton('系统通知', 'system')
      this.PreferenceButton('举报处理结果', 'report_handled')
    }.cardStyle()
  }

  @Builder
  PreferenceButton(label: string, type: string) {
    Row() {
      Text(label).fontColor(GameroTheme.textPrimary).fontSize(13).layoutWeight(1)
      Button(this.notificationEnabled(type) ? '开启' : '关闭')
        .width(72)
        .height(32)
        .fontSize(11)
        .fontColor(this.notificationEnabled(type) ? GameroTheme.voidBg : GameroTheme.textPrimary)
        .backgroundColor(this.notificationEnabled(type) ? GameroTheme.amber : '#202634')
        .borderRadius(10)
        .onClick(() => this.toggleNotificationPreference(type))
    }.padding(10).backgroundColor('#111722').borderRadius(12).margin({ top: 8 })
  }

  @Builder
  CollectionsPage() {
    Column() {
      this.BackToFeatures()
      Text('我的收藏').sectionTitle()
      this.MessagePanel()
      Button('刷新收藏')
        .width('100%')
        .height(40)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ bottom: 12 })
        .onClick(() => this.loadCollections())

      Text('收藏的项目').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold).margin({ bottom: 10 })
      if (this.collectedProjects.length > 0) {
        ForEach(this.collectedProjects, (item: ProjectItem) => {
          Column() {
            Text(item.name).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold)
            Text(item.description || '暂无项目简介').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
            Text(`${this.genreText(item.genre)} · ${this.statusText(item.status)} · ${item.follower_count || 0} 关注`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 8 })
          }.cardStyle().onClick(() => this.openProjectDetail(item.id))
        })
      } else {
        this.EmptyCard('暂无收藏项目')
      }

      Text('收藏的帖子').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold).margin({ top: 16, bottom: 10 })
      if (this.collectedPosts.length > 0) {
        ForEach(this.collectedPosts, (post: PostItem) => {
          Column() {
            Text(post.title).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold)
            Text(post.content_snippet || post.content || '暂无内容').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
          }.cardStyle().onClick(() => this.openPostDetail(post.id))
        })
      } else {
        this.EmptyCard('暂无收藏帖子')
      }

      Text('收藏的日志').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold).margin({ top: 16, bottom: 10 })
      if (this.collectedLogs.length > 0) {
        ForEach(this.collectedLogs, (log: DevLogItem) => {
          Column() {
            Text(log.title).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold)
            Text(`${log.project_name || '项目'} · ${this.typeText(log.log_type)}`).fontColor(GameroTheme.textMuted).fontSize(11).margin({ top: 6 })
            Text(log.content || '暂无日志内容').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
          }.cardStyle().onClick(() => this.openDevLogDetail(log.id))
        })
      } else {
        this.EmptyCard('暂无收藏日志')
      }
    }.width('100%')
  }

  @Builder
  MyContentPage() {
    Column() {
      this.BackToFeatures()
      Text('我的日志 / 举报').sectionTitle()
      this.MessagePanel()
      Button('刷新我的内容')
        .width('100%')
        .height(40)
        .fontColor(GameroTheme.voidBg)
        .backgroundColor(GameroTheme.amber)
        .borderRadius(14)
        .margin({ bottom: 12 })
        .onClick(() => this.loadMyContent())

      Text('我的开发日志').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold).margin({ top: 4, bottom: 10 })
      if (this.myDevLogs.length > 0) {
        ForEach(this.myDevLogs, (log: DevLogItem) => {
          Column() {
            Text(log.title).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold)
            Text(`${log.project_name || '项目'} · ${this.typeText(log.log_type)} · ${log.created_at ? log.created_at.slice(0, 10) : '刚刚'}`)
              .fontColor(GameroTheme.textMuted)
              .fontSize(11)
              .margin({ top: 6 })
            Text(log.content || '暂无日志内容').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
          }.cardStyle().onClick(() => this.openDevLogDetail(log.id))
        })
      } else {
        this.EmptyCard('暂无我的开发日志')
      }

      Text('我的举报').fontColor(GameroTheme.textPrimary).fontSize(18).fontWeight(FontWeight.Bold).margin({ top: 16, bottom: 10 })
      if (this.myReports.length > 0) {
        ForEach(this.myReports, (report: ReportItem) => {
          Column() {
            Row() {
              Text(`${this.reportTargetText(report.target_type)} #${report.target_id}`)
                .fontColor(GameroTheme.textPrimary)
                .fontSize(14)
                .fontWeight(FontWeight.Bold)
                .layoutWeight(1)
              Text(this.reportStatusText(report.status)).fontColor(GameroTheme.amber).fontSize(11)
            }
            Text(report.reason || '未填写原因').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
            if (report.supplement && report.supplement.length > 0) {
              Text(report.supplement).fontColor(GameroTheme.textMuted).fontSize(11).lineHeight(16).margin({ top: 6 })
            }
            Text(`${report.created_at ? report.created_at.slice(0, 10) : '刚刚'}${report.handler_name ? ` · 处理人 ${report.handler_name}` : ''}`)
              .fontColor(GameroTheme.textMuted)
              .fontSize(11)
              .margin({ top: 8 })
          }.cardStyle()
        })
      } else {
        this.EmptyCard('暂无我的举报')
      }
    }.width('100%')
  }

  @Builder
  EmptyCard(text: string) {
    Column() {
      Text(text).fontColor(GameroTheme.textPrimary).fontSize(15).fontWeight(FontWeight.Bold)
      Text('如果这里为空，请先确认后端有对应数据。').fontColor(GameroTheme.textSecondary).fontSize(12).lineHeight(18).margin({ top: 8 })
    }.cardStyle()
  }

  @Builder
  FeaturePage() {
    Scroll() {
      Column() {
        if (this.featureView === 'posts') {
          this.PostListPage()
        } else if (this.featureView === 'postDetail') {
          this.PostDetailPage()
        } else if (this.featureView === 'logs') {
          this.DevLogListPage()
        } else if (this.featureView === 'logDetail') {
          this.DevLogDetailPage()
        } else if (this.featureView === 'notifications') {
          this.NotificationListPage()
        } else if (this.featureView === 'search') {
          this.SearchPage()
        } else if (this.featureView === 'talents') {
          this.TalentPage()
        } else if (this.featureView === 'userProfile') {
          this.UserProfilePage()
        } else if (this.featureView === 'settings') {
          this.SettingsPage()
        } else if (this.featureView === 'collections') {
          this.CollectionsPage()
        } else if (this.featureView === 'myContent') {
          this.MyContentPage()
        } else if (this.featureView === 'admin') {
          this.BackToFeatures()
          this.MessagePanel()
          this.AdminPanel()
        } else {
          Text('完整功能').sectionTitle()
          Text('点击已接真实接口的入口可查看真实后端数据。复杂详情、创建、编辑、审核页面仍需继续逐项实现为 ArkTS 原生页面。')
            .fontColor(GameroTheme.textSecondary)
            .fontSize(13)
            .lineHeight(20)
            .margin({ bottom: 14 })
          this.FeatureCard('动态流 / Feed', '对应 /feed，登录后从真实后端拉取动态流。', '已接真实接口', 'home')
          this.FeatureCard('社区帖子', '对应 /community，已接帖子列表真实接口。', '已接真实接口', 'posts')
          this.FeatureCard('项目发现', '对应 /projects、/p/:slug，项目列表已接真实接口。', '已接真实接口', 'projects')
          this.FeatureCard('项目创建/编辑', '对应 /projects/new、/p/:slug/edit，移动端已接创建、编辑、封面、截图、视频上传。', '已接真实接口', 'projects')
          this.FeatureCard('项目成员', '对应 /project/:id/members，已接添加、改角色、移除、退出。', '已接真实接口', 'projects')
          this.FeatureCard('项目版本', '对应 /project/:id/releases，已接版本发布和版本记录。', '已接真实接口', 'projects')
          this.FeatureCard('项目任务看板', '对应 /projects/:id/tasks，已接任务列表真实接口。', '已接真实接口', 'projects')
          this.FeatureCard('开发日志', '对应 /devlogs，已接开发日志列表和发布接口。', '已接真实接口', 'logs')
          this.FeatureCard('招募协作', '对应 /recruit、/recruit/:id，招募列表已接真实接口。', '已接真实接口', 'recruit')
          this.FeatureCard('招募申请/编辑', '对应 /recruit/:id/edit、我的申请，已接详情、申请和 CRUD。', '已接真实接口', 'recruit')
          this.FeatureCard('人才市场', '对应 /talent，已接真实人才列表。', '已接真实接口', 'talents')
          this.FeatureCard('搜索发现', '对应 /search，已接真实搜索接口。', '已接真实接口', 'search')
          this.FeatureCard('通知中心', '对应 /notifications，已接通知列表和未读数接口。', '已接真实接口', 'notifications')
          this.FeatureCard('个人主页', '对应 /u/:id，已接真实用户主页和关注。', '已接真实接口', 'talents')
          this.FeatureCard('账号设置', '对应 /settings，已接资料和合作状态设置。', '已接真实接口', 'settings')
          this.FeatureCard('我的收藏', '对应 /me/collections/projects、posts、logs，已接三类收藏。', '已接真实接口', 'collections')
          this.FeatureCard('我的日志/举报', '对应 /me/logs、/me/reports，已接真实列表。', '已接真实接口', 'myContent')
          if (this.currentUser && (this.currentUser.role === 'admin' || this.currentUser.role === 'superadmin')) {
            this.FeatureCard('管理后台', '对应 /admin，已接统计、用户治理、举报处理和帖子/日志隐藏。', '已接审核', 'admin')
          }
        }
      }.pageContent()
    }.width('100%').height('100%').scrollBar(BarState.Auto)
  }

  @Builder
  HomePage() {
    Scroll() {
      Column() {
        this.Hero()
        this.MessagePanel()
        this.QuickWorkspace()
        this.HomeProjectPreview()
        this.HomeRecruitPreview()
        this.SectionHeader('最新动态', '查看社区', 'posts')
        if (this.feeds.length > 0) {
          ForEach(this.feeds, (item: FeedCardModel) => { this.FeedCard(item) })
        } else {
          Column() {
            Text('暂无真实动态数据').fontColor(GameroTheme.textPrimary).fontSize(16).fontWeight(FontWeight.Bold)
            Text('请确认后端 /api/v1/feed 有数据，并且当前账号已登录。这里不会再使用演示动态数据。')
              .fontColor(GameroTheme.textSecondary)
              .fontSize(12)
              .lineHeight(18)
              .margin({ top: 8 })
          }.cardStyle()
        }
      }.pageContent()
    }.width('100%').height('100%').scrollBar(BarState.Auto)
  }

  @Builder
  ProjectsPage() {
    Scroll() {
      Column() {
        if (this.selectedProject) {
          this.ProjectDetailPage()
        } else {
          Text('发现项目').sectionTitle()
          this.MessagePanel()
          this.ProjectComposer()
          if (this.projectItems.length > 0) {
            ForEach(this.projectItems, (item: ProjectItem) => { this.ProjectItemCard(item) })
          } else {
            this.EmptyCard('暂无真实项目数据')
          }
        }
      }.pageContent()
    }.width('100%').height('100%').scrollBar(BarState.Auto)
  }

  @Builder
  RecruitPage() {
    Scroll() {
      Column() {
        if (this.selectedRecruit) {
          this.RecruitDetailPage()
        } else {
          Text('招募协作').sectionTitle()
          this.MessagePanel()
          this.RecruitmentForm('创建招募', false)
          if (this.recruitItems.length > 0) {
            ForEach(this.recruitItems, (item: RecruitItem) => { this.RecruitItemCard(item) })
          } else {
            this.EmptyCard('暂无真实招募数据')
          }
          this.MyApplicationsPanel()
        }
      }.pageContent()
    }.width('100%').height('100%').scrollBar(BarState.Auto)
  }

  @Builder
  ProfilePage() {
    Scroll() {
      Column() {
        Text('我的 Gamero').sectionTitle()
        if (this.currentUser) {
          this.MessagePanel()
          this.AccountPanel()
          if (this.currentUser.role === 'admin' || this.currentUser.role === 'superadmin') {
            this.AdminPanel()
          }
        } else {
          this.AuthPanel()
        }
      }.pageContent()
    }.width('100%').height('100%').scrollBar(BarState.Auto)
  }

  @Builder
  BottomNav() {
    Row() {
      this.NavItem(0, '动态')
      this.NavItem(1, '项目')
      this.NavItem(2, '招募')
      this.NavItem(3, '功能')
      this.NavItem(4, '我的')
    }
    .width('100%')
    .padding({ left: 12, right: 12, top: 8, bottom: 10 })
    .backgroundColor('#F21F1F28')
    .border({ width: { top: 1 }, color: GameroTheme.borderVisible })
  }

  @Builder
  NavItem(index: number, title: string) {
    Column() {
      Text(this.tabIcon(index))
        .fontColor(this.currentTab === index ? '#16161D' : GameroTheme.textMuted)
        .fontSize(12)
        .fontWeight(FontWeight.Bold)
        .width(28)
        .height(24)
        .textAlign(TextAlign.Center)
        .backgroundColor(this.currentTab === index ? GameroTheme.amber : '#262A34')
        .borderRadius(6)
      Text(title).fontColor(this.currentTab === index ? GameroTheme.textPrimary : GameroTheme.textMuted).fontSize(11).margin({ top: 4 })
    }
    .layoutWeight(1)
    .height(52)
    .justifyContent(FlexAlign.Center)
    .onClick(() => this.selectTab(index))
  }

  build() {
    if (!this.currentUser) {
      Column() {
        this.LogoHeader()
        Scroll() {
          Column() {
            this.AuthPanel()
          }.pageContent()
        }.layoutWeight(1).width('100%').scrollBar(BarState.Auto)
      }
      .width('100%')
      .height('100%')
      .backgroundColor(GameroTheme.voidBg)
    } else {
      Column() {
        this.LogoHeader()
        Stack() {
          if (this.currentTab === 0) {
            this.HomePage()
          } else if (this.currentTab === 1) {
            this.ProjectsPage()
          } else if (this.currentTab === 2) {
            this.RecruitPage()
          } else if (this.currentTab === 3) {
            this.FeaturePage()
          } else {
            this.ProfilePage()
          }
        }.layoutWeight(1).width('100%')
        this.BottomNav()
      }
      .width('100%')
      .height('100%')
      .backgroundColor(GameroTheme.voidBg)
    }
  }
}

@Extend(Column)
function cardStyle() {
  .width('100%')
  .padding(16)
  .backgroundColor(GameroTheme.card)
  .borderRadius(8)
  .border({ width: 1, color: GameroTheme.borderSubtle })
  .margin({ bottom: 12 })
}

@Extend(Column)
function pageContent() {
  .width('100%')
  .padding({ left: 16, right: 16, top: 12, bottom: 16 })
}

@Extend(Text)
function sectionTitle() {
  .fontColor(GameroTheme.textPrimary)
  .fontSize(21)
  .fontWeight(FontWeight.Bold)
  .margin({ top: 10, bottom: 12 })
}

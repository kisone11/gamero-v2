import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthLayout } from '@/components/layout/auth-layout'
import { AppLayout } from '@/components/layout/app-layout'
import { AdminRoute } from '@/components/layout/admin-route'
import { ToastContainer } from '@/components/ui/toast'
import { ErrorBoundary } from '@/components/ui/error-boundary'

import FeedPage from '@/pages/feed'
import LoginPage from '@/pages/login'
import RegisterPage from '@/pages/register'
import ForgotPasswordPage from '@/pages/forgot-password'
import ProfilePage from '@/pages/profile'
import PostDetailPage from '@/pages/post-detail'
import DevLogDetailPage from '@/pages/devlog-detail'
import CommunityPage from '@/pages/community'
import ProjectsPage from '@/pages/projects'
import SettingsPage from '@/pages/settings'
import ProjectDetailPage from '@/pages/project-detail'
import NotificationsPage from '@/pages/notifications'
import SearchPage from '@/pages/search'
import DevLogsPage from '@/pages/devlogs'
import CreateProjectPage from '@/pages/create-project'
import EditProjectPage from '@/pages/edit-project'
import CreateDevLogPage from '@/pages/create-devlog'
import EditDevLogPage from '@/pages/edit-devlog'
import ProjectMembersPage from '@/pages/project-members'
import ProjectReleasesPage from '@/pages/project-releases'
import RecruitPage from '@/pages/recruit'
import RecruitDetailPage from '@/pages/recruit-detail'
import CreateRecruitPage from '@/pages/create-recruit'
import TalentPage from '@/pages/talent'
import CollaborationCenterPage from '@/pages/collaboration-center'
import MyApplicationsPage from '@/pages/my-applications'
import MyInvitationsPage from '@/pages/my-invitations'
import MyLogsPage from '@/pages/my-logs'
import MyReportsPage from '@/pages/my-reports'
import EditPostPage from '@/pages/edit-post'
import AdminPage from '@/pages/admin'
import NotFoundPage from '@/pages/not-found'

export default function App() {
  return (
    <>
    <ToastContainer />
    <ErrorBoundary>
    <Routes>
      <Route element={<AuthLayout />}>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
      </Route>

      <Route element={<AppLayout />}>
        <Route path="/" element={<Navigate to="/feed" replace />} />
        <Route path="/feed" element={<FeedPage />} />
        <Route path="/community" element={<CommunityPage />} />
        <Route path="/projects" element={<ProjectsPage />} />
        <Route path="/p/:slug" element={<ProjectDetailPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/u/:id" element={<ProfilePage />} />
        <Route path="/post/:id" element={<PostDetailPage />} />
        <Route path="/devlog/:id" element={<DevLogDetailPage />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/devlogs" element={<DevLogsPage />} />
        <Route path="/recruit" element={<RecruitPage />} />
        <Route path="/recruit/new" element={<CreateRecruitPage />} />
        <Route path="/recruit/:id" element={<RecruitDetailPage />} />
        <Route path="/talent" element={<TalentPage />} />
        <Route path="/me/collaboration" element={<CollaborationCenterPage />} />
        <Route path="/me/applications" element={<MyApplicationsPage />} />
        <Route path="/me/invitations" element={<MyInvitationsPage />} />
        <Route path="/notifications" element={<NotificationsPage />} />
        <Route path="/project/:id/members" element={<ProjectMembersPage />} />
        <Route path="/project/:id/releases" element={<ProjectReleasesPage />} />
        <Route path="/me/logs" element={<MyLogsPage />} />
        <Route path="/me/reports" element={<MyReportsPage />} />
        <Route path="/projects/new" element={<CreateProjectPage />} />
        <Route path="/p/:slug/edit" element={<EditProjectPage />} />
        <Route path="/p/:slug/recruit/new" element={<CreateRecruitPage />} />
        <Route path="/recruit/:id/edit" element={<CreateRecruitPage />} />
        <Route path="/projects/:projectId/logs/new" element={<CreateDevLogPage />} />
        <Route path="/devlog/:id/edit" element={<EditDevLogPage />} />
        <Route path="/post/:id/edit" element={<EditPostPage />} />
        <Route path="/admin" element={<AdminRoute><AdminPage /></AdminRoute>} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
    </ErrorBoundary>
    </>
  )
}

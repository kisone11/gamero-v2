import http, { extractData } from '@/lib/http'
import type {
  ApiResponse,
  AuthResponse,
  SendEmailCodeReq,
  RegisterByEmailReq,
  LoginByEmailReq,
  RefreshTokenReq,
  ChangePasswordReq,
  ResetPasswordByCodeReq,
  DeleteAccountReq,
} from '@/types/api'

export const authApi = {
  sendEmailCode: (data: SendEmailCodeReq) =>
    http.post<ApiResponse<null>>('/auth/email/code', data).then(extractData),

  register: (data: RegisterByEmailReq) =>
    http.post<ApiResponse<AuthResponse>>('/auth/register/email', data).then(extractData),

  login: (data: LoginByEmailReq) =>
    http.post<ApiResponse<AuthResponse>>('/auth/login/email', data).then(extractData),

  refresh: (data: RefreshTokenReq) =>
    http.post<ApiResponse<AuthResponse>>('/auth/refresh', data).then(extractData),

  logout: () =>
    http.post<ApiResponse<null>>('/auth/logout').then(extractData),

  changePassword: (data: ChangePasswordReq) =>
    http.post<ApiResponse<null>>('/auth/password/change', data).then(extractData),

  resetPasswordByCode: (data: ResetPasswordByCodeReq) =>
    http.post<ApiResponse<null>>('/auth/password/reset', data).then(extractData),

  deleteAccount: (data: DeleteAccountReq) =>
    http.delete<ApiResponse<null>>('/auth/account', { data }).then(extractData),
}

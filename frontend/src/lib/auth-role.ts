import { tokenStorage } from '@/lib/http'
import type { UserRole } from '@/types/api'

interface AccessTokenPayload {
  role?: UserRole
  exp?: number
}

function decodeAccessTokenPayload(): AccessTokenPayload | null {
  const token = tokenStorage.getAccess()
  const payload = token.split('.')[1]
  if (!payload) return null

  try {
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/')
    const json = decodeURIComponent(
      atob(normalized)
        .split('')
        .map((char) => `%${char.charCodeAt(0).toString(16).padStart(2, '0')}`)
        .join(''),
    )
    return JSON.parse(json) as AccessTokenPayload
  } catch {
    return null
  }
}

export function getAccessTokenRole(): UserRole | null {
  const payload = decodeAccessTokenPayload()
  if (!payload?.role) return null
  if (payload.exp && payload.exp * 1000 <= Date.now()) return null
  return payload.role
}

export function isAdminRole(role?: string | null) {
  return role === 'admin' || role === 'superadmin'
}

import type { User } from '@/api'
import { StudyTaskApi } from '@/api'
import { unicodeLength } from '@/utils/text'

export function routeForUser(user: User, nicknameRequired = false): string {
  if (nicknameRequired || !user.nickname || unicodeLength(user.nickname) < 2) return '/pages/nickname/nickname'
  if (user.onboarding_status === 'not_started' || user.onboarding_version < 1) return '/pages/onboarding/onboarding'
  return '/pages/checkin/checkin'
}

export async function routeForAuthenticatedUser(user: User, nicknameRequired = false): Promise<string> {
  const fallback = routeForUser(user, nicknameRequired)
  if (fallback !== '/pages/checkin/checkin') return fallback

  await StudyTaskApi.compensateMidnight().catch(() => undefined)
  const activeTask = await StudyTaskApi.active().catch(() => null)
  return activeTask ? `/pages/task/task?id=${activeTask.id}` : fallback
}

import type { User } from '@/api'
import { unicodeLength } from '@/utils/text'

export function routeForUser(user: User, nicknameRequired = false): string {
  if (nicknameRequired || !user.nickname || unicodeLength(user.nickname) < 2) return '/pages/nickname/nickname'
  if (user.onboarding_status === 'not_started' || user.onboarding_version < 1) return '/pages/onboarding/onboarding'
  return '/pages/checkin/checkin'
}

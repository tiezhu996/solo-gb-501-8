import { useAuthState, authStore } from '../stores/authStore'
import { can } from '../utils/permissions'

export function useAuth() {
  const state = useAuthState()
  return {
    ...state,
    login: authStore.login,
    logout: authStore.logout,
    refresh: authStore.refresh,
    can: (permission: string) => can(state.user, permission),
  }
}


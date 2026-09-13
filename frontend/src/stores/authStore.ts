import { useSyncExternalStore } from 'react'
import { authAPI } from '../api'
import type { User } from '../types/domain'

interface AuthState { user: User | null; loading: boolean; initialized: boolean }

function cachedUser(): User | null {
  try { return JSON.parse(localStorage.getItem('sterile.user') || 'null') as User | null }
  catch {
    localStorage.removeItem('sterile.user')
    localStorage.removeItem('sterile.accessToken')
    return null
  }
}

let state: AuthState = {
	user: cachedUser(),
  loading: false,
	initialized: !localStorage.getItem('sterile.accessToken'),
}
const listeners = new Set<() => void>()
const emit = () => listeners.forEach((listener) => listener())
const setState = (next: Partial<AuthState>) => { state = { ...state, ...next }; emit() }
let refreshPromise: Promise<void> | null = null

export const authStore = {
  subscribe: (listener: () => void) => { listeners.add(listener); return () => listeners.delete(listener) },
  getSnapshot: () => state,
  async login(username: string, password: string) {
    setState({ loading: true })
    try {
      const result = await authAPI.login(username, password)
      localStorage.setItem('sterile.accessToken', result.token)
      localStorage.setItem('sterile.user', JSON.stringify(result.user))
      setState({ user: result.user })
    } finally { setState({ loading: false }) }
  },
  async refresh() {
	if (refreshPromise) return refreshPromise
	if (!localStorage.getItem('sterile.accessToken')) { setState({ initialized: true, user: null }); return }
	refreshPromise = (async () => {
	  setState({ loading: true })
	  try {
		const user = await authAPI.me()
		localStorage.setItem('sterile.user', JSON.stringify(user))
		setState({ user })
	  } catch {
		localStorage.removeItem('sterile.user')
		localStorage.removeItem('sterile.accessToken')
		setState({ user: null })
	  } finally {
		setState({ loading: false, initialized: true })
		refreshPromise = null
	  }
	})()
	return refreshPromise
  },
  logout() {
    localStorage.removeItem('sterile.accessToken')
    localStorage.removeItem('sterile.user')
	setState({ user: null, initialized: true })
  },
}

export const useAuthState = () => useSyncExternalStore(authStore.subscribe, authStore.getSnapshot)

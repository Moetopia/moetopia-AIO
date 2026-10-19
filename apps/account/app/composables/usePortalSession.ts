import { inject, type InjectionKey, type Ref } from 'vue'
import type { PortalContext, PortalSessions } from '@moetopia/api-client'
import type { Locale, accountMessages } from '@moetopia/locales'

const destinations = new Set(['/profile', '/sessions', '/security', '/developer', '/developer/applications', '/developer/docs'])
export const portalLoginURL = (destination: string) => `/api/v1/portal/login?return_to=${encodeURIComponent(destinations.has(destination) ? destination : '/profile')}`

export const usePortalSession = (locale: Ref<Locale>) => {
  const route = useRoute()
  const context = ref<PortalContext>()
  const sessionList = ref<PortalSessions>()
  const sessions = computed(() => sessionList.value?.items ?? [])
  const sessionsLoading = ref(false)
  const loading = ref(true)
  const busy = ref(false)
  const navigating = ref(false)
  const feedback = ref<keyof typeof accountMessages['en']>()
  const authenticated = computed(() => context.value?.authenticated === true && !!context.value.user)
  let request: Promise<void> | undefined
  const headers = computed(() => ({ 'X-Moetopia-CSRF': context.value?.csrf_token ?? '' }))
  const formatTime = (value: string) => new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
  const explain = (error: unknown) => {
    const status = error && typeof error === 'object' && 'statusCode' in error ? error.statusCode : undefined
    feedback.value = status === 401 ? 'sessionExpired' : status === 403 ? 'forbidden' : status === 429 ? 'limited' : 'unavailable'
    if (status === 401) { context.value = undefined; sessionList.value = undefined }
  }
  const connect = async () => {
    if (request) return request
    loading.value = true
    feedback.value = undefined
    request = (async () => {
      try { context.value = await $fetch<PortalContext>('/api/v1/portal/context') }
      catch (error) { context.value = undefined; explain(error) }
      finally { loading.value = false; request = undefined }
    })()
    return request
  }
  const startLogin = (destination = route.path) => {
    if (navigating.value || busy.value) return
    navigating.value = true
    window.location.assign(portalLoginURL(destination))
  }
  const loadSessions = async () => {
    if (sessionsLoading.value) return
    sessionsLoading.value = true
    feedback.value = undefined
    try { sessionList.value = await $fetch<PortalSessions>('/api/v1/portal/sessions') }
    catch (error) { sessionList.value = undefined; explain(error) }
    finally { sessionsLoading.value = false }
  }
  const logout = async () => {
    if (busy.value) return
    busy.value = true
    feedback.value = undefined
    try {
      const result = await $fetch<PortalContext>('/api/v1/portal/logout', { method: 'POST', headers: headers.value })
      await navigateTo('/auth/logout')
      context.value = result
      sessionList.value = undefined
    } catch (error) { explain(error) }
    finally { busy.value = false }
  }
  const revoke = async (id: string) => {
    if (busy.value) return
    busy.value = true
    feedback.value = undefined
    try {
      await $fetch(`/api/v1/portal/sessions/${encodeURIComponent(id)}`, { method: 'DELETE', headers: headers.value })
      if (id === sessionList.value?.current_id) { await connect(); sessionList.value = undefined }
      else await loadSessions()
    } catch (error) { explain(error) }
    finally { busy.value = false }
  }
  const identityRoute = computed(() => route.path.startsWith('/auth/') || route.path === '/' && route.query.authorization_request !== undefined)
  onMounted(() => { if (!identityRoute.value) void connect() })
  watch(identityRoute, active => { if (!active) void connect() })
  return { context, sessionList, sessions, sessionsLoading, authenticated, loading, busy, navigating, feedback, formatTime, connect, startLogin, loadSessions, logout, revoke }
}

export const portalSessionKey: InjectionKey<ReturnType<typeof usePortalSession>> = Symbol('portal-session')
export const usePortalContext = () => {
  const context = inject(portalSessionKey)
  if (!context) throw new Error('Portal session provider is required')
  return context
}

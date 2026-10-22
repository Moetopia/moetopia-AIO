import type { AdminContext } from '@moetopia/api-client'
import { adminMessages } from '@moetopia/locales'
import type { InjectionKey } from 'vue'

export const adminSessionKey: InjectionKey<ReturnType<typeof useAdminSession>> = Symbol('admin-session')
export const useAdminSession = () => {
  const { locale } = usePortalPreferences()
  const text = computed(() => adminMessages[locale.value])
  const context = shallowRef<AdminContext>()
  const loading = ref(true)
  const headers = computed(() => ({ 'X-Moetopia-CSRF': context.value?.csrf_token ?? '' }))
  const accept = (value: AdminContext) => { const { recovery_codes: _, ...safe } = value; context.value = safe }
  const connect = async () => { if (!context.value) loading.value = true; try { accept(await $fetch<AdminContext>('/api/v1/admin/context', { retry: 0 })) } finally { loading.value = false } }
  const request = () => {
  const error = ref('')
  const busy = ref(false)
  watch(locale, (current, previous) => {
    const key = (Object.keys(adminMessages[previous]) as (keyof typeof adminMessages.en)[]).find(key => adminMessages[previous][key] === error.value)
    if (key) error.value = adminMessages[current][key]
  })
  const run = async (action: () => Promise<void>, refresh?: () => Promise<void>) => {
    if (busy.value) return false
    busy.value = true; error.value = ''
    try { await action(); return true }
    catch (cause) {
      const failure = cause as { statusCode?: number; data?: { code?: string } }
      const code = failure.data?.code
      if (code === 'reauthentication_failed') error.value = text.value.wrongPassword
      else if (code === 'admin_last_administrator') error.value = text.value.lastAdmin
      else if (code === 'admin_mfa_required' || failure.statusCode === 401) { try { await connect(); error.value = text.value.invalid } catch { error.value = text.value.error } }
      else if (failure.statusCode === 409) { try { await (refresh ?? connect)(); error.value = text.value.conflict } catch { error.value = text.value.error } }
      else error.value = failure.statusCode === 403 ? text.value.forbidden : failure.statusCode === 422 ? text.value.invalid : failure.statusCode === 429 ? text.value.limited : text.value.error
      return false
    } finally { busy.value = false }
  }
  return { error, busy, run }
  }
  const { run } = request()
  const login = () => window.location.assign('/api/v1/admin/login?return_to=%2F')
  const logout = (central = false) => run(async () => {
    if (central) {
      const result = await $fetch<{ redirect_to: string }>('/api/v1/admin/central-logout', { method: 'POST', headers: headers.value, retry: 0 })
      context.value = undefined; window.location.assign(result.redirect_to)
    } else { accept(await $fetch<AdminContext>('/api/v1/admin/logout', { method: 'POST', headers: headers.value, retry: 0 })) }
  })
  return { context, loading, headers, text, accept, connect, request, login, logout }
}
export const useAdminContext = () => { const session = inject(adminSessionKey); if (!session) throw new Error('Administrator session provider is missing'); return { ...session, ...session.request() } }

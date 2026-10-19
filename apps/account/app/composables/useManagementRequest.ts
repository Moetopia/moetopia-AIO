import { managementMessages } from '@moetopia/locales'

export const useManagementRequest = () => {
  const { locale } = usePortalPreferences()
  const { context, connect } = usePortalContext()
  const text = computed(() => managementMessages[locale.value])
  const busy = ref(false)
  const error = ref('')
  const success = ref(false)
  const headers = computed(() => ({ 'X-Moetopia-CSRF': context.value?.csrf_token ?? '' }))
  const run = async (action: () => Promise<void>, refresh?: () => Promise<void>) => {
    if (busy.value) return false
    busy.value = true; error.value = ''; success.value = false
    try { await action(); success.value = true; return true }
    catch (cause) {
      const failure = cause as { statusCode?: number; data?: { code?: string } }
      const status = failure.statusCode
      if (status === 409) { try { await (refresh ?? connect)(); error.value = text.value.conflict } catch { error.value = text.value.unavailable } }
      else if (status === 403) { await connect(); error.value = text.value.forbidden }
      else if (status === 401 && failure.data?.code === 'reauthentication_failed') error.value = text.value.wrongPassword
      else if (status === 401) { await navigateTo('/auth/login'); context.value = undefined }
      else error.value = status === 422 ? text.value.invalid : status === 404 ? text.value.notFound : status === 429 ? text.value.limited : text.value.unavailable
      return false
    } finally { busy.value = false }
  }
  const signedOut = async () => { await navigateTo('/auth/login'); context.value = undefined }
  return { text, busy, error, success, headers, run, signedOut }
}

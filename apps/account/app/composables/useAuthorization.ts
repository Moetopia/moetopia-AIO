import type { components } from '@moetopia/api-client'
import type { authorizationMessages } from '@moetopia/locales'

type Details = components['schemas']['AuthorizationOutputBody']
type Decision = components['schemas']['AuthorizationDecisionInputBody']
type Result = components['schemas']['AuthorizationDecisionOutputBody']
type Message = keyof typeof authorizationMessages['en']

export const useAuthorization = () => {
  const route = useRoute()
  const { account, loading: accountLoading, authenticationVersion, connect, logout, changeView, busy: accountBusy } = useAccountContext()
  const requestId = computed(() => typeof route.query.authorization_request === 'string' && /^[A-Za-z0-9_-]{43}$/.test(route.query.authorization_request) ? route.query.authorization_request : '')
  const details = ref<Details>()
  const loading = ref(true)
  const submitting = ref(false)
  const redirecting = ref(false)
  const feedback = ref<Message>()
  const forceLogin = ref(false)
  const now = ref(Date.now())
  let controller: AbortController | undefined
  let timer: ReturnType<typeof setInterval> | undefined
  const expired = computed(() => !!details.value && Date.parse(details.value.expires_at) <= now.value)
  const needsLogin = computed(() => forceLogin.value || !!details.value?.login_required || !details.value?.user)
  const ready = computed(() => !!details.value?.user && !needsLogin.value && !loading.value && !submitting.value && !accountBusy.value && !expired.value && !feedback.value && !redirecting.value)
  const errorMessage = (error: unknown): Message => {
    const status = error && typeof error === 'object' && 'statusCode' in error ? error.statusCode : undefined
    return status === 404 ? 'expired' : status === 403 ? 'forbidden' : status === 409 ? 'changed' : status === 422 ? 'invalid' : status === 429 ? 'limited' : 'unavailable'
  }
  const reload = async (notice?: Message) => {
    controller?.abort()
    const current = new AbortController()
    controller = current
    details.value = undefined
    feedback.value = undefined
    loading.value = true
    forceLogin.value = false
    if (!requestId.value) { feedback.value = 'invalid'; loading.value = false; return }
    try {
      details.value = await $fetch<Details>(`/api/v1/account/authorizations/${requestId.value}`, { signal: current.signal })
      now.value = Date.now()
      feedback.value = notice
    } catch (error) {
      if (!current.signal.aborted) feedback.value = errorMessage(error)
    } finally { if (!current.signal.aborted) loading.value = false }
  }
  const decide = async (decision: Decision['decision']) => {
    if (!ready.value || !details.value?.user) return
    const id = requestId.value
    const body: Decision = { decision, subject_id: details.value.user.id }
    submitting.value = true
    try {
      const response = await $fetch<Result>(`/api/v1/account/authorizations/${id}`, { method: 'POST', headers: { 'X-Moetopia-CSRF': account.value?.csrf_token ?? '' }, body })
      if (id !== requestId.value) return
      const destination = new URL(response.redirect_to)
      if (!['http:', 'https:'].includes(destination.protocol)) throw new Error('Invalid authorization destination')
      redirecting.value = true
      window.location.assign(response.redirect_to)
    } catch (error) {
      if (id !== requestId.value) return
      const status = error && typeof error === 'object' && 'statusCode' in error ? error.statusCode : undefined
      if (status === 401) {
        await connect()
        forceLogin.value = true
        changeView('login')
      } else if (status === 409) {
        await connect()
        await reload('changed')
      } else feedback.value = errorMessage(error)
    } finally { submitting.value = false }
  }
  const switchAccount = async () => {
    if (accountBusy.value || submitting.value) return
    await logout()
    await reload()
    changeView('login')
  }
  const acknowledge = () => { if (feedback.value === 'changed') feedback.value = undefined }
  watch([requestId, accountLoading, authenticationVersion], () => { if (!accountLoading.value) void reload() }, { immediate: true })
  onMounted(() => { timer = setInterval(() => { now.value = Date.now() }, 1000) })
  onBeforeUnmount(() => { controller?.abort(); if (timer) clearInterval(timer) })
  return { details, loading, submitting, redirecting, feedback, expired, needsLogin, ready, forceLogin, reload, decide, switchAccount, acknowledge }
}

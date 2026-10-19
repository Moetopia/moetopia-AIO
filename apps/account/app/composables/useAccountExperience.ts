import { inject, type InjectionKey, type Ref } from 'vue'
import type { AccountContext, LoginRequest, RegistrationRequest, VerificationRequest, VerificationResponse } from '@moetopia/api-client'
import type { accountMessages, Locale } from '@moetopia/locales'
import { emailLocale } from '@moetopia/locales'

export const useAccountExperience = (locale: Ref<Locale>) => {
  const route = useRoute()
  const loginDestination = computed(() => route.query.continue === 'developer-applications' ? '/developer/applications' : '/profile')
  const loginQuery = computed(() => route.query.continue === 'developer-applications' ? { continue: 'developer-applications' } : {})
  const account = ref<AccountContext>()
  const view = ref<'login' | 'register'>(route.path === '/auth/register' ? 'register' : 'login')
  const email = ref('')
  const password = ref('')
  const displayName = ref('')
  const invitation = ref('')
  const code = ref('')
  const challenge = ref('')
  const verificationEmail = ref('')
  const busy = ref(false)
  const loading = ref(true)
  const authenticationVersion = ref(0)
  const feedback = ref<keyof typeof accountMessages['en']>()
  const success = ref(false)
  const feedbackElement = ref<HTMLElement>()
  const headers = computed(() => ({ 'X-Moetopia-CSRF': account.value?.csrf_token ?? '' }))
  const formatTime = (value: string) => new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
  const explain = (error: unknown, registration = false) => {
    const status = error && typeof error === 'object' && 'statusCode' in error ? error.statusCode : undefined
    feedback.value = status === 429 ? 'limited' : status === 403 ? 'forbidden' : status === 401 ? (account.value?.user ? 'sessionExpired' : 'invalidCredentials') : status === 422 && registration ? 'invalidRegistration' : 'unavailable'
    success.value = false
  }
  const connect = async () => {
    loading.value = true
    feedback.value = undefined
    try { account.value = await $fetch<AccountContext>('/api/v1/account/context') }
    catch (error) { explain(error) }
    finally { loading.value = false }
  }
  const perform = async (action: () => Promise<void>, registration = false) => {
    if (busy.value) return
    busy.value = true
    feedback.value = undefined
    try { await action() }
    catch (error) { explain(error, registration) }
    finally { busy.value = false; await nextTick(); if (feedback.value) feedbackElement.value?.focus() }
  }
  const login = () => perform(async () => {
    const body: LoginRequest = { email: email.value, password: password.value }
    const response = await $fetch<AccountContext>('/api/v1/account/login', { method: 'POST', headers: headers.value, body })
    password.value = ''
    account.value = response
    authenticationVersion.value++
    if (route.query.authorization_request === undefined) window.location.assign(portalLoginURL(loginDestination.value))
  })
  const sendCode = () => perform(async () => {
    const body: VerificationRequest = { email: email.value, locale: emailLocale(locale.value) }
    const response = await $fetch<VerificationResponse>('/api/v1/account/verification', { method: 'POST', headers: headers.value, body })
    challenge.value = response.challenge_id; code.value = ''; verificationEmail.value = email.value; feedback.value = 'sent'; success.value = true
  }, true)
  const register = () => perform(async () => {
    const body: RegistrationRequest = { challenge_id: challenge.value, code: code.value, invitation: invitation.value, display_name: displayName.value, password: password.value }
    await $fetch('/api/v1/account/register', { method: 'POST', headers: headers.value, body })
    password.value = ''; invitation.value = ''; code.value = ''; challenge.value = ''; view.value = 'login'; feedback.value = 'registered'; success.value = true
    if (route.query.authorization_request === undefined && route.path.startsWith('/auth/')) await navigateTo({ path: '/auth/login', query: loginQuery.value })
  }, true)
  const logout = () => perform(async () => { account.value = await $fetch<AccountContext>('/api/v1/account/logout', { method: 'POST', headers: headers.value }) })
  const changeView = (value: string | undefined) => {
    if (value !== 'login' && value !== 'register') return
    view.value = value; feedback.value = undefined; password.value = ''
    if (route.query.authorization_request === undefined) void navigateTo({ path: `/auth/${value}`, query: loginQuery.value })
  }
  watch(() => route.path, path => { if (path.startsWith('/auth/')) { view.value = path === '/auth/register' ? 'register' : 'login'; password.value = '' } })
  watch(email, () => { if (email.value !== verificationEmail.value) { challenge.value = ''; code.value = '' } })
  const identityRoute = computed(() => route.path.startsWith('/auth/') || route.path === '/' && route.query.authorization_request !== undefined)
  onMounted(() => { if (identityRoute.value) void connect() })
  watch(identityRoute, active => { if (active) void connect() })
  return { account, view, email, password, displayName, invitation, code, challenge, busy, loading, authenticationVersion, feedback, success, feedbackElement, formatTime, connect, perform, login, sendCode, register, logout, changeView }
}

export const accountExperienceKey: InjectionKey<ReturnType<typeof useAccountExperience>> = Symbol('account-experience')

export const useAccountContext = () => {
  const context = inject(accountExperienceKey)
  if (!context) throw new Error('Account experience provider is required')
  return context
}

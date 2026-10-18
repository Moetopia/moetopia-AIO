import { inject, type InjectionKey, type Ref } from 'vue'
import type { AccountContext, AccountSessions, LoginRequest, RegistrationRequest, VerificationRequest, VerificationResponse } from '@moetopia/api-client'
import type { accountMessages, Locale } from '@moetopia/locales'

export const useAccountExperience = (locale: Ref<Locale>) => {
  const account = ref<AccountContext>()
  const sessionList = ref<AccountSessions>()
  const sessions = computed(() => sessionList.value?.items ?? [])
  const view = ref<'login' | 'register'>('login')
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
  const loadSessions = async () => { sessionList.value = await $fetch<AccountSessions>('/api/v1/account/sessions') }
  const connect = async () => {
    loading.value = true
    feedback.value = undefined
    try { account.value = await $fetch<AccountContext>('/api/v1/account/context'); if (account.value.user) await loadSessions() }
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
    account.value = await $fetch<AccountContext>('/api/v1/account/login', { method: 'POST', headers: headers.value, body })
    password.value = ''
    authenticationVersion.value++
    await loadSessions()
  })
  const sendCode = () => perform(async () => {
    const body: VerificationRequest = { email: email.value, locale: locale.value }
    const response = await $fetch<VerificationResponse>('/api/v1/account/verification', { method: 'POST', headers: headers.value, body })
    challenge.value = response.challenge_id; code.value = ''; verificationEmail.value = email.value; feedback.value = 'sent'; success.value = true
  }, true)
  const register = () => perform(async () => {
    const body: RegistrationRequest = { challenge_id: challenge.value, code: code.value, invitation: invitation.value, display_name: displayName.value, password: password.value }
    await $fetch('/api/v1/account/register', { method: 'POST', headers: headers.value, body })
    password.value = ''; invitation.value = ''; code.value = ''; challenge.value = ''; view.value = 'login'; feedback.value = 'registered'; success.value = true
  }, true)
  const logout = () => perform(async () => { account.value = await $fetch<AccountContext>('/api/v1/account/logout', { method: 'POST', headers: headers.value }); sessionList.value = undefined })
  const revoke = (id: string) => perform(async () => {
    await $fetch(`/api/v1/account/sessions/${encodeURIComponent(id)}`, { method: 'DELETE', headers: headers.value })
    if (id === sessionList.value?.current_id) await connect()
    else await loadSessions()
  })
  const changeView = (value: string | undefined) => { if (value === 'login' || value === 'register') { view.value = value; feedback.value = undefined; password.value = '' } }
  watch(email, () => { if (email.value !== verificationEmail.value) { challenge.value = ''; code.value = '' } })
  onMounted(connect)
  return { account, sessionList, sessions, view, email, password, displayName, invitation, code, challenge, busy, loading, authenticationVersion, feedback, success, feedbackElement, formatTime, connect, perform, login, sendCode, register, logout, revoke, changeView, loadSessions }
}

export const accountExperienceKey: InjectionKey<ReturnType<typeof useAccountExperience>> = Symbol('account-experience')

export const useAccountContext = () => {
  const context = inject(accountExperienceKey)
  if (!context) throw new Error('Account experience provider is required')
  return context
}

<script setup lang="ts">
import type { PasswordResetRequest, VerificationRequest, VerificationResponse } from '@moetopia/api-client'
import { MoePageHeading, MoeInput, MoePasswordInput, MoeButton, MoeAlert, MoeLink } from '@moetopia/ui'
import { accountMessages, managementMessages, emailLocale } from '@moetopia/locales'
defineProps<{ reset?: boolean }>()
const route = useRoute()
const { locale } = usePortalPreferences()
const text = computed(() => accountMessages[locale.value])
const m = computed(() => managementMessages[locale.value])
const { account, loading, connect } = useAccountContext()
const email = ref('')
const challenge = ref('')
const code = ref('')
const password = ref('')
const busy = ref(false)
const error = ref('')
const done = ref(false)
watch(email, () => { challenge.value = ''; code.value = '' })
const destination = computed(() => ({ path: '/auth/login', query: { ...(typeof route.query.authorization_request === 'string' ? { authorization_request: route.query.authorization_request } : {}), ...(route.query.continue === 'developer-applications' ? { continue: 'developer-applications' } : {}) } }))
const submit = async () => {
  if (busy.value || loading.value) return
  busy.value = true; error.value = ''
  try {
    const headers = { 'X-Moetopia-CSRF': account.value?.csrf_token ?? '' }
    if (!challenge.value) {
      const body: VerificationRequest = { email: email.value, locale: emailLocale(locale.value) }
      const result = await $fetch<VerificationResponse>('/api/v1/account/password-resets', { method: 'POST', headers, body })
      challenge.value = result.challenge_id
    } else {
      const body: PasswordResetRequest = { challenge_id: challenge.value, code: code.value, new_password: password.value }
      await $fetch('/api/v1/account/password-resets/confirm', { method: 'POST', headers, body })
      done.value = true; password.value = ''; code.value = ''; challenge.value = ''; await connect()
    }
  } catch (cause) {
    const status = (cause as { statusCode?: number }).statusCode
    if (status === 403) await connect()
    error.value = status === 422 ? m.value.invalid : status === 403 ? m.value.forbidden : status === 429 ? m.value.limited : m.value.unavailable
  } finally { busy.value = false }
}
onBeforeUnmount(() => { challenge.value = ''; password.value = ''; code.value = '' })
const resend = () => { challenge.value = ''; code.value = ''; password.value = ''; return submit() }
</script>
<template><div class="grid gap-6"><MoePageHeading :title="challenge || reset ? text.resetTitle : text.recoveryTitle" :description="challenge ? m.recoverySent : text.recoveryDescription" /><MoeAlert v-if="done" :title="m.resetDone" color="success" announce /><form v-else class="grid gap-4" @submit.prevent="submit"><MoeInput v-model="email" :label="text.email" type="email" autocomplete="email" required :disabled="busy || loading" /><template v-if="challenge"><MoeInput v-model="code" :label="text.code" inputmode="numeric" autocomplete="one-time-code" :maxlength="8" required :disabled="busy" /><AccountNewPassword v-model="password" :label="text.password" :show-label="text.showPassword" :hide-label="text.hidePassword" autocomplete="new-password" :minlength="15" required :disabled="busy" /></template><MoeAlert v-if="error" :title="error" color="danger" announce /><MoeButton size="lg" type="submit" :loading="busy" :disabled="loading">{{ challenge ? text.resetTitle : text.sendCode }}</MoeButton><MoeButton v-if="challenge" type="button" variant="light" :disabled="busy" @click="resend">{{ text.resend }}</MoeButton></form><MoeLink as-child class="justify-self-center"><NuxtLink :to="destination">{{ text.backToLogin }}</NuxtLink></MoeLink></div></template>

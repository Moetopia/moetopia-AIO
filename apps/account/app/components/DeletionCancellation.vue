<script setup lang="ts">
import type { VerificationResponse, DeletionCancellationRequest } from '@moetopia/api-client'
import { emailLocale, accountMessages } from '@moetopia/locales'
import { MoePageHeading, MoeInput, MoePasswordInput, MoeButton, MoeAlert, MoeLink } from '@moetopia/ui'
const { locale } = usePortalPreferences()
const a = computed(() => accountMessages[locale.value])
const { text, common } = useIntegrationMessages()
const { account, loading, connect } = useAccountContext()
const email = ref('')
const password = ref('')
const code = ref('')
const challenge = ref('')
const busy = ref(false)
const error = ref('')
const done = ref(false)
const submit = async (resend = false) => {
  if (busy.value || loading.value) return
  busy.value = true; error.value = ''
  try {
    const headers = { 'X-Moetopia-CSRF': account.value?.csrf_token ?? '' }
    if (!challenge.value || resend) { const result = await $fetch<VerificationResponse>('/api/v1/account/deletion-cancellations', { method: 'POST', headers, body: { email: email.value, locale: emailLocale(locale.value) }, retry: 0 }); challenge.value = result.challenge_id; code.value = '' }
    else { const body: DeletionCancellationRequest = { challenge_id: challenge.value, code: code.value, current_password: password.value }; await $fetch('/api/v1/account/deletion-cancellations/confirm', { method: 'POST', headers, body, retry: 0 }); done.value = true; challenge.value = ''; code.value = ''; await connect() }
  } catch (cause) { const status = (cause as { statusCode?: number }).statusCode; if (status === 403) await connect(); error.value = status === 422 ? common.value.invalid : status === 429 ? common.value.limited : common.value.error }
  finally { busy.value = false; password.value = '' }
}
watch(email, () => { challenge.value = ''; code.value = '' })
onBeforeUnmount(() => { password.value = ''; challenge.value = ''; code.value = '' })
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text.cancelDeletion" :description="text.deletionHint" /><MoeAlert v-if="done" :title="text.cancelledDeletion" color="success" /><form v-else class="grid gap-4" @submit.prevent="submit()"><MoeInput v-model="email" :label="a.email" type="email" autocomplete="email" required :disabled="busy" /><template v-if="challenge"><MoeInput v-model="code" :label="a.code" autocomplete="one-time-code" inputmode="numeric" pattern="[0-9]{8}" required /><MoePasswordInput v-model="password" :label="common.password" autocomplete="current-password" required /></template><MoeAlert v-if="error" :title="error" color="danger" announce /><MoeButton type="submit" :loading="busy" :disabled="loading">{{ challenge ? text.cancelDeletion : a.sendCode }}</MoeButton><MoeButton v-if="challenge" variant="light" :disabled="busy" @click="submit(true)">{{ a.resend }}</MoeButton></form><MoeLink as-child><NuxtLink to="/auth/forgot-password">{{ a.forgotPassword }}</NuxtLink></MoeLink><MoeLink as-child><NuxtLink to="/auth/login">{{ a.backToLogin }}</NuxtLink></MoeLink></div></template>

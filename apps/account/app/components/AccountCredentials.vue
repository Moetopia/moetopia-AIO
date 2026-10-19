<script setup lang="ts">
import type { EmailChangeRequest, EmailConfirmRequest, PasswordChangeRequest, VerificationResponse } from '@moetopia/api-client'
import { MoeCard, MoeInput, MoePasswordInput, MoeButton, MoeAlert } from '@moetopia/ui'
import { accountMessages, portalMessages, emailLocale } from '@moetopia/locales'
defineProps<{ mode: 'email' | 'password' }>()
const { locale } = usePortalPreferences()
const a = computed(() => accountMessages[locale.value])
const p = computed(() => portalMessages[locale.value])
const { context } = usePortalContext()
const { text, busy, error, headers, run, signedOut } = useManagementRequest()
const password = ref('')
const nextPassword = ref('')
const confirm = ref('')
const email = ref('')
const code = ref('')
const challenge = ref('')
watch(email, () => { challenge.value = ''; code.value = '' })
onBeforeUnmount(() => { password.value = ''; nextPassword.value = ''; confirm.value = ''; code.value = ''; challenge.value = '' })
const send = () => run(async () => {
  const body: EmailChangeRequest = { current_password: password.value, email: email.value, locale: emailLocale(locale.value) }
  const result = await $fetch<VerificationResponse>('/api/v1/portal/email-changes', { method: 'POST', headers: headers.value, body })
  challenge.value = result.challenge_id; code.value = ''
})
const finishEmail = () => run(async () => {
  const body: EmailConfirmRequest = { challenge_id: challenge.value, code: code.value, current_password: password.value }
  await $fetch('/api/v1/portal/email-changes/confirm', { method: 'POST', headers: headers.value, body })
  password.value = ''; code.value = ''; challenge.value = ''; await signedOut()
})
const changePassword = () => {
  if (nextPassword.value !== confirm.value) { error.value = text.value.passwordMismatch; return }
  return run(async () => {
    const body: PasswordChangeRequest = { current_password: password.value, new_password: nextPassword.value }
    await $fetch('/api/v1/portal/password', { method: 'POST', headers: headers.value, body })
    password.value = ''; nextPassword.value = ''; confirm.value = ''; await signedOut()
  })
}
</script>
<template>
  <MoeCard layout="form" :title="mode === 'email' ? p.emailSettings : p.changePassword" :description="text.securitySignOut">
    <form class="grid gap-4" @submit.prevent="mode === 'password' ? changePassword() : challenge ? finishEmail() : send()">
      <MoeInput v-if="mode === 'email'" :model-value="context?.user?.email" :label="a.email" type="email" readonly />
      <MoePasswordInput v-model="password" :label="p.currentPassword" autocomplete="current-password" :show-label="a.showPassword" :hide-label="a.hidePassword" required :disabled="busy" />
      <template v-if="mode === 'email'"><MoeInput v-model="email" :label="p.newEmail" type="email" required :disabled="busy" /><MoeInput v-if="challenge" v-model="code" :label="a.code" inputmode="numeric" autocomplete="one-time-code" :maxlength="8" required :disabled="busy" /><p v-if="challenge" role="status" class="text-sm text-muted">{{ text.challengeSent }}</p></template>
      <template v-else><MoePasswordInput v-model="nextPassword" :label="p.newPassword" :show-label="a.showPassword" :hide-label="a.hidePassword" autocomplete="new-password" :minlength="15" required :disabled="busy" /><MoePasswordInput v-model="confirm" :label="p.confirmPassword" :show-label="a.showPassword" :hide-label="a.hidePassword" autocomplete="new-password" required :disabled="busy" /></template>
      <MoeAlert v-if="error" :title="error" color="danger" announce />
      <MoeButton type="submit" class="w-full" :loading="busy">{{ mode === 'password' ? p.changePassword : challenge ? text.confirmEmail : a.sendCode }}</MoeButton>
      <MoeButton v-if="mode === 'email' && challenge" type="button" variant="light" :disabled="busy" @click="send">{{ a.resend }}</MoeButton>
    </form>
  </MoeCard>
</template>

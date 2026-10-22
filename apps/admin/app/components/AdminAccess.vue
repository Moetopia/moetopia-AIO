<script setup lang="ts">
import type { AdminContext, AdminEnrollment } from '@moetopia/api-client'
import { MoeCard, MoeButton, MoeInput, MoePasswordInput, MoeAlert, MoeSecretDisplay, MoeQrCode } from '@moetopia/ui'
import { securityMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const security = computed(() => securityMessages[locale.value])
const { context, headers, text, busy, error, run, accept, login, logout } = useAdminContext()
const mode = ref<'verify' | 'recover'>('verify')
const password = ref('')
const code = ref('')
const token = ref('')
const enrollment = shallowRef<AdminEnrollment>()
const codes = ref<string[]>([])
const verifiedContext = shallowRef<AdminContext>()
let expiryTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  expiryTimer = setInterval(() => {
    if (enrollment.value && Date.parse(enrollment.value.expires_at) <= Date.now()) { enrollment.value = undefined; code.value = ''; error.value = security.value.expired }
  }, 1000)
})
const finish = () => { codes.value = []; if (verifiedContext.value) accept(verifiedContext.value); verifiedContext.value = undefined }
const phase = computed(() => context.value?.mfa?.emergency_required ? 'emergency' : !context.value?.mfa?.enrolled ? 'enroll' : mode.value)
const submit = () => run(async () => {
  const base = '/api/v1/admin/mfa/'
  try {
    if (phase.value === 'enroll' && !enrollment.value) {
      enrollment.value = await $fetch<AdminEnrollment>(`${base}enrollment`, { method: 'POST', headers: headers.value, body: { current_password: password.value }, retry: 0 })
      return
    }
    const operation = phase.value === 'enroll' ? 'enrollment/confirm' : phase.value
    const body = phase.value === 'emergency' ? { current_password: password.value, token: token.value } : phase.value === 'recover' ? { current_password: password.value, recovery_code: code.value } : { code: code.value }
    const result = await $fetch<AdminContext>(`${base}${operation}`, { method: 'POST', headers: headers.value, body, retry: 0 })
    codes.value = result.recovery_codes ?? []
    if (codes.value.length) verifiedContext.value = result
    else accept(result)
    enrollment.value = undefined; mode.value = 'verify'
  } finally { password.value = ''; code.value = ''; token.value = '' }
})
watch(phase, () => { password.value = ''; code.value = ''; token.value = ''; enrollment.value = undefined })
onBeforeUnmount(() => { clearInterval(expiryTimer); password.value = ''; code.value = ''; token.value = ''; enrollment.value = undefined; codes.value = []; verifiedContext.value = undefined })
</script>
<template><div class="mx-auto grid w-full max-w-lg gap-6 py-12"><MoeCard padding="lg" :title="!context?.authenticated ? text.login : codes.length ? text.recoveryCode : phase === 'enroll' ? security.bindTitle : text[phase]">
  <div v-if="!context?.authenticated" class="grid gap-5"><p class="text-sm text-muted">{{ text.loginHint }}</p><MoeButton @click="login">{{ text.login }}</MoeButton></div>
  <div v-else-if="codes.length" class="grid gap-5"><MoeSecretDisplay :label="text.recoveryCode" :value="codes.join('\n')" :warning="security.codesHint" :copy-label="text.copy" :copied-label="text.copied" :error-label="text.error" /><MoeButton @click="finish">{{ text.savedCodes }}</MoeButton></div>
  <div v-else-if="context.user?.has_password === false" class="grid gap-4"><MoeAlert :title="security.setPassword" color="info" /><MoeButton variant="light" @click="logout()">{{ text.logout }}</MoeButton></div>
  <form v-else class="grid gap-5" @submit.prevent="submit">
    <div v-if="enrollment" class="grid gap-4"><p class="text-sm leading-6 text-muted">{{ security.bindInstructions }}</p><MoeQrCode :value="enrollment.otpauth_uri" :label="security.scan" :error-label="security.qrFailed" /><MoeSecretDisplay :label="text.secret" :value="enrollment.secret" :warning="text.secretWarning" :copy-label="text.copy" :copied-label="text.copied" :error-label="text.error" /><p class="text-xs text-muted">{{ security.expires }} {{ new Date(enrollment.expires_at).toLocaleString(locale) }}</p></div>
    <MoePasswordInput v-if="phase === 'emergency' || phase === 'recover' || phase === 'enroll' && !enrollment" v-model="password" :label="security.loginPassword" :description="security.passwordPurpose" autocomplete="current-password" required :disabled="busy" />
    <MoeInput v-if="phase === 'emergency'" v-model="token" :label="text.token" autocomplete="off" required :disabled="busy" />
    <MoeInput v-else-if="phase !== 'enroll' || enrollment" v-model="code" :label="phase === 'recover' ? text.recoveryCode : text.code" autocomplete="one-time-code" :pattern="phase === 'recover' ? undefined : '[0-9]{6}'" required :disabled="busy" />
    <MoeAlert v-if="error" :title="error" color="danger" announce />
    <MoeButton type="submit" :loading="busy">{{ text.confirm }}</MoeButton>
    <MoeButton v-if="context.mfa?.enrolled && !context.mfa.emergency_required" variant="light" :disabled="busy" @click="mode = mode === 'verify' ? 'recover' : 'verify'">{{ mode === 'verify' ? text.recover : text.verify }}</MoeButton>
    <MoeButton variant="light" :disabled="busy" @click="logout()">{{ text.logout }}</MoeButton>
  </form>
</MoeCard></div></template>

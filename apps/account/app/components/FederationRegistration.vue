<script setup lang="ts">
import type { FederationRegistration, FederationRegistrationRequest, FederationRedirect, VerificationResponse } from '@moetopia/api-client'
import { emailLocale, accountMessages } from '@moetopia/locales'
import { MoePageHeading, MoeInput, MoeButton, MoeAlert, MoeLink } from '@moetopia/ui'
const route = useRoute()
const { locale } = usePortalPreferences()
const a = computed(() => accountMessages[locale.value])
const { text, common } = useIntegrationMessages()
const { account, loading, connect } = useAccountContext()
const details = shallowRef<FederationRegistration>()
const completion = typeof route.query.completion === 'string' ? route.query.completion : ''
const name = ref('')
const email = ref('')
const invitation = ref('')
const code = ref('')
const challenge = ref('')
const busy = ref(false)
const error = ref('')
const headers = computed(() => ({ 'X-Moetopia-CSRF': account.value?.csrf_token ?? '' }))
const run = async (action: () => Promise<void>) => { if (busy.value) return; busy.value = true; error.value = ''; try { await action() } catch (cause) { const status = (cause as { statusCode?: number }).statusCode; if (status === 403) await connect(); error.value = status === 404 ? text.value.expired : status === 422 ? common.value.invalid : status === 429 ? common.value.limited : common.value.error } finally { busy.value = false } }
const send = () => run(async () => { const result = await $fetch<VerificationResponse>('/api/v1/account/verification', { method: 'POST', headers: headers.value, body: { email: email.value, locale: emailLocale(locale.value) }, retry: 0 }); challenge.value = result.challenge_id; code.value = '' })
const submit = () => run(async () => { const body: FederationRegistrationRequest = { completion, challenge_id: challenge.value, code: code.value, invitation: invitation.value, display_name: name.value }; const result = await $fetch<FederationRedirect>('/api/v1/account/federation/registration', { method: 'POST', headers: headers.value, body, retry: 0 }); invitation.value = ''; code.value = ''; window.location.assign(result.redirect_to) })
watch(email, () => { challenge.value = ''; code.value = '' })
onMounted(() => run(async () => { details.value = await $fetch<FederationRegistration>('/api/v1/account/federation/registration', { query: { completion }, retry: 0 }); name.value = details.value.display_name; email.value = details.value.email_hint }))
onBeforeUnmount(() => { invitation.value = ''; code.value = ''; challenge.value = '' })
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text.completeRegistration" :description="text.federationHint" /><MoeAlert v-if="error" :title="error" color="danger" announce /><form v-if="details" class="grid gap-4" @submit.prevent="submit"><MoeInput v-model="name" :label="a.name" :maxlength="80" required /><MoeInput v-model="email" :label="a.email" type="email" autocomplete="email" required /><MoeInput v-if="account?.registration_mode !== 'open'" v-model="invitation" :label="a.invitation" autocomplete="off" required /><MoeInput v-if="challenge" v-model="code" :label="a.code" autocomplete="one-time-code" inputmode="numeric" pattern="[0-9]{8}" required /><MoeButton variant="bordered" :disabled="busy || loading || !email" @click="send">{{ challenge ? a.resend : a.sendCode }}</MoeButton><MoeButton type="submit" :loading="busy" :disabled="loading || !challenge">{{ a.continue }}</MoeButton></form><MoeLink as-child><NuxtLink to="/auth/login">{{ a.backToLogin }}</NuxtLink></MoeLink></div></template>

<script setup lang="ts">
import { MoeAlert, MoeButton, MoeDescriptionList, MoePageHeading, MoeSettingRow, MoeSpinner, MoeAuthorizationHandshake, MoeConsentDetails, MoeAccordion, MoeUserChip } from '@moetopia/ui'
import { authorizationMessages } from '@moetopia/locales'

const { locale } = usePortalPreferences()
const text = computed(() => authorizationMessages[locale.value])
const { formatTime, busy: accountBusy } = useAccountContext()
const { details, loading, submitting, redirecting, feedback, expired, needsLogin, ready, forceLogin, reload, decide, switchAccount, acknowledge } = useAuthorization()
const scopeDescription = (scope: string) => scope === 'openid' ? text.value.openid : scope === 'profile' ? text.value.profile : scope === 'email' ? text.value.email : scope === 'offline_access' ? text.value.offlineAccess : text.value.otherScope
const alertTitle = computed(() => expired.value ? text.value.expired : feedback.value ? text.value[feedback.value] : '')
const actionScopes = computed(() => (details.value?.scopes ?? []).filter(scope => !['openid', 'profile', 'email'].includes(scope)))
const claims = computed(() => {
  const request = details.value
  if (!request?.user || needsLogin.value) return []
  const scopes = request.scopes ?? []
  return [
    ...(scopes.includes('openid') ? [{ key: 'sub', label: text.value.subject, field: 'sub', value: request.user.id }] : []),
    ...(scopes.includes('profile') ? [{ key: 'name', label: text.value.displayName, field: 'name', value: request.user.display_name }] : []),
    ...(scopes.includes('email') ? [{ key: 'email', label: text.value.emailAddress, field: 'email', value: request.user.email }] : []),
  ]
})
</script>

<template>
  <section class="moe-authorization" :aria-label="text.title" :aria-busy="loading || submitting">
    <MoeSpinner v-if="loading" :label="text.loading" />
    <template v-else>
      <MoeAuthorizationHandshake v-if="details && !expired" :application="details.client_name" provider="Moetopia" />
      <MoePageHeading :title="needsLogin && details ? forceLogin || details.user ? text.reauthenticate : text.login : text.title" :description="details ? undefined : text.description" />
      <div v-if="details && !expired"><h2 class="text-sm font-semibold">{{ details.client_name }}</h2><p class="mt-2 text-sm text-muted">{{ text.requestAccess }}</p></div>
    </template>
    <MoeAlert v-if="expired || feedback" :title="alertTitle" :color="feedback === 'changed' && !expired ? 'warning' : 'danger'" announce>
      <template #actions><MoeButton v-if="feedback === 'changed' && !expired" variant="bordered" @click="acknowledge">{{ text.title }}</MoeButton><MoeButton v-else-if="!expired && feedback !== 'expired' && feedback !== 'invalid'" variant="bordered" :disabled="submitting" @click="reload()">{{ text.retry }}</MoeButton></template>
    </MoeAlert>
    <template v-if="details && !loading && !expired && feedback !== 'expired'">
      <template v-if="needsLogin"><p class="text-sm leading-6 text-muted">{{ text.loginDescription }}</p><AccountPanel force-login embedded /></template>
      <template v-else-if="details.user">
        <MoeConsentDetails v-if="claims.length" :title="text.sharedInformation" :items="claims" />
        <section v-if="actionScopes.length" class="moe-consent-permissions"><h2>{{ text.permissions }}</h2><MoeSettingRow v-for="scope in actionScopes" :key="scope" :title="scope" :description="scopeDescription(scope)" icon="check" /></section>
        <p v-else-if="!details.scopes?.length" class="text-sm text-muted">{{ text.noPermissions }}</p>
        <p class="text-xs leading-6 text-muted">{{ text.credentialNotice }}</p>
        <div class="moe-consent-account"><MoeUserChip :name="details.user.display_name" :description="text.account" /><MoeButton variant="light" size="sm" :disabled="submitting || accountBusy || redirecting" @click="switchAccount">{{ text.switchAccount }}</MoeButton></div>
        <div class="grid gap-2.5"><MoeButton size="lg" :loading="submitting" :disabled="!ready" @click="decide('approve')">{{ text.approve }}</MoeButton><MoeButton size="lg" variant="light" color="default" :disabled="!ready" @click="decide('deny')">{{ text.deny }}</MoeButton></div>
        <p v-if="submitting || redirecting" role="status" class="text-sm text-muted">{{ redirecting ? text.redirecting : text.submitting }}</p>
        <p class="moe-consent-return">{{ text.returnNotice }} <span>{{ details.redirect_uri }}</span></p>
        <MoeAccordion :items="[{ value: 'request', title: text.technicalDetails }]"><template #request><MoeDescriptionList :items="[{ key: 'client', label: text.client, value: details.client_id }, { key: 'audience', label: text.audience, value: (details.audiences ?? []).join(', ') }, { key: 'expires', label: text.expires, value: formatTime(details.expires_at) }]" /></template></MoeAccordion>
      </template>
    </template>
  </section>
</template>

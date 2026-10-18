<script setup lang="ts">
import { MoeAlert, MoeButton, MoeCard, MoeDescriptionList, MoePageHeading, MoeSettingRow, MoeSpinner } from '@moetopia/ui'
import { authorizationMessages } from '@moetopia/locales'

const { locale } = usePortalPreferences()
const text = computed(() => authorizationMessages[locale.value])
const { formatTime, busy: accountBusy } = useAccountContext()
const { details, loading, submitting, redirecting, feedback, expired, needsLogin, ready, forceLogin, reload, decide, switchAccount, acknowledge } = useAuthorization()
const scopeDescription = (scope: string) => scope === 'openid' ? text.value.openid : scope === 'profile' ? text.value.profile : scope === 'email' ? text.value.email : text.value.otherScope
const alertTitle = computed(() => expired.value ? text.value.expired : feedback.value ? text.value[feedback.value] : '')
</script>

<template>
  <section class="grid min-w-0 gap-6" :aria-label="text.title" :aria-busy="loading || submitting">
    <MoePageHeading :title="text.title" :description="text.description" />
    <MoeSpinner v-if="loading" :label="text.loading" />
    <MoeAlert v-if="expired || feedback" :title="alertTitle" :color="feedback === 'changed' && !expired ? 'warning' : 'danger'" announce>
      <template #actions><MoeButton v-if="feedback === 'changed' && !expired" variant="bordered" @click="acknowledge">{{ text.title }}</MoeButton><MoeButton v-else-if="!expired && feedback !== 'expired' && feedback !== 'invalid'" variant="bordered" :disabled="submitting" @click="reload()">{{ text.retry }}</MoeButton></template>
    </MoeAlert>
    <template v-if="details && !loading && !expired && feedback !== 'expired'">
      <MoeCard :title="details.client_name" :description="text.application">
        <MoeDescriptionList :items="[{ key: 'client', label: text.client, value: details.client_id }, { key: 'callback', label: text.callback, value: details.redirect_uri }, { key: 'audience', label: text.audience, value: (details.audiences ?? []).join(', ') }, { key: 'expires', label: text.expires, value: formatTime(details.expires_at) }]" />
      </MoeCard>
      <MoeCard :title="text.permissions">
        <MoeSettingRow v-for="scope in details.scopes ?? []" :key="scope" :title="scope" :description="scopeDescription(scope)" icon="shield" />
        <p v-if="!details.scopes?.length" class="text-sm text-muted">{{ text.noPermissions }}</p>
      </MoeCard>
      <template v-if="needsLogin">
        <MoeAlert :title="forceLogin || details.user ? text.reauthenticate : text.login" color="info">{{ text.loginDescription }}</MoeAlert>
        <AccountPanel force-login />
      </template>
      <template v-else-if="details.user">
        <MoeCard :title="text.account">
          <MoeSettingRow :title="details.user.display_name" :description="details.user.email" icon="user"><template #actions><MoeButton variant="light" :disabled="submitting || accountBusy || redirecting" @click="switchAccount">{{ text.switchAccount }}</MoeButton></template></MoeSettingRow>
        </MoeCard>
        <div class="grid grid-cols-2 gap-3"><MoeButton variant="bordered" color="default" :disabled="!ready" @click="decide('deny')">{{ text.deny }}</MoeButton><MoeButton :loading="submitting" :disabled="!ready" @click="decide('approve')">{{ text.approve }}</MoeButton></div>
        <p v-if="submitting || redirecting" role="status" class="text-sm text-muted">{{ redirecting ? text.redirecting : text.submitting }}</p>
      </template>
    </template>
  </section>
</template>

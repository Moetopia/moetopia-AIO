<script setup lang="ts">
import type { LogoutDetails, LogoutDecisionRequest, LogoutDecisionResponse } from '@moetopia/api-client'
import { MoePageHeading, MoeCard, MoeButton, MoeAlert, MoeDescriptionList } from '@moetopia/ui'
const route = useRoute()
const { text, common } = useIntegrationMessages()
const { account, connect, loading } = useAccountContext()
const details = shallowRef<LogoutDetails>()
const busy = ref(false)
const error = ref('')
const id = typeof route.query.logout_request === 'string' && /^[A-Za-z0-9_-]{43}$/.test(route.query.logout_request) ? route.query.logout_request : ''
const endpoint = `/api/v1/account/logout-requests/${encodeURIComponent(id)}`
const refresh = async () => { details.value = undefined; await connect(); if (!id) throw new Error('Invalid logout request'); details.value = await $fetch<LogoutDetails>(endpoint, { retry: 0 }) }
const run = async (action: () => Promise<void>) => { if (busy.value) return; busy.value = true; error.value = ''; try { await action() } catch (cause) { const status = (cause as { statusCode?: number }).statusCode; if (status === 409) { try { await refresh(); error.value = common.value.conflict } catch { error.value = text.value.expired } } else error.value = status === 404 || !id ? text.value.expired : status === 429 ? common.value.limited : common.value.error } finally { busy.value = false } }
const decide = (decision: 'approve' | 'deny') => run(async () => {
  if (!details.value) return
  const body: LogoutDecisionRequest = { decision, subject_id: details.value.user?.id ?? '' }
  const result = await $fetch<LogoutDecisionResponse>(endpoint, { method: 'POST', headers: { 'X-Moetopia-CSRF': account.value?.csrf_token ?? '' }, body, retry: 0 })
  window.location.assign(result.redirect_to)
})
onMounted(() => run(refresh))
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text.confirmLogout" :description="text.logoutHint" /><MoeAlert v-if="error" :title="error" color="danger" announce /><MoeCard v-if="details" padding="lg" :title="details.client_name"><MoeDescriptionList :items="[{ key: 'account', label: common.email, value: details.user?.email ?? common.empty }, { key: 'return', label: text.returnTo, value: details.post_logout_redirect_uri || '—' }]" /><div class="mt-6 flex justify-end gap-3"><MoeButton variant="light" :disabled="busy || loading" @click="decide('deny')">{{ common.cancel }}</MoeButton><MoeButton :loading="busy" :disabled="loading" @click="decide('approve')">{{ common.confirm }}</MoeButton></div></MoeCard></div></template>

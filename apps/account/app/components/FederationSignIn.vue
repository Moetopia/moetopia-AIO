<script setup lang="ts">
import type { FederationProviders, FederationRedirect, FederationStartRequest } from '@moetopia/api-client'
import { MoeButton } from '@moetopia/ui'
const route = useRoute()
const { text } = useIntegrationMessages()
const { account, loading, busy, perform } = useAccountContext()
const providers = shallowRef<FederationProviders>()
onMounted(async () => { try { providers.value = await $fetch<FederationProviders>('/api/v1/account/federation/providers', { retry: 0 }) } catch { providers.value = undefined } })
const start = (provider: string) => perform(async () => {
  const body: FederationStartRequest = { ...(typeof route.query.authorization_request === 'string' && /^[A-Za-z0-9_-]{43}$/.test(route.query.authorization_request) ? { authorization_request: route.query.authorization_request } : {}), ...(route.query.continue === 'developer-applications' ? { continue: 'developer-applications' as const } : {}) }
  const result = await $fetch<FederationRedirect>(`/api/v1/account/federation/${encodeURIComponent(provider)}`, { method: 'POST', headers: { 'X-Moetopia-CSRF': account.value?.csrf_token ?? '' }, body, retry: 0 })
  window.location.assign(result.redirect_to)
})
</script>
<template><section v-if="providers?.items?.length" class="grid gap-3" :aria-label="text.federation"><p class="text-center text-xs text-muted">{{ text.federation }}</p><div class="flex flex-wrap gap-3"><MoeButton v-for="provider in providers.items" :key="provider" class="flex-1" variant="bordered" :disabled="loading || busy || !account" @click="start(provider)">{{ provider === 'github' ? 'GitHub' : provider === 'google' ? 'Google' : provider }}</MoeButton></div></section></template>

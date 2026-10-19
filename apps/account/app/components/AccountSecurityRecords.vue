<script setup lang="ts">
import type { UserAuthorizations, SecurityEvents } from '@moetopia/api-client'
import { MoeCard, MoeButton, MoeAlert, MoeEmptyState, MoeSpinner, MoeSettingRow, MoeDialog, MoeDescriptionList, MoeBadge } from '@moetopia/ui'
import { accountMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const a = computed(() => accountMessages[locale.value])
const { text, busy, error, headers, run } = useManagementRequest()
const { formatTime } = usePortalContext()
const grants = ref<NonNullable<UserAuthorizations['items']>>([])
const events = ref<NonNullable<SecurityEvents['items']>>([])
const cursor = ref('')
const ready = ref(false)
const selected = ref('')
const confirming = ref(false)
const load = (more = false) => run(async () => {
  const result = await $fetch<UserAuthorizations>('/api/v1/portal/authorizations', { query: more ? { cursor: cursor.value } : {} })
  grants.value = more ? [...grants.value, ...(result.items ?? [])] : result.items ?? []
  cursor.value = result.next_cursor
  if (!more) { const result = await $fetch<SecurityEvents>('/api/v1/portal/security-events'); events.value = result.items ?? [] }
  ready.value = true
})
const revoke = () => run(async () => {
  await $fetch(`/api/v1/portal/authorizations/${encodeURIComponent(selected.value)}`, { method: 'DELETE', headers: headers.value })
  grants.value = grants.value.filter(item => item.id !== selected.value); confirming.value = false
})
onMounted(() => { void load() })
</script>
<template><div class="grid gap-6">
  <MoeAlert v-if="error" :title="error" color="danger" announce><template #actions><MoeButton :disabled="busy" @click="load()">{{ a.retry }}</MoeButton></template></MoeAlert>
  <MoeSpinner v-if="busy && !ready" :label="a.checking" />
  <MoeCard layout="form" :title="text.authorizations" :description="text.grantHint"><MoeSettingRow v-for="grant in grants" :key="grant.id" :title="grant.client_name" :description="grant.client_id" icon="grid"><MoeDescriptionList :items="[{ key: 'created', label: a.created, value: formatTime(grant.created_at) }, { key: 'expires', label: a.expires, value: formatTime(grant.expires_at) }]" /><div class="flex flex-wrap gap-2"><MoeBadge v-for="scope in grant.scopes" :key="scope">{{ scope }}</MoeBadge></div><template #actions><MoeButton color="danger" variant="light" :disabled="busy" @click="selected = grant.id; confirming = true">{{ text.revokeGrant }}</MoeButton></template></MoeSettingRow><MoeEmptyState v-if="ready && !grants.length" :title="text.empty" icon="shield" /><MoeButton v-if="cursor" variant="bordered" :loading="busy" @click="load(true)">{{ text.more }}</MoeButton></MoeCard>
  <MoeCard layout="form" :title="text.securityEvents" :description="text.eventHint"><MoeSettingRow v-for="event in events" :key="event.id" :title="event.action" :description="formatTime(event.created_at)" icon="shield" /><MoeEmptyState v-if="ready && !events.length" :title="text.empty" icon="shield" /></MoeCard>
  <MoeDialog v-model:open="confirming" :title="text.revokeGrant" :description="text.revokeGrantConfirm" :close-label="text.close" :show-close-action="false"><template #footer><MoeButton variant="flat" :disabled="busy" @click="confirming = false">{{ text.close }}</MoeButton><MoeButton color="danger" :loading="busy" @click="revoke">{{ text.revokeGrant }}</MoeButton></template></MoeDialog>
</div></template>

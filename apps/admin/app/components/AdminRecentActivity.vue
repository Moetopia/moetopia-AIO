<script setup lang="ts">
import type { AdminAuditEvents } from '@moetopia/api-client'
import { securityMessages } from '@moetopia/locales'
import { MoeCard, MoeSettingRow, MoeLink, MoeAlert, MoeButton, MoeEmptyState } from '@moetopia/ui'
const { locale } = usePortalPreferences()
const security = computed(() => securityMessages[locale.value])
const { text, run, error, busy } = useAdminContext()
const events = shallowRef<AdminAuditEvents>()
const load = () => run(async () => { events.value = await $fetch<AdminAuditEvents>('/api/v1/admin/audit-events', { retry: 0 }) })
onMounted(load)
</script>
<template><MoeCard padding="lg"><div class="mb-4 flex items-center justify-between gap-3"><h2 class="text-lg font-semibold">{{ security.recentActivity }}</h2><MoeLink as-child><NuxtLink to="/audit">{{ security.viewAll }}</NuxtLink></MoeLink></div><MoeAlert v-if="error" :title="error" color="danger" announce><template #actions><MoeButton :loading="busy" @click="load">{{ text.refresh }}</MoeButton></template></MoeAlert><template v-else-if="events?.items?.length"><MoeSettingRow v-for="event in events.items.slice(0, 6)" :key="event.id" :title="event.action" :description="event.reason || event.outcome" icon="shield"><template #actions><time :datetime="event.created_at" class="text-xs text-muted">{{ new Date(event.created_at).toLocaleString(locale) }}</time></template></MoeSettingRow></template><MoeEmptyState v-else :title="busy ? text.refresh : text.empty" /></MoeCard></template>

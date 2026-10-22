<script setup lang="ts">
import type { AdminStatistics } from '@moetopia/api-client'
import { MoePageHeading, MoeButton, MoeAlert, MoeStatGrid, MoeCard, MoeTimeSeriesChart } from '@moetopia/ui'
const { text, run, error, busy } = useAdminContext()
const data = shallowRef<AdminStatistics>()
const load = () => run(async () => { data.value = await $fetch<AdminStatistics>('/api/v1/admin/statistics', { retry: 0 }) })
const items = computed(() => data.value ? [
  ['totalUsers', data.value.identity.total_users], ['enabledUsers', data.value.identity.enabled_users], ['deletions', data.value.identity.pending_deletions],
  ['applications', data.value.applications.total], ['pending', data.value.applications.pending], ['published', data.value.applications.published_enabled],
  ['queued', data.value.notifications.queued], ['inFlight', data.value.notifications.in_flight], ['exhausted', data.value.notifications.retry_exhausted],
].map(([id, value]) => ({ id: String(id), label: text.value[id as keyof typeof text.value], value: String(value), icon: 'grid' as const })) : [])
onMounted(load)
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text.overview"><template #actions><MoeButton :loading="busy" variant="bordered" @click="load">{{ text.refresh }}</MoeButton></template></MoePageHeading><MoeAlert v-if="error" :title="error" color="danger" announce /><MoeStatGrid :items="items" /><MoeCard v-if="data" padding="lg"><MoeTimeSeriesChart :label="text.registrations" :missing-label="text.missing" :partial-label="text.partial" :points="data.identity.registrations.map(day => ({ day: day.day, value: day.registrations, partial: day.partial }))" /><p class="mt-4 text-xs text-muted">UTC · {{ data.generated_at }}</p></MoeCard></div></template>

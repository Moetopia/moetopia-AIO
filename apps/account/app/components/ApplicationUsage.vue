<script setup lang="ts">
import type { DeveloperApplications, DeveloperApplicationStatistics } from '@moetopia/api-client'
import { MoePageHeading, MoeSelect, MoeButton, MoeStatGrid, MoeCard, MoeTimeSeriesChart, MoeAlert, MoeEmptyState } from '@moetopia/ui'
const { text, common } = useIntegrationMessages()
const { run, error, busy } = useManagementRequest()
const applications = shallowRef<DeveloperApplications>()
const selected = ref('')
const days = ref('7')
const metric = ref<'authorizations' | 'code_exchanges' | 'refreshes' | 'machine_tokens'>('authorizations')
const statistics = shallowRef<DeveloperApplicationStatistics>()
const metrics = ['authorizations', 'code_exchanges', 'refreshes', 'machine_tokens'] as const
const points = computed(() => statistics.value?.days.slice(-Number(days.value)).map(day => ({ day: day.day, value: day.counts?.[metric.value], partial: day.partial })) ?? [])
const items = computed(() => {
  const recorded = statistics.value?.days.slice(-Number(days.value)).flatMap(day => day.counts ? [day.counts] : [])
  return metrics.map(id => ({ id, label: text.value[id], value: recorded?.length ? String(recorded.reduce((sum, counts) => sum + counts[id], 0)) : undefined, icon: 'grid' as const }))
})
const loadApplications = () => run(async () => {
  const page = await $fetch<DeveloperApplications>('/api/v1/developer/applications', { query: { cursor: applications.value?.next_cursor ?? '' }, retry: 0 })
  applications.value = { ...page, items: [...(applications.value?.items ?? []), ...(page.items ?? [])] }
  if (!selected.value && applications.value.items?.[0]) selected.value = applications.value.items[0].id
})
const refresh = () => run(async () => { statistics.value = await $fetch<DeveloperApplicationStatistics>(`/api/v1/developer/applications/${encodeURIComponent(selected.value)}/statistics`, { retry: 0 }) })
watch(selected, () => { statistics.value = undefined; void nextTick().then(refresh) })
onMounted(loadApplications)
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text.usage" :description="text.usageHint"><template #actions><MoeButton :loading="busy" :disabled="!selected" variant="bordered" @click="refresh">{{ common.refresh }}</MoeButton></template></MoePageHeading><MoeAlert v-if="error" :title="error" color="danger" announce /><MoeEmptyState v-if="applications && !applications.items?.length" :title="common.empty" /><template v-else><MoeSelect v-model="selected" :label="common.applications" :options="applications?.items?.map(app => ({ value: app.id, label: app.latest.config.name })) ?? []" /><MoeButton v-if="applications?.next_cursor" variant="light" :disabled="busy" @click="loadApplications">{{ common.next }}</MoeButton><div class="grid gap-4 sm:grid-cols-2"><MoeSelect v-model="days" :label="text.usage" :options="['7', '14', '30'].map(value => ({ value, label: `${value} ${text.days}` }))" /><MoeSelect v-model="metric" :label="common.status" :options="metrics.map(value => ({ value, label: text[value] }))" /></div><MoeStatGrid :items="items" /><MoeCard v-if="statistics" padding="lg"><MoeTimeSeriesChart :label="text[metric]" :points="points" :missing-label="common.missing" :partial-label="common.partial" /><p class="mt-3 text-xs text-muted">UTC · {{ statistics.generated_at }}</p></MoeCard></template></div></template>

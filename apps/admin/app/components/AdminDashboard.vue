<script setup lang="ts">
import type { AdminStatistics } from '@moetopia/api-client'
import { securityMessages, integrationMessages } from '@moetopia/locales'
import { MoePageHeading, MoeButton, MoeAlert, MoeStatGrid, MoeCard, MoeTimeSeriesChart, MoeActionGrid, MoeDescriptionList, MoeButtonGroup } from '@moetopia/ui'
import type { MoeNavEntry } from '@moetopia/ui'
const { locale } = usePortalPreferences()
const security = computed(() => securityMessages[locale.value])
const integration = computed(() => integrationMessages[locale.value])
const { text, context, run, error, busy } = useAdminContext()
const data = shallowRef<AdminStatistics>()
const days = ref('30')
const load = () => run(async () => { data.value = await $fetch<AdminStatistics>('/api/v1/admin/statistics', { retry: 0 }) })
const items = computed(() => [
  { id: 'users', label: text.value.totalUsers, value: data.value && String(data.value.identity.total_users), icon: 'user' as const, description: data.value && `${text.value.enabledUsers} ${data.value.identity.enabled_users} · ${text.value.deletions} ${data.value.identity.pending_deletions}` },
  { id: 'applications', label: text.value.applications, value: data.value && String(data.value.applications.total), icon: 'grid' as const, description: data.value && `${text.value.published} ${data.value.applications.published_enabled} · ${text.value.suspend} ${data.value.applications.suspended}` },
  { id: 'pending', label: text.value.pending, value: data.value && String(data.value.applications.pending), icon: 'shield' as const },
])
const actions = computed<MoeNavEntry[]>(() => [
  { id: 'users', label: text.value.users, href: '/users', icon: 'user' },
  { id: 'applications', label: text.value.applications, href: '/applications', icon: 'grid' },
  { id: 'invitations', label: text.value.invitations, href: '/invitations', icon: 'plus' },
  { id: 'settings', label: text.value.settings, href: '/settings', icon: 'settings' },
])
const navigate = (event: MouseEvent, item: MoeNavEntry) => { if (event.button === 0 && !event.ctrlKey && !event.metaKey && !event.shiftKey && !event.altKey) { event.preventDefault(); void navigateTo(item.href) } }
const notifications = computed(() => {
  const statistics = data.value?.notifications
  return statistics ? (['queued', 'in_flight', 'retry_exhausted'] as const).map(key => ({ key, label: text.value[key === 'in_flight' ? 'inFlight' : key === 'retry_exhausted' ? 'exhausted' : 'queued'], value: String(statistics[key]) })) : []
})
onMounted(load)
</script>
<template>
  <div class="grid gap-6">
    <MoePageHeading :title="text.overview" :description="`${security.welcome}${context?.user?.display_name ?? ''}`"><template #actions><MoeButton :loading="busy" variant="bordered" @click="load">{{ text.refresh }}</MoeButton></template></MoePageHeading>
    <MoeAlert v-if="error" :title="error" color="danger" announce />
    <MoeStatGrid :items="items" />
    <MoeCard v-if="data" padding="lg">
      <div class="mb-5 flex justify-end"><MoeButtonGroup :label="text.registrations"><MoeButton v-for="value in ['7', '14', '30']" :key="value" size="sm" :variant="days === value ? 'flat' : 'light'" :aria-pressed="days === value" @click="days = value">{{ value }} {{ integration.days }}</MoeButton></MoeButtonGroup></div>
      <MoeTimeSeriesChart :label="text.registrations" :missing-label="text.missing" :partial-label="text.partial" :points="data.identity.registrations.slice(-Number(days)).map(day => ({ day: day.day, value: day.registrations, partial: day.partial }))" />
      <p class="mt-4 text-xs text-muted">UTC · {{ data.generated_at }}</p>
    </MoeCard>
    <MoeCard :title="security.quickActions" padding="lg"><MoeActionGrid :items="actions" @navigate="navigate" /></MoeCard>
    <AdminRecentActivity />
    <MoeCard v-if="data" :title="security.notifications" padding="lg"><MoeDescriptionList :items="notifications" /></MoeCard>
  </div>
</template>

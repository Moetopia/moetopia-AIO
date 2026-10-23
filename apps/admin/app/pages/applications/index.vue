<script setup lang="ts">
import type { AdminApplications } from '@moetopia/api-client'
import { MoePageHeading, MoeSelect, MoeTable, MoeButton, MoeLink, MoeAlert, MoeBadge } from '@moetopia/ui'
const { text, run, error, busy } = useAdminContext()
const status = ref('all')
const cursor = ref('')
const history = ref<string[]>([])
const data = shallowRef<AdminApplications>()
const refresh = async () => { data.value = await $fetch<AdminApplications>('/api/v1/admin/applications', { query: { status: status.value, ...(cursor.value ? { cursor: cursor.value } : {}) }, retry: 0 }) }
const load = () => run(refresh)
watch(status, () => { cursor.value = ''; history.value = []; void load() })
const page = (next: boolean) => { if (next) { history.value.push(cursor.value); cursor.value = data.value?.next_cursor ?? '' } else cursor.value = history.value.pop() ?? ''; void load() }
const statusLabel = (value: string) => ['pending', 'approved', 'rejected', 'draft'].includes(value) ? text.value[value as 'pending' | 'approved' | 'rejected' | 'draft'] : value
onMounted(load)
</script>
<template><div class="grid gap-6">
  <MoePageHeading :title="text.applications" :description="text.allApplicationsHint"><template #actions><MoeButton variant="bordered" :loading="busy" @click="load">{{ text.refresh }}</MoeButton></template></MoePageHeading>
  <MoeSelect v-model="status" :label="text.status" :options="[{ value: 'all', label: text.all }, { value: 'pending', label: text.pending }]" />
  <MoeAlert v-if="error" :title="error" color="danger" announce />
  <MoeTable :caption="text.applications" :columns="[{ key: 'name', label: text.name }, { key: 'client_id', label: 'Client ID' }, { key: 'status', label: text.status }, { key: 'enabled', label: text.enabled }, { key: 'created_at', label: text.created }, { key: 'actions', label: text.actions }]" :rows="data?.items ?? []" row-key="id" :empty-title="text.empty" :loading="busy && !data" :loading-label="text.refresh">
    <template #cell-name="{ row }"><MoeLink as-child><NuxtLink :to="`/applications/${row.id}`">{{ row.latest.config.name }}</NuxtLink></MoeLink></template>
    <template #cell-status="{ row }"><div class="flex flex-wrap gap-2"><MoeBadge>{{ statusLabel(row.latest.status) }}</MoeBadge><MoeBadge v-if="row.suspended" color="danger">{{ text.suspend }}</MoeBadge></div></template>
    <template #cell-enabled="{ value }">{{ value ? text.enabled : text.disabled }}</template>
    <template #cell-actions="{ row }"><div class="flex items-center gap-3"><MoeLink as-child><NuxtLink :to="`/applications/${row.id}`">{{ text.viewApplication }}</NuxtLink></MoeLink><AdminApplicationActions :application="row" :reload="refresh" compact @deleted="load" /></div></template>
  </MoeTable>
  <div class="flex justify-end gap-3"><MoeButton :disabled="busy || !history.length" variant="light" @click="page(false)">{{ text.previous }}</MoeButton><MoeButton :disabled="busy || !data?.next_cursor" variant="bordered" @click="page(true)">{{ text.next }}</MoeButton></div>
</div></template>

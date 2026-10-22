<script setup lang="ts">
import type { AdminApplications } from '@moetopia/api-client'
import { MoePageHeading, MoeSelect, MoeTable, MoeButton, MoeLink, MoeAlert } from '@moetopia/ui'
const { text, run, error, busy } = useAdminContext()
const status = ref('pending')
const cursor = ref('')
const history = ref<string[]>([])
const data = shallowRef<AdminApplications>()
const load = () => run(async () => { data.value = await $fetch<AdminApplications>('/api/v1/admin/applications', { query: { status: status.value, cursor: cursor.value }, retry: 0 }) })
watch(status, () => { cursor.value = ''; history.value = []; void load() })
const page = (next: boolean) => { if (next) { history.value.push(cursor.value); cursor.value = data.value?.next_cursor ?? '' } else cursor.value = history.value.pop() ?? ''; void load() }
onMounted(load)
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text.applications"><template #actions><MoeButton variant="bordered" :loading="busy" @click="load">{{ text.refresh }}</MoeButton></template></MoePageHeading><MoeSelect v-model="status" :label="text.status" :options="[{ value: 'pending', label: text.pending }, { value: 'all', label: text.all }]" /><MoeAlert v-if="error" :title="error" color="danger" announce /><MoeTable :caption="text.applications" :columns="[{ key: 'name', label: text.name }, { key: 'client_id', label: 'Client ID' }, { key: 'status', label: text.status }, { key: 'created_at', label: text.created }]" :rows="data?.items ?? []" row-key="id" :empty-title="text.empty" :loading="busy && !data" :loading-label="text.refresh"><template #cell-name="{ row }"><MoeLink as-child><NuxtLink :to="`/applications/${row.id}`">{{ row.latest.config.name }}</NuxtLink></MoeLink></template><template #cell-status="{ row }">{{ row.latest.status }} {{ row.suspended ? `(${text.suspend})` : '' }}</template></MoeTable><div class="flex justify-end gap-3"><MoeButton :disabled="busy || !history.length" variant="light" @click="page(false)">{{ text.previous }}</MoeButton><MoeButton :disabled="busy || !data?.next_cursor" variant="bordered" @click="page(true)">{{ text.next }}</MoeButton></div></div></template>

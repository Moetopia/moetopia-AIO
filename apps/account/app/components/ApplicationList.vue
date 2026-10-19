<script setup lang="ts">
import type { DeveloperApplication, DeveloperApplications } from '@moetopia/api-client'
import { MoeCard, MoeButton, MoeBadge, MoeEmptyState, MoeSpinner, MoeAlert, MoeSettingRow, MoeLink } from '@moetopia/ui'
import { accountMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const a = computed(() => accountMessages[locale.value])
const { text, busy, error, run } = useManagementRequest()
const items = ref<DeveloperApplication[]>([])
const cursor = ref('')
const ready = ref(false)
const load = (more = false) => run(async () => {
  const result = await $fetch<DeveloperApplications>('/api/v1/developer/applications', { query: more ? { cursor: cursor.value } : {} })
  items.value = more ? [...items.value, ...(result.items ?? [])] : result.items ?? []
  cursor.value = result.next_cursor; ready.value = true
})
const created = (app: DeveloperApplication) => { items.value = [app, ...items.value]; ready.value = true }
defineExpose({ created })
const status = (value: string) => ['draft', 'pending', 'approved', 'rejected'].includes(value) ? text.value[value as 'draft' | 'pending' | 'approved' | 'rejected'] : value
onMounted(() => { void load() })
</script>
<template><div class="grid gap-4"><MoeAlert v-if="error" :title="error" color="danger" announce><template #actions><MoeButton :disabled="busy" @click="load()">{{ a.retry }}</MoeButton></template></MoeAlert><MoeSpinner v-if="busy && !ready" :label="a.checking" /><MoeCard v-if="items.length" padding="none"><MoeSettingRow v-for="app in items" :key="app.id" :title="app.latest.config.name" :description="app.latest.config.description" icon="grid"><div class="flex flex-wrap gap-2"><MoeBadge>{{ status(app.latest.status) }}</MoeBadge><MoeBadge :color="app.enabled ? 'success' : 'default'">{{ app.enabled ? text.enabled : text.disabled }}</MoeBadge></div><template #actions><MoeLink as-child><NuxtLink :to="`/developer/applications/${app.id}`">{{ text.details }}</NuxtLink></MoeLink></template></MoeSettingRow></MoeCard><MoeCard v-else-if="ready" class="moe-console-empty" padding="none"><MoeEmptyState :title="text.empty" :description="text.testOnly" icon="grid"><slot /></MoeEmptyState></MoeCard><MoeButton v-if="cursor" :loading="busy" variant="bordered" @click="load(true)">{{ text.more }}</MoeButton></div></template>

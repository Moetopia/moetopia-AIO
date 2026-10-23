<script setup lang="ts">
import type { DeveloperApplication, AdminApplicationHistory } from '@moetopia/api-client'
import { MoePageHeading, MoeResourceSummary, MoeAlert, MoeLink, MoeButton } from '@moetopia/ui'
const route = useRoute()
const { text, run, error, busy } = useAdminContext()
const endpoint = `/api/v1/admin/applications/${encodeURIComponent(String(route.params.id))}`
const app = shallowRef<DeveloperApplication>()
const versions = shallowRef<AdminApplicationHistory>()
const refresh = async () => { const [a, v] = await Promise.all([$fetch<DeveloperApplication>(endpoint, { retry: 0 }), $fetch<AdminApplicationHistory>(`${endpoint}/versions`, { retry: 0 })]); app.value = a; versions.value = v }
const load = () => run(refresh)
const more = () => run(async () => { if (!versions.value?.next_before) return; const page = await $fetch<AdminApplicationHistory>(`${endpoint}/versions`, { query: { before: versions.value.next_before }, retry: 0 }); versions.value = { ...page, items: [...(versions.value.items ?? []), ...(page.items ?? [])] } })
onMounted(load)
</script>
<template><div class="grid gap-8">
  <MoeLink as-child><NuxtLink to="/applications">{{ text.backApplications }}</NuxtLink></MoeLink>
  <MoePageHeading :title="text.applications"><template #actions><MoeButton variant="bordered" :loading="busy" @click="load">{{ text.refresh }}</MoeButton></template></MoePageHeading>
  <MoeAlert v-if="error" :title="error" color="danger" announce />
  <template v-if="app">
    <MoeResourceSummary :title="app.latest.config.name" :description="app.latest.config.description" :identifier="app.client_id" :copy-label="text.copy" :copied-label="text.copied" :error-label="text.error" :items="[{ key: 'id', label: 'Application ID', value: app.id }, { key: 'type', label: 'Client type', value: app.public ? 'Public' : 'Confidential' }, { key: 'enabled', label: text.enabled, value: app.enabled ? text.enabled : text.disabled }, { key: 'status', label: text.status, value: app.latest.status }, { key: 'revision', label: text.revision, value: String(app.revision) }, { key: 'active', label: text.active, value: String(app.active_version) }, { key: 'created', label: text.created, value: app.created_at }]">
      <template #actions><AdminApplicationActions :application="app" :reload="refresh" @deleted="navigateTo('/applications')" /></template>
    </MoeResourceSummary>
    <MoeAlert v-if="app.suspended" :title="text.suspend" color="danger">{{ app.suspension_reason }}</MoeAlert>
    <ApplicationConfiguration :title="text.latestConfiguration" :config="app.latest.config" />
    <ApplicationConfiguration v-if="app.approved && app.active_version === app.approved.version" :title="text.effectiveConfiguration" :config="app.approved.config" />
    <MoeAlert v-else :title="text.noActiveConfiguration" />
    <VersionHistory :history="versions" :busy="busy" @more="more" />
  </template>
</div></template>

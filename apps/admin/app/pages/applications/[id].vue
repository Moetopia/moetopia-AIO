<script setup lang="ts">
import type { DeveloperApplication, AdminApplicationHistory, AdminReviewRequest, AdminSuspensionRequest } from '@moetopia/api-client'
import { MoePageHeading, MoeResourceSummary, MoeButton, MoeAlert, MoeDialog, MoeTextarea } from '@moetopia/ui'
const route = useRoute()
const { text, headers, run, error, busy } = useAdminContext()
const endpoint = `/api/v1/admin/applications/${encodeURIComponent(String(route.params.id))}`
const app = shallowRef<DeveloperApplication>()
const versions = shallowRef<AdminApplicationHistory>()
const action = ref<'approve' | 'reject' | 'suspend' | 'resume'>('approve')
const dialog = ref(false)
const reason = ref('')
const refresh = async () => { const [a, v] = await Promise.all([$fetch<DeveloperApplication>(endpoint, { retry: 0 }), $fetch<AdminApplicationHistory>(`${endpoint}/versions`, { retry: 0 })]); app.value = a; versions.value = v }
const load = () => run(refresh)
const more = () => run(async () => { if (!versions.value?.next_before) return; const page = await $fetch<AdminApplicationHistory>(`${endpoint}/versions`, { query: { before: versions.value.next_before }, retry: 0 }); versions.value = { ...page, items: [...(versions.value.items ?? []), ...(page.items ?? [])] } })
const ask = (next: typeof action.value) => { action.value = next; reason.value = ''; error.value = ''; dialog.value = true }
const perform = () => run(async () => {
  if (!app.value) return
  if (action.value === 'approve' || action.value === 'reject') {
    const body: AdminReviewRequest = { revision: app.value.revision, approve: action.value === 'approve', reason: reason.value }
    app.value = await $fetch<DeveloperApplication>(`${endpoint}/review`, { method: 'POST', headers: headers.value, body, retry: 0 })
  } else {
    const body: AdminSuspensionRequest = { revision: app.value.revision, suspended: action.value === 'suspend', reason: reason.value }
    app.value = await $fetch<DeveloperApplication>(`${endpoint}/suspension`, { method: 'POST', headers: headers.value, body, retry: 0 })
  }
  dialog.value = false; await refresh()
}, refresh)
onMounted(load)
</script>
<template><div class="grid gap-8"><MoePageHeading :title="text.applications" /><MoeAlert v-if="error && !dialog" :title="error" color="danger" announce /><template v-if="app"><MoeResourceSummary :title="app.latest.config.name" :description="app.latest.config.description" :identifier="app.client_id" :copy-label="text.copy" :copied-label="text.copied" :error-label="text.error" :items="[{ key: 'status', label: text.status, value: app.latest.status }, { key: 'revision', label: text.revision, value: String(app.revision) }, { key: 'active', label: text.active, value: String(app.active_version) }, { key: 'created', label: text.created, value: app.created_at }]"><template #actions><MoeButton v-if="app.latest.status === 'pending'" :disabled="busy" @click="ask('approve')">{{ text.approve }}</MoeButton><MoeButton v-if="app.latest.status === 'pending'" color="danger" variant="flat" :disabled="busy" @click="ask('reject')">{{ text.reject }}</MoeButton><MoeButton variant="bordered" :disabled="busy" @click="ask(app.suspended ? 'resume' : 'suspend')">{{ app.suspended ? text.resume : text.suspend }}</MoeButton></template></MoeResourceSummary><MoeAlert v-if="app.suspended" :title="text.suspend" :description="app.suspension_reason" color="danger" /><VersionHistory :history="versions" :busy="busy" @more="more" /></template><MoeDialog v-model:open="dialog" :title="text[action]" :close-label="text.cancel" :show-close-action="false"><form class="grid gap-4" @submit.prevent="perform"><MoeTextarea v-model="reason" :label="text.reason" :required="action === 'reject' || action === 'suspend'" :maxlength="500" /><MoeAlert v-if="error" :title="error" color="danger" announce /><div class="flex justify-end gap-3"><MoeButton variant="light" :disabled="busy" @click="dialog = false">{{ text.cancel }}</MoeButton><MoeButton type="submit" :loading="busy">{{ text.confirm }}</MoeButton></div></form></MoeDialog></div></template>

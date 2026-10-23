<script setup lang="ts">
import type { DeveloperApplication, AdminReviewRequest, AdminSuspensionRequest, AdminApplicationDeleteRequest } from '@moetopia/api-client'
import { MoeButton, MoeDropdown, MoeDialog, MoeTextarea, MoeInput, MoeAlert, type MoeMenuItem } from '@moetopia/ui'
const props = defineProps<{ application: DeveloperApplication; reload: () => Promise<void>; compact?: boolean }>()
const emit = defineEmits<{ deleted: [] }>()
const { text, headers, run, error, busy } = useAdminContext()
const action = ref<'approve' | 'reject' | 'suspend' | 'resume' | 'deleteApplication'>('approve')
const open = ref(false)
const reason = ref('')
const confirmation = ref('')
const items = computed<MoeMenuItem[]>(() => [
  ...(props.application.latest.status === 'pending' ? [{ id: 'approve', label: text.value.approve }, { id: 'reject', label: text.value.reject, danger: true }] : []),
  { id: props.application.suspended ? 'resume' : 'suspend', label: props.application.suspended ? text.value.resume : text.value.suspend },
  { id: 'deleteApplication', label: text.value.deleteApplication, danger: true, separatorBefore: true },
])
const ask = (id: string) => { if (!items.value.some(item => item.id === id)) return; action.value = id as typeof action.value; reason.value = ''; confirmation.value = ''; error.value = ''; open.value = true }
const perform = () => run(async () => {
  const revision = props.application.revision
  const endpoint = `/api/v1/admin/applications/${encodeURIComponent(props.application.id)}`
  if (action.value === 'deleteApplication') {
    if (confirmation.value !== props.application.latest.config.name) return
    const body: AdminApplicationDeleteRequest = { revision, reason: reason.value.trim() }
    await $fetch(endpoint, { method: 'DELETE', headers: headers.value, body, retry: 0 })
    open.value = false; emit('deleted'); return
  }
  if (action.value === 'approve' || action.value === 'reject') {
    const body: AdminReviewRequest = { revision, approve: action.value === 'approve', reason: reason.value.trim() }
    await $fetch(`${endpoint}/review`, { method: 'POST', headers: headers.value, body, retry: 0 })
  } else {
    const body: AdminSuspensionRequest = { revision, suspended: action.value === 'suspend', reason: reason.value.trim() }
    await $fetch(`${endpoint}/suspension`, { method: 'POST', headers: headers.value, body, retry: 0 })
  }
  await props.reload(); open.value = false
}, props.reload)
</script>
<template><div class="flex flex-wrap gap-2"><MoeDropdown v-if="compact" :label="text.actions" :items="items.map(item => ({ ...item, disabled: busy }))" @select="ask($event.id)"><MoeButton size="sm" variant="bordered" :disabled="busy">{{ text.actions }}</MoeButton></MoeDropdown><template v-else><MoeButton v-for="item in items" :key="item.id" :color="item.danger ? 'danger' : 'primary'" :variant="item.id === 'approve' ? 'solid' : 'bordered'" :disabled="busy" @click="ask(item.id)">{{ item.label }}</MoeButton></template><MoeDialog v-model:open="open" :title="text[action]" :close-label="text.cancel" :show-close-action="false"><form class="grid gap-4" @submit.prevent="perform"><MoeAlert v-if="action === 'deleteApplication'" :title="text.deleteWarning" color="danger" /><MoeTextarea v-model="reason" :label="text.reason" :required="['reject', 'suspend', 'deleteApplication'].includes(action)" :maxlength="500" :disabled="busy" /><MoeInput v-if="action === 'deleteApplication'" v-model="confirmation" :label="text.deleteConfirmation" :description="application.latest.config.name" autocomplete="off" required :disabled="busy" /><MoeAlert v-if="error" :title="error" color="danger" announce /><div class="flex justify-end gap-3"><MoeButton variant="light" :disabled="busy" @click="open = false">{{ text.cancel }}</MoeButton><MoeButton type="submit" :color="action === 'deleteApplication' || action === 'reject' ? 'danger' : 'primary'" :loading="busy" :disabled="action === 'deleteApplication' && (confirmation !== application.latest.config.name || !reason.trim())">{{ text.confirm }}</MoeButton></div></form></MoeDialog></div></template>

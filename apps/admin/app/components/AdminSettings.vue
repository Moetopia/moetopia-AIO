<script setup lang="ts">
import type { AdminRegistrationPolicy, components } from '@moetopia/api-client'
import { MoePageHeading, MoeCard, MoeSelect, MoeButton, MoeAlert, MoeTable, MoeDialog, MoePasswordInput, MoeSecretDisplay } from '@moetopia/ui'
const { text, headers, context, connect, run, error, busy, logout } = useAdminContext()
const policy = shallowRef<AdminRegistrationPolicy>()
const mode = ref('invite')
const sessions = shallowRef<components['schemas']['SessionsOutputBody']>()
const password = ref('')
const codes = ref<string[]>([])
const action = ref<'codes' | 'session' | 'central'>('codes')
const selected = ref('')
const dialog = ref(false)
const refresh = async () => {
  const [p, s] = await Promise.all([$fetch<AdminRegistrationPolicy>('/api/v1/admin/registration-policy', { retry: 0 }), $fetch<components['schemas']['SessionsOutputBody']>('/api/v1/admin/sessions', { retry: 0 })])
  policy.value = p; mode.value = p.mode; sessions.value = s
}
const load = () => run(refresh)
const save = () => run(async () => { if (policy.value) policy.value = await $fetch<AdminRegistrationPolicy>('/api/v1/admin/registration-policy', { method: 'PUT', headers: headers.value, body: { mode: mode.value, revision: policy.value.revision }, retry: 0 }) }, refresh)
const ask = (kind: typeof action.value, id = '') => { action.value = kind; selected.value = id; password.value = ''; error.value = ''; dialog.value = true }
const perform = () => action.value === 'central' ? logout(true) : run(async () => {
  try {
    if (action.value === 'codes') {
      const result = await $fetch<components['schemas']['AdminRecoveryOutputBody']>('/api/v1/admin/mfa/recovery-codes', { method: 'POST', headers: headers.value, body: { current_password: password.value }, retry: 0 })
      codes.value = result.recovery_codes ?? []; await connect()
    } else {
      await $fetch(`/api/v1/admin/sessions/${encodeURIComponent(selected.value)}`, { method: 'DELETE', headers: headers.value, retry: 0 })
      if (selected.value === sessions.value?.current_id) await connect()
      else await refresh()
    }
    dialog.value = false
  } finally { password.value = '' }
}, refresh)
watch(dialog, open => { if (!open) password.value = '' })
onMounted(load)
onBeforeUnmount(() => { password.value = ''; codes.value = [] })
</script>
<template><div class="grid gap-8"><MoePageHeading :title="text.settings" /><MoeAlert v-if="error && !dialog" :title="error" color="danger" announce /><MoeCard :title="text.registration" padding="lg"><form class="grid gap-4" @submit.prevent="save"><MoeSelect v-model="mode" :label="text.registration" :options="[{ value: 'invite', label: text.inviteMode }, { value: 'open', label: text.openMode }]" /><MoeButton type="submit" :disabled="!policy" :loading="busy">{{ text.save }}</MoeButton></form></MoeCard><MoeCard :title="text.settings" padding="lg"><div class="flex flex-wrap gap-3"><MoeButton :disabled="busy" @click="ask('codes')">{{ text.rotateCodes }} ({{ context?.mfa?.recovery_codes_remaining }})</MoeButton><MoeButton variant="bordered" :disabled="busy" @click="ask('central')">{{ text.centralLogout }}</MoeButton></div></MoeCard>
  <MoeTable :caption="text.sessions" :columns="[{ key: 'id', label: 'ID' }, { key: 'created_at', label: text.created }, { key: 'expires_at', label: text.expires }, { key: 'actions', label: text.actions }]" :rows="sessions?.items ?? []" row-key="id" :empty-title="text.empty"><template #cell-actions="{ row }"><MoeButton size="sm" color="danger" variant="light" :disabled="busy" @click="ask('session', row.id)">{{ text.revoke }} {{ row.id === sessions?.current_id ? `(${text.active})` : '' }}</MoeButton></template></MoeTable>
  <MoeDialog v-model:open="dialog" :title="action === 'codes' ? text.rotateCodes : action === 'central' ? text.centralLogout : text.revoke" :close-label="text.cancel" :show-close-action="false"><form class="grid gap-4" @submit.prevent="perform"><MoePasswordInput v-if="action === 'codes'" v-model="password" :label="text.password" autocomplete="current-password" required /><MoeAlert v-if="error" :title="error" color="danger" announce /><div class="flex justify-end gap-3"><MoeButton variant="light" :disabled="busy" @click="dialog = false">{{ text.cancel }}</MoeButton><MoeButton type="submit" :loading="busy">{{ text.confirm }}</MoeButton></div></form></MoeDialog>
  <MoeDialog :open="codes.length > 0" :title="text.recoveryCode" :close-label="text.savedCodes" @update:open="codes = []"><MoeSecretDisplay :label="text.recoveryCode" :value="codes.join('\n')" :warning="text.secretWarning" :copy-label="text.copy" :copied-label="text.copied" :error-label="text.error" /></MoeDialog>
</div></template>

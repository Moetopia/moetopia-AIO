<script setup lang="ts">
import type { AdminUsers, AdminInvitations, AdminAuditEvents, AdminPrincipals, AdminFailedNotifications, AdminInvitationCreated, AdminUserAvailabilityRequest, AdminPrincipalChangeRequest, AdminNotificationRetryRequest } from '@moetopia/api-client'
import { MoePageHeading, MoeButton, MoeInput, MoePasswordInput, MoeTextarea, MoeTable, MoeAlert, MoeDialog, MoeSecretDisplay, MoeBadge } from '@moetopia/ui'
const props = defineProps<{ section: 'users' | 'invitations' | 'audit' | 'principals' | 'notifications' }>()
const { text, context, headers, run, error, busy, connect } = useAdminContext()
type List = AdminUsers | AdminInvitations | AdminAuditEvents | AdminPrincipals | AdminFailedNotifications
type Row = NonNullable<List['items']>[number]
const result = shallowRef<List>()
const cursor = ref('')
const previous = ref<string[]>([])
const query = ref('')
const appliedQuery = ref('')
const selected = shallowRef<Row>()
const dialog = ref(false)
const issuing = ref(false)
const issued = ref('')
const userId = ref('')
const reason = ref('')
const password = ref('')
const hours = ref('24')
const endpoint = computed(() => `/api/v1/admin/${({ users: 'users', invitations: 'invitations', audit: 'audit-events', principals: 'principals', notifications: 'notifications/failed' })[props.section]}`)
const next = computed(() => result.value && ('next_cursor' in result.value ? result.value.next_cursor : result.value.next_after))
const rows = computed<Row[]>(() => result.value?.items ?? [])
const columns = computed(() => ({ users: [['display_name', text.value.name], ['email', text.value.email], ['enabled', text.value.status], ['deletion_due_at', text.value.deletions], ['actions', text.value.actions]], invitations: [['id', 'ID'], ['expires_at', text.value.expires], ['consumed', text.value.used], ['revoked', text.value.revoke], ['actions', text.value.actions]], audit: [['created_at', text.value.created], ['action', text.value.action], ['actor_id', text.value.actor], ['target_id', text.value.target], ['outcome', text.value.outcome], ['reason', text.value.reason], ['request_id', 'Request ID']], principals: [['id', 'ID'], ['enabled', text.value.status], ['enrolled', text.value.enroll], ['emergency_required', text.value.emergency], ['actions', text.value.actions]], notifications: [['id', 'ID'], ['attempts', text.value.attempts], ['available_at', text.value.created], ['expires_at', text.value.expires], ['actions', text.value.actions]] })[props.section].map(([key, label]) => ({ key: key!, label: label! })))
const load = () => run(refresh)
const refresh = async () => {
  const query: Record<string, string> = {}
  if (cursor.value) query[props.section === 'principals' || props.section === 'notifications' ? 'after' : 'cursor'] = cursor.value
  if (props.section === 'users' && appliedQuery.value) query.query = appliedQuery.value
  result.value = await $fetch<List>(endpoint.value, { query, retry: 0 })
  selected.value = undefined; dialog.value = false
}
const search = () => { appliedQuery.value = query.value.trim(); cursor.value = ''; previous.value = []; void load() }
const page = (forward: boolean) => { if (forward) { previous.value.push(cursor.value); cursor.value = next.value ?? '' } else cursor.value = previous.value.pop() ?? ''; void load() }
const label = computed(() => issuing.value ? props.section === 'invitations' ? text.value.issue : text.value.grant : props.section === 'notifications' ? text.value.retry : props.section === 'invitations' ? text.value.revoke : selected.value && 'enabled' in selected.value && selected.value.enabled ? text.value.disabled : text.value.enabled)
const ask = (row?: Row) => { selected.value = row; issuing.value = !row; reason.value = ''; password.value = ''; userId.value = ''; error.value = ''; dialog.value = true }
const mutate = () => run(async () => {
  try {
    if (props.section === 'invitations') {
      if (issuing.value) { const created = await $fetch<AdminInvitationCreated>('/api/v1/admin/invitations', { method: 'POST', headers: headers.value, body: { expires_in_hours: Number(hours.value) }, retry: 0 }); issued.value = created.code }
      else if (selected.value) await $fetch(`/api/v1/admin/invitations/${encodeURIComponent(selected.value.id)}`, { method: 'DELETE', headers: headers.value, retry: 0 })
    } else if (props.section === 'users' && selected.value && 'email' in selected.value) {
      const body: AdminUserAvailabilityRequest = { revision: selected.value.revision ?? 0, enabled: !selected.value.enabled, reason: reason.value }
      await $fetch(`/api/v1/admin/users/${selected.value.id}/availability`, { method: 'POST', headers: headers.value, body, retry: 0 })
    } else if (props.section === 'principals') {
      const principal = selected.value && 'enrolled' in selected.value ? selected.value : undefined
      const id = principal?.id ?? userId.value.trim()
      const body: AdminPrincipalChangeRequest = { revision: principal?.revision ?? 0, enabled: !principal?.enabled, current_password: password.value, reason: reason.value }
      await $fetch(`/api/v1/admin/principals/${encodeURIComponent(id)}/availability`, { method: 'POST', headers: headers.value, body, retry: 0 })
      if (id === context.value?.user?.id) await connect()
    } else if (props.section === 'notifications' && selected.value && 'attempts' in selected.value) {
      const body: AdminNotificationRetryRequest = { attempts: selected.value.attempts, reason: reason.value }
      await $fetch(`/api/v1/admin/notifications/${selected.value.id}/retry`, { method: 'POST', headers: headers.value, body, retry: 0 })
    }
    dialog.value = false
    if (context.value?.mfa?.verified) await refresh()
  } finally { password.value = '' }
}, refresh)
watch(dialog, open => { if (!open) { password.value = ''; reason.value = '' } })
onMounted(load)
onBeforeUnmount(() => { issued.value = ''; password.value = '' })
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text[section]"><template #actions><MoeButton v-if="section === 'invitations' || section === 'principals'" :disabled="busy" @click="ask()">{{ section === 'invitations' ? text.issue : text.grant }}</MoeButton><MoeButton variant="bordered" :loading="busy" @click="load">{{ text.refresh }}</MoeButton></template></MoePageHeading>
  <form v-if="section === 'users'" class="flex items-end gap-3" @submit.prevent="search"><MoeInput v-model="query" class="flex-1" :label="text.search" type="search" :minlength="query ? 2 : undefined" /><MoeButton type="submit" :disabled="busy">{{ text.refresh }}</MoeButton></form>
  <MoeAlert v-if="error && !dialog" :title="error" color="danger" announce />
  <MoeTable :caption="text[section]" :columns="columns" :rows="rows" row-key="id" :loading="busy && !result" :loading-label="text.refresh" :empty-title="text.empty">
    <template #cell-enabled="{ value }"><MoeBadge :color="value ? 'success' : 'danger'">{{ value ? text.enabled : text.disabled }}</MoeBadge></template>
    <template #cell-actions="{ row }"><MoeButton size="sm" variant="bordered" :disabled="busy || section === 'invitations' && Boolean('consumed' in row && row.consumed || 'revoked' in row && row.revoked)" @click="ask(row)">{{ section === 'notifications' ? text.retry : section === 'invitations' ? text.revoke : 'enabled' in row && row.enabled ? text.disabled : text.enabled }}</MoeButton></template>
  </MoeTable>
  <div class="flex justify-end gap-3"><MoeButton variant="light" :disabled="busy || !previous.length" @click="page(false)">{{ text.previous }}</MoeButton><MoeButton variant="bordered" :disabled="busy || !next" @click="page(true)">{{ text.next }}</MoeButton></div>
  <MoeDialog v-model:open="dialog" :title="label" :close-label="text.cancel" :show-close-action="false"><form class="grid gap-4" @submit.prevent="mutate"><MoeInput v-if="section === 'invitations' && issuing" v-model="hours" :label="text.hours" inputmode="numeric" pattern="[0-9]{1,3}" required /><MoeInput v-if="section === 'principals' && issuing" v-model="userId" :label="text.userId" required /><MoeTextarea v-if="section !== 'invitations'" v-model="reason" :label="text.reason" required :maxlength="500" /><MoePasswordInput v-if="section === 'principals'" v-model="password" :label="text.password" autocomplete="current-password" required /><p v-if="section === 'notifications'" class="text-sm text-muted">{{ text.retryHint }}</p><MoeAlert v-if="error" :title="error" color="danger" announce /><div class="flex justify-end gap-3"><MoeButton variant="light" :disabled="busy" @click="dialog = false">{{ text.cancel }}</MoeButton><MoeButton type="submit" :loading="busy">{{ text.confirm }}</MoeButton></div></form></MoeDialog>
  <MoeDialog :open="!!issued" :title="text.issue" :close-label="text.savedCodes" @update:open="issued = ''"><MoeSecretDisplay :label="text.issue" :value="issued" :warning="text.secretWarning" :copy-label="text.copy" :copied-label="text.copied" :error-label="text.error" /></MoeDialog>
</div></template>

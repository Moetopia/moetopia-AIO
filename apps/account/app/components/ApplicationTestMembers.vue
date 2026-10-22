<script setup lang="ts">
import type { DeveloperApplication, DeveloperTestMembers, DeveloperTestMemberRequest } from '@moetopia/api-client'
import { MoeSectionPanel, MoeInput, MoeButton, MoeTable, MoeAlert, MoeDialog } from '@moetopia/ui'
const props = defineProps<{ application: DeveloperApplication }>()
const emit = defineEmits<{ changed: [] }>()
const { text, common } = useIntegrationMessages()
const { headers, run, error, busy } = useManagementRequest()
const data = shallowRef<DeveloperTestMembers>()
const userId = ref('')
const removing = ref('')
const endpoint = computed(() => `/api/v1/developer/applications/${props.application.id}/test-members`)
const refresh = async () => { data.value = await $fetch<DeveloperTestMembers>(endpoint.value, { retry: 0 }) }
const conflict = async () => { emit('changed'); await refresh() }
const load = () => run(refresh)
const change = (remove = false) => run(async () => {
  if (!data.value) return
  const body: DeveloperTestMemberRequest = { user_id: userId.value.trim(), revision: data.value.revision }
  data.value = remove ? await $fetch<DeveloperTestMembers>(`${endpoint.value}/${encodeURIComponent(removing.value)}`, { method: 'DELETE', headers: headers.value, body: { revision: data.value.revision }, retry: 0 }) : await $fetch<DeveloperTestMembers>(endpoint.value, { method: 'POST', headers: headers.value, body, retry: 0 })
  removing.value = ''; userId.value = ''; emit('changed')
}, conflict)
onMounted(load)
watch(() => props.application.revision, load)
</script>
<template><MoeSectionPanel :title="text.testMembers"><div class="grid gap-4 p-5"><p class="text-sm text-muted">{{ text.testHint }} ({{ data?.items.length ?? 0 }}/{{ data?.limit ?? 20 }})</p><MoeAlert v-if="error" :title="error" color="danger" announce /><form v-if="application.active_version === 0" class="flex items-end gap-3" @submit.prevent="change()"><MoeInput v-model="userId" class="min-w-0 flex-1" :label="common.userId" required /><MoeButton type="submit" :loading="busy" :disabled="!data || data.items.length >= data.limit">{{ text.add }}</MoeButton></form><MoeTable :caption="text.testMembers" :rows="data?.items ?? []" row-key="user_id" :columns="[{ key: 'user_id', label: common.userId }, { key: 'created_at', label: common.created }, { key: 'actions', label: common.actions }]" :empty-title="common.empty"><template #cell-actions="{ row }"><MoeButton size="sm" color="danger" variant="light" :disabled="busy" @click="removing = row.user_id">{{ text.remove }}</MoeButton></template></MoeTable></div><MoeDialog :open="!!removing" :title="text.remove" :description="text.testHint" :close-label="common.cancel" @update:open="removing = ''"><MoeButton color="danger" :loading="busy" @click="change(true)">{{ common.confirm }}</MoeButton></MoeDialog></MoeSectionPanel></template>

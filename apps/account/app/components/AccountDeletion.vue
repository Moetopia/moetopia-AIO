<script setup lang="ts">
import type { AccountDeletion, AccountDeletionRequest } from '@moetopia/api-client'
import { MoeDangerZone, MoeButton, MoeDialog, MoePasswordInput, MoeAlert } from '@moetopia/ui'
const { text, common } = useIntegrationMessages()
const { context } = usePortalContext()
const { headers, busy, run, error } = useManagementRequest()
const open = ref(false)
const password = ref('')
const submit = () => run(async () => {
  const body: AccountDeletionRequest = { current_password: password.value, revision: context.value?.user?.revision ?? 0 }
  try { await $fetch<AccountDeletion>('/api/v1/portal/deletion', { method: 'POST', headers: headers.value, body, retry: 0 }); password.value = ''; await navigateTo('/auth/deletion-requested'); context.value = undefined }
  catch (cause) { if ((cause as { data?: { code?: string } }).data?.code === 'account_published_applications') { error.value = text.value.publishedError; return } throw cause }
  finally { password.value = '' }
})
watch(open, value => { if (!value) password.value = '' })
onBeforeUnmount(() => { password.value = '' })
</script>
<template><MoeDangerZone :title="text.deleteAccount" :description="text.deletionHint"><MoeButton color="danger" variant="flat" :disabled="context?.user?.has_password === false" @click="open = true">{{ text.deleteAccount }}</MoeButton><MoeDialog v-model:open="open" :title="text.deleteAccount" :description="text.deletionHint" :close-label="common.cancel" :show-close-action="false"><form class="grid gap-4" @submit.prevent="submit"><MoePasswordInput v-model="password" :label="common.password" autocomplete="current-password" required /><MoeAlert v-if="error" :title="error" color="danger" announce /><div class="flex justify-end gap-3"><MoeButton variant="light" :disabled="busy" @click="open = false">{{ common.cancel }}</MoeButton><MoeButton type="submit" color="danger" :loading="busy">{{ text.deleteAccount }}</MoeButton></div></form></MoeDialog></MoeDangerZone></template>

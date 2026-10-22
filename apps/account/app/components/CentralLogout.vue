<script setup lang="ts">
import type { CentralLogoutResponse } from '@moetopia/api-client'
import { MoeButton, MoeDialog, MoeAlert } from '@moetopia/ui'
const { text, common } = useIntegrationMessages()
const { headers, run, error, busy } = useManagementRequest()
const open = ref(false)
const { context, sessionList, navigating } = usePortalContext()
const submit = () => run(async () => { const result = await $fetch<CentralLogoutResponse>('/api/v1/portal/central-logout', { method: 'POST', headers: headers.value, retry: 0 }); navigating.value = true; context.value = undefined; sessionList.value = undefined; window.location.assign(result.redirect_to) })
</script>
<template><MoeButton variant="light" color="danger" :disabled="busy" @click="open = true">{{ common.centralLogout }}</MoeButton><MoeDialog v-model:open="open" :title="text.confirmLogout" :description="text.logoutHint" :close-label="common.cancel"><MoeAlert v-if="error" :title="error" color="danger" announce /><MoeButton :loading="busy" @click="submit">{{ common.confirm }}</MoeButton></MoeDialog></template>

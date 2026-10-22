<script setup lang="ts">
import { MoeButton, MoeAlert, MoeSpinner, MoeIcon } from '@moetopia/ui'
const { context, loading, error, text, busy, run, connect, logout } = useAdminContext()
onMounted(() => run(connect))
</script>
<template><PortalHome kind="admin"><template #account><MoeButton v-if="context?.authenticated" size="sm" variant="light" :aria-label="text.logout" :disabled="busy" @click="logout()"><MoeIcon name="logout" /><span class="hidden xl:inline">{{ text.logout }}</span></MoeButton></template>
  <MoeSpinner v-if="loading" :label="text.refresh" />
  <div v-else-if="!context" class="grid gap-4"><MoeAlert :title="error || text.error" color="danger" /><MoeButton @click="run(connect)">{{ text.refresh }}</MoeButton></div>
  <AdminAccess v-else-if="!context.authenticated || !context.mfa?.verified" />
  <slot v-else />
</PortalHome></template>

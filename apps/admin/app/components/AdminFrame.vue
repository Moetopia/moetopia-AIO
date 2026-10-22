<script setup lang="ts">
import { MoeButton, MoeAlert, MoeSpinner, MoeAccountMenu } from '@moetopia/ui'
import type { MoeMenuItem } from '@moetopia/ui'
import { accountMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const accountText = computed(() => accountMessages[locale.value])
const { context, loading, error, text, busy, run, connect, logout } = useAdminContext()
const menu = computed<MoeMenuItem[]>(() => [{ id: 'settings', label: text.value.settings, icon: 'settings' }, { id: 'logout', label: text.value.logout, icon: 'logout', danger: true, separatorBefore: true }])
const select = (item: MoeMenuItem) => item.id === 'logout' ? logout() : navigateTo('/settings')
onMounted(() => run(connect))
</script>
<template><PortalHome kind="admin"><template #account><MoeAccountMenu v-if="context?.authenticated && context.user" :label="accountText.signedIn" :name="context.user.display_name" :email="context.user.email" :src="context.user.avatar_url" :items="menu" @select="select" /></template>
  <MoeSpinner v-if="loading" :label="text.refresh" />
  <div v-else-if="!context" class="grid gap-4"><MoeAlert :title="error || text.error" color="danger" /><MoeButton @click="run(connect)">{{ text.refresh }}</MoeButton></div>
  <AdminAccess v-else-if="!context.authenticated || !context.mfa?.verified" />
  <slot v-else />
</PortalHome></template>

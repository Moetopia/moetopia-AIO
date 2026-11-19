<script setup lang="ts">
import { MoeButton, MoeAlert, MoeSpinner, MoeAccountMenu, MoeIllustration } from '@moetopia/ui'
import type { MoeMenuItem } from '@moetopia/ui'
import { accountMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const accountText = computed(() => accountMessages[locale.value])
const { context, loading, error, text, busy, run, connect, logout } = useAdminContext()
const menu = computed<MoeMenuItem[]>(() => [
  { id: 'overview', label: text.value.overview, icon: 'home' },
  { id: 'applications', label: text.value.applications, icon: 'layers' },
  { id: 'users', label: text.value.users, icon: 'users' },
  { id: 'settings', label: text.value.settings, icon: 'settings' },
  { id: 'logout', label: text.value.logout, icon: 'logout', danger: true, separatorBefore: true, disabled: busy.value },
])
const select = (item: MoeMenuItem) => item.id === 'logout' ? logout() : navigateTo(item.id === 'overview' ? '/' : `/${item.id}`)
onMounted(() => run(connect))
</script>
<template><PortalHome kind="admin"><template #account><MoeAccountMenu v-if="context?.authenticated && context.user" :label="accountText.signedIn" :name="context.user.display_name" :email="context.user.email" :src="context.user.avatar_url" :items="menu" @select="select" /></template>
  <div v-if="loading" class="moe-access-state"><MoeSpinner :label="text.refresh" /></div>
  <div v-else-if="!context" class="moe-access-state mx-auto w-full max-w-md"><MoeIllustration name="service-unavailable" size="md" /><MoeAlert class="w-full" :title="error || text.error" color="danger" /><MoeButton @click="run(connect)">{{ text.refresh }}</MoeButton></div>
  <AdminAccess v-else-if="!context.authenticated || !context.mfa?.verified" />
  <slot v-else />
</PortalHome></template>

<script setup lang="ts">
import { MoeAccountMenu, MoeButton, MoeCard, MoeUserChip, MoeIcon } from '@moetopia/ui'
import type { MoeMenuItem } from '@moetopia/ui'
import { accountMessages, portalMessages } from '@moetopia/locales'
defineProps<{ sidebar?: boolean }>()
const { locale } = usePortalPreferences()
const text = computed(() => accountMessages[locale.value])
const pageText = computed(() => portalMessages[locale.value])
const { context: account, authenticated, logout, busy } = usePortalContext()
const items = computed<MoeMenuItem[]>(() => [
  { id: '/profile', label: pageText.value.profile, icon: 'user' },
  { id: '/sessions', label: pageText.value.sessions, icon: 'shield' },
  { id: '/security', label: pageText.value.security, icon: 'settings' },
  { id: '/developer/applications', label: pageText.value.myApps, icon: 'grid' },
  { id: 'logout', label: text.value.logout, icon: 'logout', danger: true, separatorBefore: true, disabled: busy.value },
])
const select = (item: MoeMenuItem) => item.id === 'logout' ? logout() : navigateTo(item.id)
</script>
<template>
  <MoeCard v-if="sidebar && authenticated && account?.user" class="moe-console-account" padding="sm">
    <MoeUserChip :name="account.user.display_name" :description="account.user.email" :src="account.user.avatar_url" />
    <MoeButton class="mt-3 w-full" color="danger" variant="light" size="sm" :loading="busy" @click="logout"><MoeIcon name="logout" />{{ text.logout }}</MoeButton>
  </MoeCard>
  <MoeAccountMenu v-else-if="!sidebar && authenticated && account?.user" :label="text.signedIn" :name="account.user.display_name" :email="account.user.email" :src="account.user.avatar_url" :items="items" @select="select" />
  <DeveloperLogin v-else-if="!sidebar" />
</template>

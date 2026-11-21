<script setup lang="ts">
import { MoeProvider } from '@moetopia/ui'
import { adminSessionKey } from './composables/useAdminSession'
const { locale } = usePortalPreferences()
const session = useAdminSession()
provide(adminSessionKey, session)
const route = useRoute()
</script>
<template><MoeProvider :locale="locale"><PortalRouteFeedback /><NuxtPage v-if="route.path === '/auth/sign-in-error'" /><AdminFrame v-else><NuxtPage /></AdminFrame></MoeProvider></template>

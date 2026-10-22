<script setup lang="ts">
import { MoeProvider } from '@moetopia/ui'
import { adminSessionKey } from './composables/useAdminSession'
const { locale } = usePortalPreferences()
const session = useAdminSession()
provide(adminSessionKey, session)
</script>
<template><MoeProvider :locale="locale"><PortalRouteFeedback /><AdminFrame><NuxtPage /></AdminFrame></MoeProvider></template>

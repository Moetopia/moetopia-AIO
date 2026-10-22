<script setup lang="ts">
import { MoeProvider } from '@moetopia/ui'
import { accountExperienceKey } from './composables/useAccountExperience'
import { portalSessionKey } from './composables/usePortalSession'
const { locale } = usePortalPreferences()
const experience = useAccountExperience(locale)
provide(accountExperienceKey, experience)
provide(portalSessionKey, usePortalSession(locale))
const route = useRoute()
const identity = computed(() => route.path.startsWith('/auth/') || route.path === '/' && route.query.authorization_request !== undefined)
const layout = computed(() => identity.value ? 'auth' : route.path.startsWith('/developer/applications') || route.path === '/developer/usage' ? 'developer-console' : route.path.startsWith('/developer/docs') ? 'developer-docs' : route.path.startsWith('/developer') ? 'developer' : 'account')
const protectedPage = computed(() => route.path === '/' ? !identity.value : ['/profile', '/sessions', '/security', '/developer/usage'].includes(route.path) || route.path.startsWith('/developer/applications'))
</script>
<template><MoeProvider :locale="locale"><PortalRouteFeedback /><NuxtLayout :name="layout"><PortalGuard :required="protectedPage"><NuxtPage /></PortalGuard></NuxtLayout></MoeProvider></template>

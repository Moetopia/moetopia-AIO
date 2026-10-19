<script setup lang="ts">
import { accountExperienceKey } from './composables/useAccountExperience'
import { portalSessionKey } from './composables/usePortalSession'
const { locale } = usePortalPreferences()
const experience = useAccountExperience(locale)
provide(accountExperienceKey, experience)
provide(portalSessionKey, usePortalSession(locale))
const route = useRoute()
const identity = computed(() => route.path.startsWith('/auth/') || route.path === '/' && route.query.authorization_request !== undefined)
const layout = computed(() => identity.value ? 'auth' : route.path.startsWith('/developer/applications') ? 'developer-console' : route.path.startsWith('/developer/docs') ? 'developer-docs' : route.path.startsWith('/developer') ? 'developer' : 'account')
const protectedPage = computed(() => route.path === '/' ? !identity.value : ['/profile', '/sessions', '/security'].includes(route.path) || route.path.startsWith('/developer/applications'))
</script>
<template><PortalRouteFeedback /><NuxtLayout :name="layout"><PortalGuard :required="protectedPage"><NuxtPage /></PortalGuard></NuxtLayout></template>

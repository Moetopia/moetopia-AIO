<script setup lang="ts">
import { accountExperienceKey } from './composables/useAccountExperience'
const { locale } = usePortalPreferences()
const experience = useAccountExperience(locale)
provide(accountExperienceKey, experience)
const route = useRoute()
const layout = computed(() => route.query.authorization_request !== undefined || route.path.startsWith('/auth/') ? 'auth' : route.path.startsWith('/developer') ? 'developer' : experience.account.value?.user ? 'account' : 'auth')
</script>
<template><PortalRouteFeedback /><NuxtLayout :name="layout"><NuxtPage /></NuxtLayout></template>

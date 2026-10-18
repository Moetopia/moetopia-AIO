<script setup lang="ts">
import { accountExperienceKey } from './composables/useAccountExperience'
const { locale } = usePortalPreferences()
const experience = useAccountExperience(locale)
provide(accountExperienceKey, experience)
const route = useRoute()
const authenticated = computed(() => !!experience.account.value?.user && route.query.authorization_request === undefined)
</script>
<template><PortalHome kind="account" :workspace="authenticated"><NuxtPage /></PortalHome></template>

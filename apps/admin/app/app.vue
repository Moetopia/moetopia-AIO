<script setup lang="ts">
import { MoeProvider } from '@moetopia/ui'
const { locale } = usePortalPreferences()
</script>
<template><MoeProvider :locale="locale"><PortalRouteFeedback /><NuxtPage /></MoeProvider></template>

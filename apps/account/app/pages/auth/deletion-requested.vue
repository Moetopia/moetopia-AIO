<script setup lang="ts">
import { MoePageHeading, MoeLink } from '@moetopia/ui'
const { text } = useIntegrationMessages()
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text.deletionRequested" :description="text.deletionHint" /><MoeLink as-child><NuxtLink to="/auth/cancel-deletion">{{ text.cancelDeletion }}</NuxtLink></MoeLink></div></template>

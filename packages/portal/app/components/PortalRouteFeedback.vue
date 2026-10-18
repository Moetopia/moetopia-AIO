<script setup lang="ts">
import { MoeNavigationProgress } from '@moetopia/ui'
const { text } = usePortalPreferences()
const { isLoading, progress, error } = useLoadingIndicator({ throttle: 100, duration: 1800, hideDelay: 180, resetDelay: 150 })
</script>
<template><MoeNavigationProgress :active="isLoading" :value="progress" :error="error" :label="text.pageLoading" /></template>

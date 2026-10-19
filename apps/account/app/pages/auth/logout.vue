<script setup lang="ts">
import { MoePageHeading, MoeButton } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
const { startLogin, navigating } = usePortalContext()
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text.signedOut" :description="text.localLogoutDescription" /><MoeButton size="lg" :loading="navigating" @click="startLogin('/profile')">{{ text.ssoLogin }}</MoeButton></div></template>

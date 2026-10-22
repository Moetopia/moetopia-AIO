<script setup lang="ts">
import { MoePageHeading, MoeButton } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
const { startLogin, navigating } = usePortalContext()
const route = useRoute()
const { account, loading } = useAccountContext()
const { text: integration } = useIntegrationMessages()
const description = computed(() => route.query.central === '1' || route.query.completed === '1' || route.query.cancelled === '1' ? loading.value ? '' : account.value?.user ? integration.value.logoutCancelled : integration.value.logoutHint : text.value.localLogoutDescription)
</script>
<template><div class="grid gap-6"><MoePageHeading :title="text.signedOut" :description="description" /><MoeButton size="lg" :loading="navigating" @click="startLogin('/profile')">{{ text.ssoLogin }}</MoeButton></div></template>

<script setup lang="ts">
import { MoeSearchPalette } from '@moetopia/ui'
import { portalMessages, managementMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
const m = computed(() => managementMessages[locale.value])
const items = computed(() => [
  { value: '/developer/docs', label: text.value.docsOverview },
  { value: '/developer/docs/authentication', label: text.value.docsAuthentication, description: 'OAuth · OIDC · PKCE' },
  { value: '/developer/docs/credentials', label: text.value.docsCredentials },
  { value: '/developer/docs/endpoints', label: text.value.docsEndpoints, description: 'Discovery · Token · UserInfo · JWKS' },
  { value: '/developer/docs/errors', label: text.value.docsErrors },
])
const select = (value: string) => { if (items.value.some(item => item.value === value)) void navigateTo(value) }
</script>
<template><MoeSearchPalette :title="m.searchDocs" :placeholder="m.searchPlaceholder" :close-label="m.close" :empty-label="m.noSearchResults" :items="items" @select="select" /></template>

<script setup lang="ts">
import { MoeSearchPalette } from '@moetopia/ui'
import { managementMessages, documentationMessages, guideIds } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const m = computed(() => managementMessages[locale.value])
const items = computed(() => [
  ...guideIds.map(id => ({ value: id === 'overview' ? '/developer/docs' : `/developer/docs/${id}`, label: documentationMessages[locale.value].guides[id].title, description: documentationMessages[locale.value].guides[id].description })),
  { value: '/developer/docs/endpoints', label: documentationMessages[locale.value].labels.reference, description: 'Discovery · Token · UserInfo · JWKS' },
])
const select = (value: string) => { if (items.value.some(item => item.value === value)) void navigateTo(value) }
</script>
<template><MoeSearchPalette :title="m.searchDocs" :placeholder="m.searchPlaceholder" :close-label="m.close" :empty-label="m.noSearchResults" :items="items" @select="select" /></template>

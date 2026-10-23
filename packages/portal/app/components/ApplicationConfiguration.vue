<script setup lang="ts">
import type { DeveloperApplicationDraft } from '@moetopia/api-client'
import { adminMessages } from '@moetopia/locales'
import { MoeSectionPanel, MoeDescriptionList } from '@moetopia/ui'
const props = defineProps<{ title: string; config: DeveloperApplicationDraft['config'] }>()
const { locale } = usePortalPreferences()
const text = computed(() => adminMessages[locale.value])
const items = computed(() => [
  { key: 'name', label: text.value.name, value: props.config.name },
  { key: 'description', label: text.value.description, value: props.config.description },
  { key: 'homepage', label: 'Homepage', value: props.config.homepage_url },
  { key: 'redirect', label: 'Redirect URI', value: props.config.redirect_uris?.join('\n') ?? '' },
  { key: 'logout', label: 'Post logout redirect URI', value: props.config.post_logout_redirect_uris?.join('\n') ?? '' },
  { key: 'scopes', label: 'Scope', value: props.config.scopes?.join(' ') ?? '' },
])
</script>
<template><MoeSectionPanel :title="title"><div class="p-5"><MoeDescriptionList :items="items" /></div></MoeSectionPanel></template>

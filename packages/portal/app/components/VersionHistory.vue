<script setup lang="ts">
import type { DeveloperApplicationHistory } from '@moetopia/api-client'
import { adminMessages } from '@moetopia/locales'
import { MoeSectionPanel, MoeButton, MoeDescriptionList, MoeBadge } from '@moetopia/ui'
const props = defineProps<{ history?: DeveloperApplicationHistory; busy?: boolean }>()
defineEmits<{ more: [] }>()
const { locale } = usePortalPreferences()
const text = computed(() => adminMessages[locale.value])
const details = (config: NonNullable<NonNullable<typeof props.history>['items']>[number]['config']) => [
  { label: text.value.name, value: config.name }, { label: 'Homepage', value: config.homepage_url }, { label: 'Redirect URI', value: config.redirect_uris?.join('\n') ?? '' }, { label: 'Post logout redirect URI', value: config.post_logout_redirect_uris?.join('\n') ?? '' }, { label: 'Scope', value: config.scopes?.join(' ') ?? '' },
].map((item, index) => ({ ...item, key: String(index) }))
</script>
<template><MoeSectionPanel :title="text.history"><div class="grid gap-4"><details v-for="item in history?.items" :key="item.version" class="rounded-xl border border-line p-4"><summary class="cursor-pointer font-medium">v{{ item.version }} · {{ item.status }} <MoeBadge v-if="item.version === history?.active_version" size="sm">{{ text.active }}</MoeBadge><span class="ml-2 text-xs text-muted">{{ item.created_at }}</span></summary><div class="mt-4 grid gap-3"><MoeDescriptionList :items="details(item.config)" /><p v-if="item.review_reason" class="text-sm text-muted">{{ item.review_reason }}</p></div></details><p v-if="!history?.items?.length" class="p-4 text-sm text-muted">{{ text.empty }}</p><MoeButton v-if="history?.next_before" :loading="busy" variant="bordered" @click="$emit('more')">{{ text.next }}</MoeButton></div></MoeSectionPanel></template>

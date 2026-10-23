<script setup lang="ts">
import { MoeSelect, MoeAlert, MoeButton } from '@moetopia/ui'
import type { GuideId } from '@moetopia/locales'
import type { DocDetail } from '../../shared/documentation'
const props = defineProps<{ guide: GuideId }>()
const { docs } = useDocumentation()
const groups: Record<GuideId, string[]> = { overview: ['oidc-discovery', 'oauth-jwks'], authentication: ['oauth-authorize', 'oauth-token', 'oauth-userinfo'], credentials: ['oauth-jwks', 'oauth-introspect'], tokens: ['oauth-token', 'oauth-revoke', 'oauth-introspect'], scopes: ['oauth-userinfo'], logout: ['oauth-logout-get'], errors: ['oidc-discovery'], production: ['oidc-discovery', 'oauth-jwks'] }
const selected = ref(groups[props.guide][0]!)
watch(() => props.guide, value => { selected.value = groups[value][0]! })
const { data, error, refresh } = await useFetch<DocDetail>(() => `/developer/docs/operations/${selected.value}`)
</script>
<template><section id="test" class="grid gap-4"><h2>{{ docs.labels.test }}</h2><MoeSelect v-model="selected" :label="docs.labels.testOperation" :options="groups[guide].map(value => ({ value, label: value }))" /><MoeAlert v-if="error" :title="docs.labels.loadError" color="danger" announce><template #actions><MoeButton @click="refresh()">{{ docs.labels.retry }}</MoeButton></template></MoeAlert><DocumentationTryIt v-if="data && !error && data.entry.id === selected" :key="data.entry.id" :detail="data" /></section></template>

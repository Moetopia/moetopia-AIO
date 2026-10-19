<script setup lang="ts">
import type { DeveloperApplicationCreate } from '@moetopia/api-client'
import { MoeInput, MoeTextarea, MoeCheckbox, MoeCheckboxGroup } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const model = defineModel<DeveloperApplicationCreate['config']>({ required: true })
defineProps<{ disabled?: boolean }>()
const { locale } = usePortalPreferences()
const p = computed(() => portalMessages[locale.value])
const { text } = useManagementRequest()
const callbacks = computed({ get: () => model.value.redirect_uris?.join('\n') ?? '', set: value => { model.value = { ...model.value, redirect_uris: value.split('\n') } } })
const scopes = computed({ get: () => model.value.scopes?.filter(scope => scope !== 'openid') ?? [], set: value => { model.value = { ...model.value, scopes: ['openid', ...value] } } })
</script>
<template><div class="grid gap-4">
  <MoeInput v-model="model.name" :label="text.applicationName" :maxlength="80" required :disabled="disabled" />
  <MoeTextarea v-model="model.description" :label="p.appDescription" :rows="3" :maxlength="500" :disabled="disabled" />
  <MoeInput v-model="model.homepage_url" :label="text.homepage" type="url" :maxlength="2048" :disabled="disabled" />
  <MoeTextarea v-model="callbacks" :label="text.redirectURIs" :rows="3" required :disabled="disabled" />
  <MoeCheckbox label="openid" :model-value="true" disabled />
  <MoeCheckboxGroup v-model="scopes" :label="text.scopes" :options="[{ value: 'profile', label: 'profile' }, { value: 'email', label: 'email' }, { value: 'offline_access', label: 'offline_access' }]" :disabled="disabled" />
  <p class="text-sm text-muted">{{ text.configHint }}</p>
</div></template>

<script setup lang="ts">
import type { DeveloperApplicationCreate } from '@moetopia/api-client'
import { MoeInput, MoeTextarea, MoeCheckboxGroup, MoeRepeatableInput } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const model = defineModel<DeveloperApplicationCreate['config']>({ required: true })
withDefaults(defineProps<{ disabled?: boolean; section?: 'all' | 'identity' | 'login' }>(), { section: 'all' })
const { locale } = usePortalPreferences()
const p = computed(() => portalMessages[locale.value])
const { text } = useManagementRequest()
const callbacks = computed<string[]>({ get: () => model.value.redirect_uris?.length ? model.value.redirect_uris : [''], set: value => { model.value = { ...model.value, redirect_uris: value } } })
const scopes = computed({ get: () => model.value.scopes?.filter(scope => scope !== 'openid') ?? [], set: value => { model.value = { ...model.value, scopes: ['openid', ...value] } } })
</script>
<template><div class="grid gap-4">
  <template v-if="section !== 'login'"><MoeInput v-model="model.name" :label="text.applicationName" :maxlength="80" required :disabled="disabled" /><MoeTextarea v-model="model.description" :label="p.appDescription" :rows="3" :maxlength="500" :disabled="disabled" /><MoeInput v-model="model.homepage_url" :label="text.homepage" type="url" :maxlength="2048" :disabled="disabled" /></template>
  <template v-if="section !== 'identity'"><MoeRepeatableInput v-model="callbacks" :label="text.callbacksLabel" :add-label="text.addCallback" :remove-label="text.removeCallback" :description="text.configHint" :max="10" :disabled="disabled" /><MoeCheckboxGroup v-model="scopes" variant="cards" :label="text.scopes" :options="[{ value: 'profile', label: text.scopeProfile, description: text.scopeProfileHint }, { value: 'email', label: text.scopeEmail, description: text.scopeEmailHint }, { value: 'offline_access', label: text.scopeOffline, description: text.scopeOfflineHint }]" :disabled="disabled" /><p class="rounded-lg bg-surface-raised p-3 text-xs leading-6 text-muted">{{ text.pkceHint }}</p></template>
</div></template>

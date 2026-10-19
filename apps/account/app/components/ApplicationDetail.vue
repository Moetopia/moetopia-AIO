<script setup lang="ts">
import type { DeveloperApplication, DeveloperApplicationDraft, components } from '@moetopia/api-client'
import { MoePageHeading, MoeLink, MoeCard, MoeDescriptionList, MoeBadge, MoeButton, MoeAlert, MoeSpinner, MoeDialog, MoePasswordInput, MoeDangerZone } from '@moetopia/ui'
import { portalMessages, accountMessages } from '@moetopia/locales'
const props = defineProps<{ id: string }>()
const { locale } = usePortalPreferences()
const p = computed(() => portalMessages[locale.value])
const a = computed(() => accountMessages[locale.value])
const { text, busy, error, success, headers, run } = useManagementRequest()
const app = ref<DeveloperApplication>()
const config = ref<DeveloperApplicationDraft['config']>({ name: '', description: '', homepage_url: '', redirect_uris: [], scopes: ['openid'] })
const password = ref('')
const secret = ref('')
const action = ref<'submit' | 'availability' | 'secret' | 'delete'>('submit')
const confirming = ref(false)
const editing = ref(false)
const endpoint = computed(() => `/api/v1/developer/applications/${encodeURIComponent(props.id)}`)
const refresh = async () => {
  app.value = await $fetch<DeveloperApplication>(endpoint.value)
  config.value = structuredClone(toRaw(app.value.latest.config))
}
const load = () => run(refresh)
const save = () => run(async () => {
  if (!app.value) return
  const body: DeveloperApplicationDraft = { revision: app.value.revision, config: { ...config.value, redirect_uris: config.value.redirect_uris?.map(uri => uri.trim()).filter(Boolean) ?? [] } }
  app.value = await $fetch<DeveloperApplication>(`${endpoint.value}/draft`, { method: 'PUT', headers: headers.value, body, retry: 0 })
  editing.value = false
}, refresh)
const ask = (next: typeof action.value) => { action.value = next; password.value = ''; error.value = ''; confirming.value = true }
const edit = () => { if (app.value) config.value = structuredClone(toRaw(app.value.latest.config)); error.value = ''; editing.value = true }
watch(confirming, value => { if (!value) password.value = '' })
const labels = computed(() => ({ submit: text.value.submit, availability: app.value?.enabled ? text.value.disable : text.value.enable, secret: text.value.rotate, delete: text.value.deleteApplication }))
const description = computed(() => action.value === 'secret' ? text.value.rotateWarning : action.value === 'delete' ? text.value.deleteWarning : action.value === 'availability' ? text.value.availabilityWarning : text.value.testOnly)
const perform = () => run(async () => {
  if (!app.value) return
  const revision = app.value.revision
  if (action.value === 'secret') {
    const body: components['schemas']['ApplicationSecretInputBody'] = { revision, current_password: password.value }
    try {
      const result = await $fetch<components['schemas']['ApplicationSecretOutputBody']>(`${endpoint.value}/secret`, { method: 'POST', headers: headers.value, body, retry: 0 })
      secret.value = result.client_secret; confirming.value = false; await refresh()
    } finally { password.value = '' }
  } else if (action.value === 'delete') {
    await $fetch(endpoint.value, { method: 'DELETE', headers: headers.value, body: { revision } satisfies components['schemas']['ApplicationRevisionInputBody'], retry: 0 })
    confirming.value = false; await navigateTo('/developer/applications')
  } else if (action.value === 'availability') {
    const body: components['schemas']['ApplicationAvailabilityInputBody'] = { revision, enabled: !app.value.enabled }
    app.value = await $fetch<DeveloperApplication>(`${endpoint.value}/availability`, { method: 'POST', headers: headers.value, body, retry: 0 }); confirming.value = false
  } else {
    app.value = await $fetch<DeveloperApplication>(`${endpoint.value}/submit`, { method: 'POST', headers: headers.value, body: { revision } satisfies components['schemas']['ApplicationRevisionInputBody'], retry: 0 }); confirming.value = false
  }
}, refresh)
const status = computed(() => { const value = app.value?.latest.status ?? ''; return ['draft', 'pending', 'approved', 'rejected'].includes(value) ? text.value[value as 'draft' | 'pending' | 'approved' | 'rejected'] : value })
onMounted(() => { void load() })
onBeforeUnmount(() => { password.value = ''; secret.value = '' })
</script>
<template><div class="grid gap-6">
  <MoeLink as-child class="justify-self-start"><NuxtLink to="/developer/applications">{{ text.back }}</NuxtLink></MoeLink>
  <MoeAlert v-if="error && !confirming" :title="error" color="danger" announce><template #actions><MoeButton :disabled="busy" @click="load">{{ a.retry }}</MoeButton></template></MoeAlert>
  <MoeSpinner v-if="busy && !app" :label="a.checking" />
  <template v-if="app">
    <MoePageHeading :title="app.latest.config.name" :description="app.latest.config.description"><template #actions><MoeBadge>{{ status }}</MoeBadge><MoeBadge :color="app.enabled ? 'success' : 'default'">{{ app.enabled ? text.enabled : text.disabled }}</MoeBadge></template></MoePageHeading>
    <MoeCard layout="form" :title="text.details"><MoeDescriptionList :items="[{ key: 'client', label: 'client_id', value: app.client_id }, { key: 'revision', label: text.version, value: String(app.latest.version) }, { key: 'active', label: text.activeVersion, value: app.active_version ? String(app.active_version) : text.draft }]" /><p class="text-sm text-muted">{{ text.testOnly }}</p><MoeAlert v-if="app.latest.review_reason" :title="app.latest.review_reason" color="warning" /><div class="mt-4 flex flex-wrap gap-3"><MoeButton variant="bordered" :disabled="busy || app.latest.status === 'pending'" @click="edit">{{ text.configuration }}</MoeButton><MoeButton :disabled="busy || !['draft', 'rejected'].includes(app.latest.status)" @click="ask('submit')">{{ text.submit }}</MoeButton></div><p v-if="app.latest.status === 'pending'" class="mt-3 text-sm text-muted">{{ text.reviewUnavailable }}</p></MoeCard>
    <MoeCard layout="form" :title="text.configuration"><MoeDescriptionList :items="[{ key: 'homepage', label: text.homepage, value: app.latest.config.homepage_url || '—' }, { key: 'callbacks', label: text.redirectURIs, value: app.latest.config.redirect_uris?.join('\n') || '—' }, { key: 'scopes', label: text.scopes, value: app.latest.config.scopes?.join(' ') || '—' }]" /><template v-if="app.approved && app.approved.version !== app.latest.version"><h3 class="mt-5 font-semibold">{{ text.activeVersion }} · {{ app.approved.version }}</h3><MoeDescriptionList :items="[{ key: 'name', label: text.applicationName, value: app.approved.config.name }, { key: 'redirects', label: text.redirectURIs, value: app.approved.config.redirect_uris?.join('\n') || '—' }, { key: 'scopes', label: text.scopes, value: app.approved.config.scopes?.join(' ') || '—' }]" /></template></MoeCard>
    <MoeCard layout="form" :title="text.credentials"><MoeDescriptionList :items="[{ key: 'type', label: text.clientType, value: app.public ? 'Public / PKCE S256' : 'Confidential / PKCE S256' }, { key: 'client', label: 'client_id', value: app.client_id }]" /><MoeButton v-if="!app.public" variant="bordered" :disabled="busy" @click="ask('secret')">{{ text.rotate }}</MoeButton></MoeCard>
    <MoeDangerZone :title="app.enabled ? text.disable : text.enable" :description="text.availabilityWarning"><MoeButton color="warning" variant="flat" :disabled="busy" @click="ask('availability')">{{ app.enabled ? text.disable : text.enable }}</MoeButton></MoeDangerZone>
    <MoeDangerZone :title="text.deleteApplication" :description="text.deleteWarning"><MoeButton color="danger" variant="flat" :disabled="busy || app.enabled" @click="ask('delete')">{{ text.deleteApplication }}</MoeButton></MoeDangerZone>
  </template>
  <MoeDialog v-model:open="editing" :title="text.configuration" :description="text.configHint" :close-label="text.close" :show-close-action="false"><form class="grid gap-4" @submit.prevent="save"><ApplicationConfigForm v-model="config" :disabled="busy" /><MoeAlert v-if="error" :title="error" color="danger" announce /><MoeButton type="submit" :loading="busy">{{ text.saveDraft }}</MoeButton></form></MoeDialog>
  <MoeDialog v-model:open="confirming" :title="labels[action]" :description="description" :close-label="text.close" :show-close-action="false"><form class="grid gap-4" @submit.prevent="perform"><MoePasswordInput v-if="action === 'secret'" v-model="password" :label="p.currentPassword" :show-label="a.showPassword" :hide-label="a.hidePassword" autocomplete="current-password" required :disabled="busy" /><MoeAlert v-if="error" :title="error" color="danger" announce /><div class="flex justify-end gap-3"><MoeButton type="button" variant="flat" :disabled="busy" @click="confirming = false">{{ p.cancel }}</MoeButton><MoeButton type="submit" :color="action === 'delete' ? 'danger' : 'primary'" :loading="busy">{{ text.confirm }}</MoeButton></div></form></MoeDialog>
  <ApplicationSecret v-model="secret" />
</div></template>

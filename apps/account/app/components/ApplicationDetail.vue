<script setup lang="ts">
import type { DeveloperApplication, DeveloperApplicationDraft, components } from '@moetopia/api-client'
import { MoePageHeading, MoeLink, MoeIcon, MoeButton, MoeAlert, MoeSpinner, MoeDialog, MoePasswordInput } from '@moetopia/ui'
import { portalMessages, accountMessages } from '@moetopia/locales'
const props = defineProps<{ id: string }>()
const { locale } = usePortalPreferences()
const p = computed(() => portalMessages[locale.value])
const a = computed(() => accountMessages[locale.value])
const { text, busy, error, headers, run } = useManagementRequest()
const app = ref<DeveloperApplication>()
const config = ref<DeveloperApplicationDraft['config']>({ name: '', description: '', homepage_url: '', redirect_uris: [], scopes: ['openid'] })
const password = ref('')
const secret = ref('')
const action = ref<'submit' | 'availability' | 'secret' | 'delete'>('submit')
const confirming = ref(false)
const editing = ref(false)
const editMode = ref<'identity' | 'login'>('identity')
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
const edit = (mode: 'identity' | 'login') => { editMode.value = mode; if (app.value) config.value = structuredClone(toRaw(app.value.latest.config)); error.value = ''; editing.value = true }
watch(confirming, value => { if (!value) password.value = '' })
const labels = computed(() => ({ submit: text.value.submit, availability: app.value?.enabled ? text.value.disable : text.value.enable, secret: text.value.rotate, delete: text.value.deleteApplication }))
const description = computed(() => action.value === 'secret' ? text.value.rotateWarning : action.value === 'delete' ? app.value?.enabled ? text.value.deleteEnabledWarning : text.value.deleteWarning : action.value === 'availability' ? text.value.availabilityWarning : text.value.testOnly)
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
    if (app.value.enabled) app.value = await $fetch<DeveloperApplication>(`${endpoint.value}/availability`, { method: 'POST', headers: headers.value, body: { revision, enabled: false }, retry: 0 })
    await $fetch(endpoint.value, { method: 'DELETE', headers: headers.value, body: { revision: app.value.revision } satisfies components['schemas']['ApplicationRevisionInputBody'], retry: 0 })
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
<template>
  <div class="grid gap-8">
    <MoePageHeading :title="text.details"><template #leading><MoeLink as-child><NuxtLink to="/developer/applications" :aria-label="text.back"><MoeIcon name="arrow-left" /></NuxtLink></MoeLink></template></MoePageHeading>
    <MoeAlert v-if="error && !confirming && !editing" :title="error" color="danger" announce><template #actions><MoeButton :disabled="busy" @click="load">{{ a.retry }}</MoeButton></template></MoeAlert>
    <MoeSpinner v-if="busy && !app" :label="a.checking" />
    <template v-if="app">
      <ApplicationOverview :application="app" :status="status" :busy="busy" @edit="edit('identity')" @delete="ask('delete')" @submit="ask('submit')" @availability="ask('availability')" />
      <ApplicationLoginSettings :application="app" :busy="busy" @edit="edit('login')" />
      <ApplicationCredentialsPanel :application="app" :busy="busy" @rotate="ask('secret')" />
    </template>
    <MoeDialog v-model:open="editing" :title="editMode === 'identity' ? text.editIdentity : text.editLogin" :close-label="text.close" :show-close-action="false">
      <form class="grid gap-5" @submit.prevent="save">
        <ApplicationConfigForm v-model="config" :section="editMode" :disabled="busy" />
        <MoeAlert v-if="error" :title="error" color="danger" announce />
        <div class="flex justify-end gap-3"><MoeButton type="button" color="default" variant="flat" :disabled="busy" @click="editing = false">{{ p.cancel }}</MoeButton><MoeButton type="submit" :loading="busy">{{ text.saveDraft }}</MoeButton></div>
      </form>
    </MoeDialog>
    <MoeDialog v-model:open="confirming" :title="labels[action]" :description="description" :close-label="text.close" :show-close-action="false">
      <form class="grid gap-4" @submit.prevent="perform">
        <MoePasswordInput v-if="action === 'secret'" v-model="password" :label="p.currentPassword" :show-label="a.showPassword" :hide-label="a.hidePassword" autocomplete="current-password" required :disabled="busy" />
        <MoeAlert v-if="error" :title="error" color="danger" announce />
        <div class="flex justify-end gap-3"><MoeButton type="button" color="default" variant="flat" :disabled="busy" @click="confirming = false">{{ p.cancel }}</MoeButton><MoeButton type="submit" :color="action === 'delete' ? 'danger' : 'primary'" :loading="busy">{{ action === 'delete' && app?.enabled ? text.disableAndDelete : text.confirm }}</MoeButton></div>
      </form>
    </MoeDialog>
    <ApplicationSecret v-model="secret" />
  </div>
</template>

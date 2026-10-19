<script setup lang="ts">
import type { DeveloperApplicationCreate, DeveloperApplicationCreated, DeveloperApplication } from '@moetopia/api-client'
import { MoeDialog, MoeCheckbox, MoeButton, MoeAlert } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ created: [application: DeveloperApplication] }>()
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
const { text: m, busy, error, headers, run } = useManagementRequest()
const config = ref<DeveloperApplicationCreate['config']>({ name: '', description: '', homepage_url: '', redirect_uris: [], scopes: ['openid'] })
const publicClient = ref(true)
const secret = ref('')
watch(open, active => { if (active) { config.value = { name: '', description: '', homepage_url: '', redirect_uris: [], scopes: ['openid'] }; publicClient.value = true; error.value = '' } })
const create = () => run(async () => {
  const body: DeveloperApplicationCreate = { public: publicClient.value, config: { ...config.value, redirect_uris: config.value.redirect_uris?.map(uri => uri.trim()).filter(Boolean) ?? [] } }
  const result = await $fetch<DeveloperApplicationCreated>('/api/v1/developer/applications', { method: 'POST', headers: headers.value, body, retry: 0 })
  secret.value = result.client_secret ?? ''; open.value = false; emit('created', result.application)
})
</script>
<template><div><MoeDialog v-model:open="open" :title="text.createAppTitle" :description="m.testOnly" :close-label="text.cancel" :show-close-action="false"><form class="grid gap-4" @submit.prevent="create"><ApplicationConfigForm v-model="config" :disabled="busy" /><MoeCheckbox v-model="publicClient" :label="m.publicClient" :description="m.clientTypeHint" :disabled="busy" /><MoeAlert v-if="error" :title="error" color="danger" announce /><div class="flex justify-end gap-3"><MoeButton type="button" color="default" variant="flat" :disabled="busy" @click="open = false">{{ text.cancel }}</MoeButton><MoeButton type="submit" :loading="busy">{{ text.createAppTitle }}</MoeButton></div></form></MoeDialog><ApplicationSecret v-model="secret" /></div></template>

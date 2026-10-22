<script setup lang="ts">
import type { FederationProviders, ExternalIdentities, FederationRedirect } from '@moetopia/api-client'
import { MoeCard, MoeSettingRow, MoeButton, MoeDialog, MoePasswordInput, MoeAlert, MoeEmptyState } from '@moetopia/ui'
const { context } = usePortalContext()
const { text, common } = useIntegrationMessages()
const { run, headers, error, busy, signedOut } = useManagementRequest()
const providers = shallowRef<FederationProviders>()
const bindings = shallowRef<ExternalIdentities>()
const provider = ref('')
const password = ref('')
const unlink = ref(false)
const refresh = async () => { const [p, b] = await Promise.all([$fetch<FederationProviders>('/api/v1/account/federation/providers', { retry: 0 }), $fetch<ExternalIdentities>('/api/v1/portal/identities', { retry: 0 })]); providers.value = p; bindings.value = b }
const items = computed(() => [...new Set([...(providers.value?.items ?? []), ...(bindings.value?.items?.map(item => item.provider) ?? [])])])
const ask = (id: string, removing: boolean) => { provider.value = id; unlink.value = removing; password.value = ''; error.value = '' }
const submit = () => run(async () => { try { const url = `/api/v1/portal/identities/${encodeURIComponent(provider.value)}`; if (unlink.value) { await $fetch(url, { method: 'DELETE', headers: headers.value, body: { current_password: password.value }, retry: 0 }); provider.value = ''; await signedOut() } else { const result = await $fetch<FederationRedirect>(url, { method: 'POST', headers: headers.value, body: { current_password: password.value }, retry: 0 }); window.location.assign(result.redirect_to) } } finally { password.value = '' } }, refresh)
onMounted(() => run(refresh))
watch(provider, () => { password.value = '' })
onBeforeUnmount(() => { password.value = '' })
</script>
<template><MoeCard layout="form" :title="text.bindings"><MoeAlert v-if="error && !provider" :title="error" color="danger" announce /><MoeEmptyState v-if="providers && !items.length" :title="text.noProviders" /><MoeSettingRow v-for="id in items" :key="id" :title="id === 'github' ? 'GitHub' : id === 'google' ? 'Google' : id" :description="bindings?.items?.find(item => item.provider === id)?.created_at"><template #actions><MoeButton variant="bordered" :disabled="busy || context?.user?.has_password === false" @click="ask(id, !!bindings?.items?.some(item => item.provider === id))">{{ bindings?.items?.some(item => item.provider === id) ? text.unbind : text.bind }}</MoeButton></template></MoeSettingRow><MoeDialog :open="!!provider" :title="unlink ? text.unbind : text.bind" :close-label="common.cancel" :show-close-action="false" @update:open="provider = ''"><form class="grid gap-4" @submit.prevent="submit"><MoePasswordInput v-model="password" :label="common.password" autocomplete="current-password" required /><MoeAlert v-if="error" :title="error" color="danger" announce /><div class="flex justify-end gap-3"><MoeButton variant="light" :disabled="busy" @click="provider = ''">{{ common.cancel }}</MoeButton><MoeButton type="submit" :loading="busy">{{ common.confirm }}</MoeButton></div></form></MoeDialog></MoeCard></template>

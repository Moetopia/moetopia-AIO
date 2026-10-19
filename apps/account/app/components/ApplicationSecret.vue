<script setup lang="ts">
import { MoeDialog, MoePasswordInput, MoeCopyButton } from '@moetopia/ui'
import { accountMessages, portalMessages } from '@moetopia/locales'
const secret = defineModel<string>({ required: true })
const open = computed({ get: () => !!secret.value, set: value => { if (!value) secret.value = '' } })
const { locale } = usePortalPreferences()
const a = computed(() => accountMessages[locale.value])
const p = computed(() => portalMessages[locale.value])
const { text } = useManagementRequest()
onBeforeUnmount(() => { secret.value = '' })
</script>
<template><MoeDialog v-model:open="open" :title="text.credentials" :description="text.secretOnce" :close-label="text.close"><MoePasswordInput :model-value="secret" label="client_secret" readonly autocomplete="off" :show-label="a.showPassword" :hide-label="a.hidePassword" /><MoeCopyButton :value="secret" :label="p.copy" :copied-label="p.copied" :error-label="p.copyFailed" /></MoeDialog></template>

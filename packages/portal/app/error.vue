<script setup lang="ts">
import type { NuxtError } from '#app'
import { MoeProvider, MoeStatusPage, MoeButton, MoeLink } from '@moetopia/ui'
import { feedbackMessages } from '@moetopia/locales'
const props = defineProps<{ error: NuxtError }>()
const { locale, theme } = usePortalPreferences()
const text = computed(() => feedbackMessages[locale.value])
const kind = computed(() => props.error.statusCode === 404 ? 'notFound' : props.error.statusCode === 403 ? 'restricted' : 'unavailable')
const illustration = computed(() => kind.value === 'notFound' ? 'not-found' : kind.value === 'restricted' ? 'access-restricted' : 'service-unavailable')
useHead(() => ({ title: `${props.error.statusCode} · Moetopia`, htmlAttrs: { lang: locale.value, 'data-theme': theme.value } }))
</script>
<template>
  <MoeProvider><MoeStatusPage :code="String(error.statusCode)" :title="text[kind]" :description="text[`${kind}Description`]" :illustration="illustration">
    <template #header><MoeLink href="/">Moetopia</MoeLink><PortalPreferences /></template>
    <MoeButton @click="clearError({ redirect: '/' })">{{ text.home }}</MoeButton><MoeButton v-if="kind === 'unavailable'" variant="bordered" @click="clearError()">{{ text.retry }}</MoeButton>
  </MoeStatusPage></MoeProvider>
</template>

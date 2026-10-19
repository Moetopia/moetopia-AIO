<script setup lang="ts">
import { MoePageHeading, MoeInput, MoePasswordInput, MoeButton, MoeAlert, MoeLink } from '@moetopia/ui'
import { accountMessages } from '@moetopia/locales'
defineProps<{ reset?: boolean }>()
const route = useRoute()
const { locale } = usePortalPreferences()
const text = computed(() => accountMessages[locale.value])
const destination = computed(() => typeof route.query.authorization_request === 'string' && /^[A-Za-z0-9_-]{43}$/.test(route.query.authorization_request) ? { path: '/auth/authorize', query: { authorization_request: route.query.authorization_request } } : '/auth/login')
</script>
<template><div class="grid gap-6"><MoePageHeading :title="reset ? text.resetTitle : text.recoveryTitle" :description="reset ? text.resetDescription : text.recoveryDescription" /><MoeAlert :title="text.recoveryUnavailable" color="info" /><MoePasswordInput v-if="reset" :label="text.password" :show-label="text.showPassword" :hide-label="text.hidePassword" autocomplete="new-password" disabled /><MoeInput v-else :label="text.email" type="email" autocomplete="email" disabled /><MoeButton size="lg" disabled>{{ reset ? text.resetTitle : text.sendCode }}</MoeButton><MoeLink as-child class="justify-self-center"><NuxtLink :to="destination">{{ text.backToLogin }}</NuxtLink></MoeLink></div></template>

<script setup lang="ts">
import { MoeDialog, MoeButton, MoeIcon, MoeLink } from '@moetopia/ui'
import { accountMessages, portalMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const text = computed(() => accountMessages[locale.value])
const pageText = computed(() => portalMessages[locale.value])
const { startLogin, navigating, loading } = usePortalContext()
</script>
<template>
  <MoeDialog :title="pageText.developerLogin" :description="pageText.developerLoginDescription" :close-label="pageText.cancel" :show-close-action="false">
    <template #trigger><MoeButton size="sm" :disabled="loading">{{ text.login }}</MoeButton></template>
    <div class="moe-login-mark"><MoeIcon name="grid" /></div>
    <MoeButton size="lg" class="w-full" :loading="navigating" @click="startLogin('/developer/applications')">{{ pageText.ssoLogin }}</MoeButton>
    <p class="moe-auth-alternative">{{ text.noAccount }}<MoeLink as-child><NuxtLink to="/auth/register?continue=developer-applications">{{ text.registerLink }}</NuxtLink></MoeLink></p>
  </MoeDialog>
</template>

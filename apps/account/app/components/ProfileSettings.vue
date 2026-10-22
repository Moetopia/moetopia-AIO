<script setup lang="ts">
import type { PortalContext } from '@moetopia/api-client'
import { MoeAlert, MoeLink } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
defineProps<{ user: NonNullable<PortalContext['user']>; securityOnly?: boolean }>()
const { locale } = usePortalPreferences()
const pageText = computed(() => portalMessages[locale.value])
const { text } = useIntegrationMessages()
</script>
<template>
  <div class="grid gap-6">
    <ProfileEditor v-if="!securityOnly" />
    <MoeAlert v-if="user.has_password === false" :title="text.passwordNeeded" color="info"><template #actions><MoeLink as-child><NuxtLink to="/auth/forgot-password">{{ pageText.changePassword }}</NuxtLink></MoeLink></template></MoeAlert>
    <template v-else><AccountCredentials mode="email" /><AccountCredentials mode="password" /></template>
    <AccountSecurityRecords v-if="securityOnly" />
    <AccountBindings v-if="securityOnly" />
    <CentralLogout v-if="securityOnly" />
    <AccountDeletion />
  </div>
</template>

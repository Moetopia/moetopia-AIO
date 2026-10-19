<script setup lang="ts">
import type { PortalContext } from '@moetopia/api-client'
import { MoeCard, MoeButton, MoeSettingRow, MoeBadge, MoeDangerZone } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
defineProps<{ user: NonNullable<PortalContext['user']>; securityOnly?: boolean }>()
const { locale } = usePortalPreferences()
const pageText = computed(() => portalMessages[locale.value])
</script>
<template>
  <div class="grid gap-6">
    <ProfileEditor v-if="!securityOnly" />
    <AccountCredentials mode="email" />
    <AccountCredentials mode="password" />
    <AccountSecurityRecords v-if="securityOnly" />
    <MoeCard v-if="securityOnly" layout="form" :title="pageText.providerAccess"><MoeSettingRow :title="'GitHub'" :description="pageText.providerAccessDescription" icon="code"><template #actions><MoeBadge color="secondary">{{ pageText.planned }}</MoeBadge></template></MoeSettingRow><MoeSettingRow title="Google" :description="pageText.providerAccessDescription" icon="user"><template #actions><MoeBadge color="secondary">{{ pageText.planned }}</MoeBadge></template></MoeSettingRow></MoeCard>
    <MoeDangerZone :title="pageText.deleteAccount" :description="pageText.deleteAccountDescription"><MoeButton color="danger" variant="flat" disabled>{{ pageText.deleteAccount }}</MoeButton></MoeDangerZone>
  </div>
</template>

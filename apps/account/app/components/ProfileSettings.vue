<script setup lang="ts">
import type { PortalContext } from '@moetopia/api-client'
import { MoeCard, MoeInput, MoeTextarea, MoeAvatar, MoeButton, MoePasswordInput, MoeLink, MoeSettingRow, MoeBadge, MoeDangerZone } from '@moetopia/ui'
import { accountMessages, portalMessages } from '@moetopia/locales'
defineProps<{ user: NonNullable<PortalContext['user']>; securityOnly?: boolean }>()
const { locale } = usePortalPreferences()
const text = computed(() => accountMessages[locale.value])
const pageText = computed(() => portalMessages[locale.value])
</script>
<template>
  <div class="grid gap-6">
    <MoeCard v-if="!securityOnly" layout="form" :title="pageText.profileTitle" :description="pageText.profileReadOnly">
      <div class="grid gap-4"><div class="flex items-center gap-4"><MoeAvatar :name="user.display_name" size="lg" /><MoeButton disabled variant="bordered" size="sm">{{ pageText.changeAvatar }}</MoeButton></div><MoeInput :model-value="user.display_name" :label="text.name" readonly /><MoeTextarea :label="pageText.biography" :placeholder="pageText.biographyPlaceholder" :rows="3" disabled /><MoeButton class="w-full" disabled>{{ pageText.saveChanges }}</MoeButton></div>
    </MoeCard>
    <MoeCard layout="form" :title="pageText.emailSettings" :description="pageText.emailSettingsDescription"><div class="grid gap-4"><MoeInput :model-value="user.email" :label="text.email" type="email" readonly /><MoeInput :label="pageText.newEmail" type="email" disabled /><MoeButton disabled class="w-full">{{ text.sendCode }}</MoeButton></div></MoeCard>
    <MoeCard layout="form" :title="pageText.changePassword" :description="pageText.passwordAccessDescription"><div class="grid gap-4"><MoePasswordInput :label="pageText.currentPassword" autocomplete="current-password" :show-label="text.showPassword" :hide-label="text.hidePassword" disabled /><MoePasswordInput :label="pageText.newPassword" autocomplete="new-password" :show-label="text.showPassword" :hide-label="text.hidePassword" disabled /><MoePasswordInput :label="pageText.confirmPassword" autocomplete="new-password" :show-label="text.showPassword" :hide-label="text.hidePassword" disabled /><MoeButton disabled class="w-full">{{ pageText.changePassword }}</MoeButton><MoeLink as-child class="justify-self-center"><NuxtLink to="/auth/forgot-password">{{ text.forgotPassword }}</NuxtLink></MoeLink></div></MoeCard>
    <MoeCard v-if="securityOnly" layout="form" :title="pageText.providerAccess"><MoeSettingRow :title="'GitHub'" :description="pageText.providerAccessDescription" icon="code"><template #actions><MoeBadge color="secondary">{{ pageText.planned }}</MoeBadge></template></MoeSettingRow><MoeSettingRow title="Google" :description="pageText.providerAccessDescription" icon="user"><template #actions><MoeBadge color="secondary">{{ pageText.planned }}</MoeBadge></template></MoeSettingRow></MoeCard>
    <MoeDangerZone :title="pageText.deleteAccount" :description="pageText.deleteAccountDescription"><MoeButton color="danger" variant="flat" disabled>{{ pageText.deleteAccount }}</MoeButton></MoeDangerZone>
  </div>
</template>

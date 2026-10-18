<script setup lang="ts">
import { MoePageHeading, MoeCard, MoeEmptyState, MoeAlert, MoeSettingRow, MoeBadge } from '@moetopia/ui'
import { portalMessages, type Locale, type Portal } from '@moetopia/locales'
const props = defineProps<{ locale: Locale; kind: Portal }>()
const text = computed(() => portalMessages[props.locale])
const developer = computed(() => props.kind === 'developer')
</script>
<template>
  <div class="grid gap-7">
    <section id="overview"><MoePageHeading :title="developer ? text.developerTitle : text.adminTitle" :description="developer ? text.developerDescription : text.adminDescription" :eyebrow="developer ? text.developerEyebrow : text.adminEyebrow"><template #actions><MoeBadge color="secondary">{{ text.development }}</MoeBadge></template></MoePageHeading></section>
    <MoeAlert :title="text.development" color="info">{{ text.developmentDescription }}</MoeAlert>
    <MoeCard :id="developer ? 'applications' : 'management'" :title="developer ? text.myApps : text.adminArea" :description="developer ? text.myAppsDescription : text.adminAreaDescription"><MoeEmptyState :icon="developer ? 'code' : 'shield'" :title="developer ? text.appsEmpty : text.adminEmpty" :description="developer ? text.appsEmptyDescription : text.adminEmptyDescription" /></MoeCard>
    <MoeCard id="preparation" :title="developer ? text.prepareTitle : text.managementTitle" :description="developer ? text.prepareDescription : undefined">
      <MoeSettingRow :title="developer ? text.appIdentity : text.users" :description="developer ? text.appIdentityDescription : text.accountsDescription" :icon="developer ? 'grid' : 'user'"><template v-if="!developer" #actions><MoeBadge>{{ text.planned }}</MoeBadge></template></MoeSettingRow>
      <MoeSettingRow :title="developer ? text.callbacks : text.review" :description="developer ? text.callbacksDescription : text.reviewDescription" :icon="developer ? 'code' : 'grid'"><template v-if="!developer" #actions><MoeBadge>{{ text.planned }}</MoeBadge></template></MoeSettingRow>
      <MoeSettingRow :title="developer ? text.permissions : text.security" :description="developer ? text.permissionsDescription : text.securityDescription" icon="shield"><template v-if="!developer" #actions><MoeBadge>{{ text.planned }}</MoeBadge></template></MoeSettingRow>
    </MoeCard>
  </div>
</template>

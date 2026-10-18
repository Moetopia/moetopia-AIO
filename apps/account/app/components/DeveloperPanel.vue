<script setup lang="ts">
import { MoeAlert, MoeBadge, MoeCard, MoeEmptyState, MoeLink, MoePageHeading, MoeProfileLayout, MoeSettingRow, MoeUserChip } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
defineProps<{ section: 'overview' | 'applications' | 'docs' }>()
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
const { account, loading } = useAccountContext()
</script>
<template>
  <div class="grid gap-8">
    <MoePageHeading :title="section === 'overview' ? text.developerIntro : section === 'applications' ? text.myApps : text.docsTitle" :description="section === 'overview' ? text.developerIntroDescription : section === 'applications' ? text.myAppsDescription : text.docsDescription">
      <template v-if="section === 'overview'" #actions><MoeLink as-child><NuxtLink to="/developer/applications">{{ text.developerStart }}</NuxtLink></MoeLink><MoeLink as-child><NuxtLink to="/developer/docs">{{ text.developerDocs }}</NuxtLink></MoeLink></template>
    </MoePageHeading>
    <template v-if="section === 'applications'">
      <MoeCard v-if="account?.user" :title="text.yourAccount"><MoeUserChip :name="account.user.display_name" :description="account.user.email" /></MoeCard>
      <MoeAlert v-else-if="!loading" :title="text.signInNeeded" color="info">{{ text.signInNeededDescription }}<template #actions><MoeLink as-child><NuxtLink to="/auth/login?continue=developer-applications">{{ text.accountSignIn }}</NuxtLink></MoeLink></template></MoeAlert>
      <MoeCard :title="text.myApps"><MoeEmptyState :title="text.appsEmpty" :description="text.appsEmptyDescription" icon="code"><MoeLink as-child><NuxtLink to="/developer/docs">{{ text.developerDocs }}</NuxtLink></MoeLink></MoeEmptyState></MoeCard>
    </template>
    <MoeProfileLayout v-else>
      <template v-if="section === 'overview'" #summary><MoeCard :title="text.oauthSupport"><p class="text-sm leading-7 text-muted">{{ text.oauthSupportDescription }}</p><MoeBadge class="mt-4" color="secondary">{{ text.development }}</MoeBadge></MoeCard></template>
      <MoeCard :title="section === 'overview' ? text.integrationTitle : text.prepareTitle" :description="section === 'overview' ? text.integrationDescription : text.prepareDescription">
        <MoeSettingRow :title="text.appIdentity" :description="text.appIdentityDescription" icon="grid" />
        <MoeSettingRow :title="text.callbacks" :description="text.callbacksDescription" icon="code" />
        <MoeSettingRow :title="text.permissions" :description="text.permissionsDescription" icon="shield" />
        <MoeSettingRow :title="text.applicationReview" :description="text.applicationReviewDescription" icon="check" />
      </MoeCard>
    </MoeProfileLayout>
    <MoeAlert :title="text.development" color="info">{{ text.developmentDescription }}</MoeAlert>
  </div>
</template>

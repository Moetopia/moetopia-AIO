<script setup lang="ts">
import { MoeAlert, MoeBadge, MoeButton, MoeIcon, MoeCard, MoeEmptyState, MoeLink, MoePageHeading, MoeProfileLayout, MoeLandingLayout } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
defineProps<{ section: 'overview' | 'applications' | 'docs' }>()
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
const { account, loading } = useAccountContext()
</script>
<template>
  <div class="grid gap-8">
    <MoeLandingLayout v-if="section === 'overview'" :title="text.developerIntro" :description="text.developerIntroDescription">
      <template #actions><MoeLink as-child><NuxtLink to="/developer/applications">{{ text.developerStart }}</NuxtLink></MoeLink><MoeLink as-child><NuxtLink to="/developer/docs">{{ text.developerDocs }}</NuxtLink></MoeLink></template>
      <MoeProfileLayout><template #summary><MoeCard :title="text.oauthSupport"><p class="text-sm leading-7 text-muted">{{ text.oauthSupportDescription }}</p><MoeBadge class="mt-4" color="secondary">{{ text.development }}</MoeBadge></MoeCard></template><DeveloperGuide /></MoeProfileLayout>
      <MoeAlert :title="text.development" color="info">{{ text.developmentDescription }}</MoeAlert>
    </MoeLandingLayout>
    <template v-else>
      <MoePageHeading :title="section === 'applications' ? text.myApps : text.docsTitle" :description="section === 'applications' ? text.myAppsDescription : text.docsDescription"><template v-if="section === 'applications'" #actions><MoeButton disabled :title="text.createUnavailable"><MoeIcon name="plus" />{{ text.createApplication }}</MoeButton></template></MoePageHeading>
      <template v-if="section === 'applications'">
        <MoeAlert v-if="!account?.user && !loading" :title="text.signInNeeded" color="info">{{ text.signInNeededDescription }}<template #actions><MoeLink as-child><NuxtLink to="/auth/login?continue=developer-applications">{{ text.accountSignIn }}</NuxtLink></MoeLink></template></MoeAlert>
        <MoeCard class="moe-console-empty" padding="none"><MoeEmptyState :title="text.appsEmpty" :description="text.appsEmptyDescription" icon="grid"><MoeLink as-child><NuxtLink to="/developer/docs">{{ text.developerDocs }}</NuxtLink></MoeLink></MoeEmptyState></MoeCard>
      </template>
      <DeveloperGuide v-else />
    </template>
  </div>
</template>

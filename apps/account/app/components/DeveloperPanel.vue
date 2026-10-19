<script setup lang="ts">
import { MoeAlert, MoeBadge, MoeButton, MoeIcon, MoeCard, MoeEmptyState, MoeLink, MoePageHeading, MoeLandingLayout, MoeCodeBlock, MoeFeatureGrid } from '@moetopia/ui'
import { accountMessages, portalMessages } from '@moetopia/locales'
defineProps<{ section: 'overview' | 'applications' }>()
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
const accountText = computed(() => accountMessages[locale.value])
const { feedback } = usePortalContext()
const createOpen = ref(false)
const code = 'GET /.well-known/openid-configuration\n\nresponse_type=code\nscope=openid profile email\ncode_challenge_method=S256'
</script>
<template>
  <div class="grid gap-8">
    <MoeLandingLayout v-if="section === 'overview'" :title="text.developerIntro" :description="text.developerIntroDescription">
      <template #badge><MoeBadge>{{ text.protocolBadge }}</MoeBadge></template>
      <template #actions><MoeLink as-child appearance="solid"><NuxtLink to="/developer/applications">{{ text.developerStart }}</NuxtLink></MoeLink><MoeLink as-child appearance="bordered"><NuxtLink to="/developer/docs">{{ text.developerDocs }}</NuxtLink></MoeLink></template>
      <template #preview><MoeCodeBlock :title="text.codeExampleTitle" :code="code" :copy-label="text.copy" :copied-label="text.copied" :error-label="text.copyFailed" /></template>
      <MoePageHeading heading-level="h2" :title="text.integrationTitle" :description="text.integrationDescription" />
      <MoeFeatureGrid :items="[{ id: 'identity', title: text.appIdentity, description: text.appIdentityDescription, icon: 'grid' }, { id: 'callback', title: text.callbacks, description: text.callbacksDescription, icon: 'code' }, { id: 'scopes', title: text.permissions, description: text.permissionsDescription, icon: 'shield' }]" />
    </MoeLandingLayout>
    <template v-else>
      <MoePageHeading :title="text.myApps" :description="text.myAppsDescription"><template #actions><MoeButton @click="createOpen = true"><MoeIcon name="plus" />{{ text.createApplication }}</MoeButton></template></MoePageHeading>
      <MoeCard class="moe-console-empty" padding="none"><MoeEmptyState :title="text.appsEmpty" :description="text.appsEmptyDescription" icon="grid"><MoeButton @click="createOpen = true"><MoeIcon name="plus" />{{ text.createApplication }}</MoeButton></MoeEmptyState></MoeCard>
      <MoeAlert v-if="feedback" :title="accountText[feedback]" color="danger" announce />
      <ApplicationCreate v-model:open="createOpen" />
    </template>
  </div>
</template>

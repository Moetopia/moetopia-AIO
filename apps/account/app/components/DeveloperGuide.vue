<script setup lang="ts">
import { MoeCard, MoeSettingRow } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
</script>
<template><MoeCard :title="text.prepareTitle" :description="text.prepareDescription"><MoeSettingRow :title="text.appIdentity" :description="text.appIdentityDescription" icon="grid" /><MoeSettingRow :title="text.callbacks" :description="text.callbacksDescription" icon="code" /><MoeSettingRow :title="text.permissions" :description="text.permissionsDescription" icon="shield" /><MoeSettingRow :title="text.applicationReview" :description="text.applicationReviewDescription" icon="check" /></MoeCard></template>

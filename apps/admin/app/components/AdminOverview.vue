<script setup lang="ts">
import { MoePageHeading, MoeAlert, MoeStatGrid, MoeCard, MoeButton, MoeIcon, MoeEmptyState } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
</script>
<template>
  <div class="grid gap-6">
    <MoePageHeading :title="text.dashboard" :description="text.adminDescription" />
    <MoeAlert :title="text.adminUnavailable" color="info" />
    <MoeStatGrid :items="[{ id: 'users', label: text.users, icon: 'user' }, { id: 'applications', label: text.applications, icon: 'grid' }, { id: 'review', label: text.review, icon: 'shield' }]" />
    <MoeCard layout="form" :title="text.registrationTrend"><MoeEmptyState :title="text.adminUnavailable" :description="text.adminEmptyDescription" icon="grid" /></MoeCard>
    <MoeCard layout="form" :title="text.quickActions"><div class="moe-admin-actions"><MoeButton v-for="item in [{ id: 'users', label: text.users, icon: 'user' as const }, { id: 'review', label: text.review, icon: 'grid' as const }, { id: 'security', label: text.security, icon: 'shield' as const }]" :key="item.id" color="default" variant="bordered" disabled><MoeIcon :name="item.icon" />{{ item.label }}</MoeButton></div></MoeCard>
    <MoeCard layout="form" :title="text.recentActivity"><MoeEmptyState :title="text.adminUnavailable" icon="inbox" /></MoeCard>
  </div>
</template>

<script setup lang="ts">
import { MoeNavbar, MoePageFrame, MoeAuthShell, MoeWorkspaceLayout, MoeSidebar, MoeStatusIndicator } from '@moetopia/ui'
import type { MoeNavEntry } from '@moetopia/ui'
import { portalMessages, type Portal } from '@moetopia/locales'
import type { StatusResponse } from '@moetopia/api-client'

const props = defineProps<{ kind: Portal; workspace?: boolean; auth?: boolean }>()
const config = useRuntimeConfig()
const route = useRoute()
const { locale, theme, text } = usePortalPreferences()
const pageText = computed(() => portalMessages[locale.value])
const workspace = computed(() => props.kind === 'admin' || props.kind === 'developer' && props.workspace)
const activeSection = computed(() => props.kind === 'account' ? route.path === '/sessions' ? 'sessions' : route.path === '/security' ? 'security' : 'profile' : props.kind === 'developer' ? route.path.endsWith('/applications') ? 'applications' : route.path.endsWith('/docs') ? 'preparation' : 'overview' : route.hash.slice(1) || 'overview')
const appItems = computed<MoeNavEntry[]>(() => (['account', 'developer', 'admin'] as const).map(kind => {
  const accountUrl = typeof config.public.accountUrl === 'string' ? config.public.accountUrl : ''
  const href = kind === 'developer' ? props.kind === 'admin' ? accountUrl ? `${accountUrl.replace(/\/$/, '')}/developer` : '' : '/developer' : kind === props.kind ? '/' : config.public[`${kind}Url`]
  return { id: `app-${kind}`, label: text.value[kind], href: typeof href === 'string' ? href : '', disabled: !href }
}))
const sectionItems = computed<MoeNavEntry[]>(() => props.kind === 'account' ? [
  { id: 'profile', label: pageText.value.profile, href: '/profile', icon: 'user' },
  { id: 'sessions', label: pageText.value.sessions, href: '/sessions', icon: 'shield' },
  { id: 'security', label: pageText.value.security, href: '/security', icon: 'settings' },
] : props.kind === 'developer' ? [
  { id: 'overview', label: pageText.value.overview, href: '/developer', icon: 'home' },
  { id: 'applications', label: pageText.value.applications, href: '/developer/applications', icon: 'grid' },
  { id: 'preparation', label: pageText.value.preparation, href: '/developer/docs', icon: 'code' },
] : [
  { id: 'overview', label: pageText.value.overview, href: '#overview', icon: 'home' },
  { id: 'users', label: pageText.value.users, href: '#users', icon: 'user', disabled: true },
  { id: 'review', label: pageText.value.review, href: '#review', icon: 'grid', disabled: true },
  { id: 'security', label: pageText.value.security, href: '#security', icon: 'shield', disabled: true },
])
const navigate = (event: MouseEvent, item: MoeNavEntry) => {
  if (event.button !== 0 || event.ctrlKey || event.metaKey || event.altKey || event.shiftKey) return
  if (item.href.startsWith('/')) { event.preventDefault(); return navigateTo(item.href) }
}
const navigation = computed<MoeNavEntry[]>(() => props.kind === 'admin' ? appItems.value : props.kind === 'account' ? [...sectionItems.value, { id: 'developer', label: text.value.developer, href: '/developer', icon: 'code' }] : [{ id: 'account', label: text.value.account, href: '/profile', icon: 'user' }, ...sectionItems.value])
const status = ref<'checking' | 'connected' | 'disconnected'>('checking')

useHead(() => ({ title: `${text.value[props.kind]} · Moetopia`, htmlAttrs: { lang: locale.value, 'data-theme': theme.value } }))

onMounted(async () => {
  try {
    const result = await $fetch<StatusResponse>('/api/v1/status', { timeout: 3000 })
    status.value = result.service === 'moetopia' ? 'connected' : 'disconnected'
  } catch { status.value = 'disconnected' }
})
</script>

<template>
  <MoeAuthShell v-if="auth" :skip-label="pageText.skip" brand-label="Moetopia">
    <template #header><NuxtLink to="/" class="moe-navbar-brand moe-focus"><span>Moetopia<span class="moe-navbar-subtitle">{{ text.account }}</span></span></NuxtLink><PortalPreferences /></template>
    <slot :locale="locale" />
    <template #footer><span>© 2026 Moetopia</span><MoeStatusIndicator :label="text[status]" :state="status === 'connected' ? 'positive' : status === 'disconnected' ? 'negative' : 'pending'" announce /></template>
    <template #brand><div><span class="moe-auth-wordmark">Moetopia</span><p class="moe-auth-brand-detail">{{ pageText.brandDetail }}</p></div></template>
  </MoeAuthShell>
  <MoePageFrame v-else :skip-label="pageText.skip" :density="kind === 'account' ? 'compact' : 'default'">
    <template #header>
      <MoeNavbar brand="Moetopia" :subtitle="text[kind]" :brand-href="kind === 'developer' ? '/developer' : kind === 'account' ? '/profile' : '/'" :label="pageText.navigation" :items="navigation" :mobile-items="navigation" :active="kind === 'admin' ? 'app-admin' : activeSection" :menu-label="pageText.openNavigation" :close-label="pageText.closeNavigation" @navigate="navigate">
        <template #actions><PortalPreferences /></template>
      </MoeNavbar>
    </template>
    <MoeWorkspaceLayout v-if="workspace">
      <template #sidebar><MoeSidebar :label="pageText.workspace" :groups="[{ label: text[kind], items: sectionItems }]" :active="activeSection" collapsible @navigate="navigate"><template #header><span class="pl-2 text-sm font-medium">{{ pageText.workspace }}</span></template><template #footer="{ collapsed }"><p v-if="!collapsed" class="px-2 text-xs text-muted">Moetopia · {{ pageText.identity }}</p></template></MoeSidebar></template>
      <slot :locale="locale"><PortalPreparation :kind="kind" :locale="locale" /></slot>
    </MoeWorkspaceLayout>
    <slot v-else :locale="locale"><PortalPreparation :kind="kind" :locale="locale" /></slot>
    <template #footer><span>© 2026 Moetopia</span><MoeStatusIndicator :label="text[status]" :state="status === 'connected' ? 'positive' : status === 'disconnected' ? 'negative' : 'pending'" announce /></template>
  </MoePageFrame>
</template>

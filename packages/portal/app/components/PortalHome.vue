<script setup lang="ts">
import { MoeNavbar, MoePageFrame, MoeAuthShell, MoeConsoleLayout, MoeSidebar, MoeStatusIndicator } from '@moetopia/ui'
import type { MoeNavEntry } from '@moetopia/ui'
import { portalMessages, type Portal } from '@moetopia/locales'
import type { StatusResponse } from '@moetopia/api-client'

const props = defineProps<{ kind: Portal; workspace?: boolean; auth?: boolean }>()
const route = useRoute()
const { locale, theme, text } = usePortalPreferences()
const pageText = computed(() => portalMessages[locale.value])
const workspace = computed(() => props.kind === 'developer' && props.workspace)
const activeSection = computed(() => props.kind === 'account' ? route.path === '/sessions' ? 'sessions' : route.path === '/security' ? 'security' : 'profile' : props.kind === 'developer' ? route.path.endsWith('/applications') ? 'applications' : route.path.endsWith('/docs') ? 'preparation' : 'overview' : route.hash.slice(1) || 'overview')
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
const navigation = computed<MoeNavEntry[]>(() => props.kind === 'admin' ? [] : props.kind === 'account' ? [sectionItems.value[0]!, { id: 'developer', label: text.value.developer, href: '/developer' }] : [sectionItems.value[0]!, sectionItems.value[2]!, { id: 'console', label: pageText.value.console, href: '/developer/applications' }])
const consoleGroups = computed(() => [{ label: pageText.value.console, items: [sectionItems.value[1]!, sectionItems.value[2]!] }])
const sidebar = computed(() => ({ label: pageText.value.workspace, groups: [{ label: text.value[props.kind], items: sectionItems.value }], active: activeSection.value, collapsible: true, title: props.kind === 'admin' ? 'Moetopia' : pageText.value.workspace, footerText: `Moetopia · ${pageText.value.identity}`, collapseLabel: pageText.value.collapseNavigation, expandLabel: pageText.value.expandNavigation }))
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
    <template #brand><div><span class="moe-auth-wordmark">Moetopia</span><h2 class="moe-auth-headline">{{ pageText.artTitle }}</h2><p class="moe-auth-brand-detail">{{ pageText.artDescription }}</p></div></template>
  </MoeAuthShell>
  <MoePageFrame v-else :skip-label="pageText.skip" :data-portal="kind" :density="kind === 'account' ? 'compact' : kind === 'developer' && route.path === '/developer' ? 'landing' : 'default'">
    <template v-if="kind === 'admin'" #sidebar><MoeSidebar v-bind="sidebar" layout="rail" :title="`Moetopia · ${text.admin}`" :footer-text="undefined" @navigate="navigate" /></template>
    <template #header>
      <MoeNavbar :brand="kind === 'developer' ? `Moetopia ${text.developer}` : kind === 'admin' ? text.admin : 'Moetopia'" :subtitle="kind === 'account' ? text.account : undefined" :brand-href="kind === 'developer' ? '/developer' : kind === 'account' ? '/profile' : '/'" :label="pageText.navigation" :items="navigation" :mobile-items="kind === 'admin' ? sectionItems : workspace ? [navigation[0]!, ...consoleGroups[0]!.items] : navigation" :active="workspace ? 'console' : activeSection" :menu-label="pageText.openNavigation" :close-label="pageText.closeNavigation" :breakpoint="kind === 'admin' ? 'tablet' : 'desktop'" @navigate="navigate">
        <template #actions><PortalPreferences /><slot name="account" /></template>
      </MoeNavbar>
    </template>
    <MoeConsoleLayout v-if="workspace" :label="pageText.console" :groups="consoleGroups" :active="activeSection" @navigate="navigate">
      <template #account><slot name="workspace-account" /></template>
      <slot :locale="locale"><PortalPreparation :kind="kind" :locale="locale" /></slot>
    </MoeConsoleLayout>
    <slot v-else :locale="locale"><PortalPreparation :kind="kind" :locale="locale" /></slot>
    <template v-if="kind === 'developer' && !workspace" #footer><span>© 2026 Moetopia</span><MoeStatusIndicator :label="text[status]" :state="status === 'connected' ? 'positive' : status === 'disconnected' ? 'negative' : 'pending'" announce /></template>
  </MoePageFrame>
</template>

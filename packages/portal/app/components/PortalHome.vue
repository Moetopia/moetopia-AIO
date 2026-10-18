<script setup lang="ts">
import { MoeButton, MoeIcon, MoeNavbar, MoePageFrame, MoeAuthLayout, MoeWorkspaceLayout, MoeSidebar, MoeStatusIndicator } from '@moetopia/ui'
import type { MoeNavEntry } from '@moetopia/ui'
import { portalMessages, type Portal } from '@moetopia/locales'
import type { StatusResponse } from '@moetopia/api-client'

const props = defineProps<{ kind: Portal; workspace?: boolean }>()
const config = useRuntimeConfig()
const route = useRoute()
const { locale, theme, text, toggleLocale, toggleTheme } = usePortalPreferences()
const pageText = computed(() => portalMessages[locale.value])
const workspace = computed(() => props.kind !== 'account' || props.workspace)
const activeSection = computed(() => props.kind === 'account' ? route.path === '/sessions' ? 'sessions' : route.path === '/security' ? 'security' : 'profile' : route.hash.slice(1) || 'overview')
const appItems = computed<MoeNavEntry[]>(() => (['account', 'developer', 'admin'] as const).map(kind => {
  const href = kind === props.kind ? '/' : config.public[`${kind}Url`]
  return { id: `app-${kind}`, label: text.value[kind], href: typeof href === 'string' ? href : '', disabled: !href }
}))
const sectionItems = computed<MoeNavEntry[]>(() => props.kind === 'account' ? [
  { id: 'profile', label: pageText.value.profile, href: '/', icon: 'user' },
  { id: 'sessions', label: pageText.value.sessions, href: '/sessions', icon: 'shield' },
  { id: 'security', label: pageText.value.security, href: '/security', icon: 'settings' },
] : props.kind === 'developer' ? [
  { id: 'overview', label: pageText.value.overview, href: '#overview', icon: 'home' },
  { id: 'applications', label: pageText.value.applications, href: '#applications', icon: 'grid' },
  { id: 'preparation', label: pageText.value.preparation, href: '#preparation', icon: 'code' },
] : [
  { id: 'overview', label: pageText.value.overview, href: '#overview', icon: 'home' },
  { id: 'users', label: pageText.value.users, href: '#users', icon: 'user', disabled: true },
  { id: 'review', label: pageText.value.review, href: '#review', icon: 'grid', disabled: true },
  { id: 'security', label: pageText.value.security, href: '#security', icon: 'shield', disabled: true },
])
const navigate = (event: MouseEvent, item: MoeNavEntry) => {
  if (item.id.startsWith('app-') || event.button !== 0 || event.ctrlKey || event.metaKey || event.altKey || event.shiftKey) return
  if (item.href.startsWith('/')) { event.preventDefault(); return navigateTo(item.href) }
}
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
  <MoePageFrame :skip-label="pageText.skip">
    <template #header>
      <MoeNavbar brand="Moetopia" :subtitle="text[kind]" :label="pageText.navigation" :items="appItems" :mobile-items="workspace ? [...sectionItems, ...appItems] : appItems" :active="`app-${kind}`" :menu-label="pageText.openNavigation" :close-label="pageText.closeNavigation" @navigate="navigate">
        <template #actions><MoeButton variant="light" color="default" icon-only :aria-label="text.themeLabel" @click="toggleTheme"><MoeIcon :name="theme === 'light' ? 'moon' : 'sun'" /></MoeButton><MoeButton variant="light" color="default" size="sm" :aria-label="text.languageLabel" @click="toggleLocale">{{ locale === 'en' ? '中文' : 'EN' }}</MoeButton></template>
      </MoeNavbar>
    </template>
    <MoeWorkspaceLayout v-if="workspace">
      <template #sidebar><MoeSidebar :label="pageText.workspace" :groups="[{ label: text[kind], items: sectionItems }]" :active="activeSection" collapsible @navigate="navigate"><template #header><span class="pl-2 text-sm font-medium">{{ pageText.workspace }}</span></template><template #footer="{ collapsed }"><p v-if="!collapsed" class="px-2 text-xs text-muted">Moetopia · {{ pageText.identity }}</p></template></MoeSidebar></template>
      <slot :locale="locale"><PortalPreparation :kind="kind" :locale="locale" /></slot>
    </MoeWorkspaceLayout>
    <MoeAuthLayout v-else><slot :locale="locale" /></MoeAuthLayout>
    <template #footer><span>© 2026 Moetopia</span><MoeStatusIndicator :label="text[status]" :state="status === 'connected' ? 'positive' : status === 'disconnected' ? 'negative' : 'pending'" announce /></template>
  </MoePageFrame>
</template>

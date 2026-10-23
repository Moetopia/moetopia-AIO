<script setup lang="ts">
import { MoeNavigationTree, MoeInput, MoeAlert, MoeButton, type MoeTreeEntry } from '@moetopia/ui'
import { guideIds } from '@moetopia/locales'
import type { DocEntry } from '../../shared/documentation'
const { docs, common } = useDocumentation()
const route = useRoute()
const { data, error, refresh } = useFetch<DocEntry[]>('/developer/docs/catalog')
const expanded = useState<string[]>('documentation-folders', () => ['guides', 'reference'])
const search = useState('documentation-navigation-search', () => '')
const container = ref<HTMLElement>()
const locate = async () => { await nextTick(); if (container.value?.offsetParent) container.value.querySelector('[aria-current="page"]')?.scrollIntoView({ block: 'nearest' }) }
const active = computed(() => route.path === '/developer/docs' ? 'overview' : String(route.params.operation ?? route.params.section ?? ''))
const items = computed<MoeTreeEntry[]>(() => [{ id: 'guides', label: docs.value.labels.guides, children: guideIds.map(id => ({ id, label: docs.value.guides[id].title, href: id === 'overview' ? '/developer/docs' : `/developer/docs/${id}` })) }, { id: 'reference', label: docs.value.labels.reference, children: [{ id: 'endpoints', label: docs.value.labels.reference, href: '/developer/docs/endpoints' }, ...[...new Set(data.value?.map(entry => entry.group))].map(group => ({ id: `group:${group}`, label: group, children: data.value?.filter(entry => entry.group === group).map(entry => ({ id: entry.id, label: entry.title, method: entry.method, description: entry.path, href: `/developer/docs/api/${entry.id}` })) ?? [] }))] }])
const visible = computed(() => {
  const query = search.value.trim().toLowerCase()
  const filter = (entries: MoeTreeEntry[]): MoeTreeEntry[] => entries.flatMap(item => {
    if (!query || `${item.label} ${item.description ?? ''} ${item.id}`.toLowerCase().includes(query)) return [item]
    const children = item.children ? filter(item.children) : []
    return children.length ? [{ ...item, children }] : []
  })
  return filter(items.value)
})
watch([active, data], () => {
  const group = data.value?.find(entry => entry.id === active.value)?.group
  expanded.value = [...new Set([...expanded.value, ...(group ? ['reference', `group:${group}`] : ['guides'])])]
}, { immediate: true })
watch(search, value => { if (value.trim()) expanded.value = ['guides', 'reference', ...[...new Set(data.value?.map(entry => `group:${entry.group}`))]] })
watch(active, () => { search.value = ''; void locate() })
onMounted(locate)
const navigate = (event: MouseEvent, item: MoeTreeEntry) => { if (!item.href || event.button || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return; event.preventDefault(); (event.currentTarget as HTMLElement).closest('.moe-docs-mobile-navigation')?.removeAttribute('open'); void navigateTo(item.href) }
</script>
<template><div ref="container" class="grid gap-3"><MoeInput v-model="search" type="search" :label="docs.labels.title" :placeholder="docs.labels.searchPlaceholder" size="sm" /><MoeNavigationTree v-model:expanded="expanded" :label="common.docsNavigation" :items="visible" :active="active" @navigate="navigate" /><MoeAlert v-if="error" :title="docs.labels.loadError" color="danger"><template #actions><MoeButton size="sm" @click="refresh()">{{ docs.labels.retry }} · {{ docs.labels.guides }}</MoeButton></template></MoeAlert></div></template>

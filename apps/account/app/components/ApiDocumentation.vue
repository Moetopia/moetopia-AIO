<script setup lang="ts">
import { MoePageHeading, MoeEndpointList, MoeInput, MoeSelect, MoeLink, MoeAlert, MoeButton, MoeSpinner, MoeCopyButton } from '@moetopia/ui'
import type { DocEntry } from '../../shared/documentation'
const { docs, common } = useDocumentation()
const route = useRoute()
const { data, error, status, refresh } = await useFetch<DocEntry[]>('/developer/docs/catalog')
const filter = (name: string, fallback = '') => computed({ get: () => typeof route.query[name] === 'string' ? String(route.query[name]) : fallback, set: value => { void navigateTo({ path: route.path, query: { ...route.query, [name]: value === fallback || !value ? undefined : value } }, { replace: true }) } })
const search = filter('q')
const group = filter('group', '__all__')
const method = filter('method', '__all__')
const groups = computed(() => [...new Set(data.value?.map(entry => entry.group))])
const methods = computed(() => [...new Set(data.value?.map(entry => entry.method))].sort())
const filtered = computed(() => data.value?.filter(entry => (group.value === '__all__' || entry.group === group.value) && (method.value === '__all__' || entry.method === method.value) && `${entry.path} ${entry.title} ${entry.id}`.toLowerCase().includes(search.value.trim().toLowerCase())) ?? [])
const sections = computed(() => groups.value.map(group => ({ group, items: filtered.value.filter(entry => entry.group === group).map(entry => ({ ...entry, href: `/developer/docs/api/${entry.id}` })) })).filter(section => section.items.length))
const documentText = computed(() => [docs.value.labels.reference, ...filtered.value.map(entry => `${entry.method} ${entry.path}\n${entry.title}`)].join('\n\n'))
const navigate = (event: MouseEvent, item: { href: string }) => { if (event.button || event.ctrlKey || event.metaKey || event.altKey || event.shiftKey) return; event.preventDefault(); void navigateTo(item.href) }
useHead(() => ({ title: `${docs.value.labels.reference} · Moetopia` }))
</script>
<template><div class="grid gap-6 min-w-0">
  <MoePageHeading :title="docs.labels.reference" :description="docs.labels.source"><template #actions><MoeLink href="/developer/docs/openapi.json" download="moetopia-openapi.json">{{ docs.labels.download }}</MoeLink><MoeCopyButton :value="documentText" :label="common.copy" :copied-label="common.copied" :error-label="common.copyFailed" /></template></MoePageHeading>
  <div class="grid items-end gap-3 sm:grid-cols-2 lg:grid-cols-3"><MoeInput v-model="search" type="search" :label="docs.labels.search" :placeholder="docs.labels.searchPlaceholder" /><MoeSelect v-model="group" :label="docs.labels.group" :options="[{ value: '__all__', label: docs.labels.all }, ...groups.map(value => ({ value, label: value }))]" /><MoeSelect v-model="method" :label="docs.labels.method" :options="[{ value: '__all__', label: docs.labels.all }, ...methods.map(value => ({ value, label: value }))]" /></div>
  <p class="text-xs text-muted">{{ docs.labels.languageNote }}</p>
  <MoeSpinner v-if="status === 'pending'" :label="docs.labels.reference" />
  <MoeAlert v-if="error" :title="docs.labels.loadError" color="danger" announce><template #actions><MoeButton @click="refresh()">{{ docs.labels.retry }}</MoeButton></template></MoeAlert>
  <template v-else><p class="text-sm text-muted" aria-live="polite">{{ filtered.length }} {{ docs.labels.count }}</p><section v-for="section in sections" :key="section.group" class="grid gap-3"><h2 class="text-base font-semibold">{{ section.group }}</h2><MoeEndpointList :label="section.group" :items="section.items" :empty-label="docs.labels.empty" @navigate="navigate" /></section><p v-if="data && !filtered.length" class="py-8 text-muted">{{ docs.labels.empty }}</p></template>
</div></template>

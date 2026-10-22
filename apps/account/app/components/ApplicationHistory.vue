<script setup lang="ts">
import type { DeveloperApplicationHistory } from '@moetopia/api-client'
import { MoeAlert } from '@moetopia/ui'
const props = defineProps<{ id: string; revision: number }>()
const { run, error, busy } = useManagementRequest()
const history = shallowRef<DeveloperApplicationHistory>()
const load = (more = false) => run(async () => { const page = await $fetch<DeveloperApplicationHistory>(`/api/v1/developer/applications/${props.id}/versions`, { query: { before: more ? history.value?.next_before : 0 }, retry: 0 }); history.value = { ...page, items: more ? [...(history.value?.items ?? []), ...(page.items ?? [])] : page.items } })
onMounted(() => load())
watch(() => props.revision, () => load())
</script>
<template><div class="grid gap-3"><MoeAlert v-if="error" :title="error" color="danger" announce /><VersionHistory :history="history" :busy="busy" @more="load(true)" /></div></template>

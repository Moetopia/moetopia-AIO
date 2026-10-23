<script setup lang="ts">
import { MoePageHeading, MoeDocumentLayout, MoeSchemaFields, MoeCodeBlock, MoeSelect, MoeAlert, MoeButton, MoeLink, MoeSpinner, MoeHttpMethod, MoeCopyButton } from '@moetopia/ui'
import type { DocDetail, DocParameter } from '../../shared/documentation'
import { schemaFields } from '../utils/documentation-schema'
import { documentationExamples } from '../utils/documentation-examples'
const props = defineProps<{ id: string }>()
const { docs, common } = useDocumentation()
const { data, error, status, refresh } = await useFetch<DocDetail>(() => `/developer/docs/operations/${encodeURIComponent(props.id)}`)
if (error.value?.statusCode === 404) throw createError({ statusCode: 404, statusMessage: 'API operation not found' })
const operation = computed(() => data.value?.operation)
const protocol = computed(() => data.value?.entry.group === 'OAuth / OIDC')
const outline = computed(() => ['authentication', 'parameters', 'body', 'responses', 'test', 'raw'].map(id => ({ id, label: docs.value.labels[id as 'authentication' | 'parameters' | 'body' | 'responses' | 'test' | 'raw'] })))
const parameters = computed(() => operation.value?.parameters?.map(parameter => ({ ...schemaFields({ type: 'object', properties: { [parameter.name]: parameter.schema ?? {} }, required: parameter.required ? [parameter.name] : [] }, data.value?.schemas ?? {})[0]!, description: [parameter.in, parameter.description ?? parameter.schema?.description].filter(Boolean).join(' · ') })) ?? [])
const bodyMedia = ref('')
const responseStatus = ref('')
const responseMedia = ref('')
const bodyTypes = computed(() => Object.keys(operation.value?.requestBody?.content ?? {}))
const responseTypes = computed(() => Object.keys(operation.value?.responses ?? {}))
const response = computed(() => operation.value?.responses?.[responseStatus.value])
const responseMediaTypes = computed(() => Object.keys(response.value?.content ?? {}))
watch(bodyTypes, values => { bodyMedia.value = values[0] ?? '' }, { immediate: true })
watch(responseTypes, values => { responseStatus.value = values[0] ?? '' }, { immediate: true })
watch(responseMediaTypes, values => { responseMedia.value = values[0] ?? '' }, { immediate: true })
const requestFields = computed(() => schemaFields(operation.value?.requestBody?.content?.[bodyMedia.value]?.schema, data.value?.schemas ?? {}))
const responseFields = computed(() => schemaFields(response.value?.content?.[responseMedia.value]?.schema, data.value?.schemas ?? {}))
const headerFields = computed(() => Object.entries(response.value?.headers ?? {}).map(([name, header]: [string, DocParameter]) => ({ name, type: String(header.schema?.type ?? 'string'), required: !!header.required, description: header.description })))
const security = computed(() => [...new Set(operation.value?.security?.flatMap(requirement => Object.keys(requirement)) ?? [])].map(name => ({ name, ...data.value?.securitySchemes[name] })))
const example = computed(() => documentationExamples[({ 'oidc-discovery': 'discovery', 'oauth-authorize': 'authorize', 'oauth-token': 'token', 'oauth-userinfo': 'userinfo', 'oauth-introspect': 'introspect', 'oauth-logout-get': 'logout' } as Record<string, string>)[props.id] ?? ''])
const documentText = computed(() => data.value ? `${data.value.entry.method} ${data.value.entry.path}\n${data.value.entry.title}\n\n${JSON.stringify({ operation: operation.value, schemas: data.value.schemas }, null, 2)}` : '')
useHead(() => ({ title: `${data.value?.entry.title ?? docs.value.labels.reference} · Moetopia` }))
</script>
<template><MoeDocumentLayout :label="docs.labels.outline" :items="outline">
  <div class="flex flex-wrap items-center justify-between gap-3"><MoeLink as-child><NuxtLink to="/developer/docs/endpoints">{{ docs.labels.back }}</NuxtLink></MoeLink><MoeCopyButton v-if="data" :value="documentText" :label="common.copy" :copied-label="common.copied" :error-label="common.copyFailed" /></div>
  <MoeSpinner v-if="status === 'pending'" :label="docs.labels.reference" />
  <MoeAlert v-if="error" :title="docs.labels.loadError" color="danger" announce><template #actions><MoeButton @click="refresh()">{{ docs.labels.retry }}</MoeButton></template></MoeAlert>
  <template v-if="data && operation">
    <MoePageHeading :title="data.entry.title" :description="operation.description" />
    <p class="text-xs text-muted">{{ docs.labels.languageNote }}</p>
    <div class="flex flex-wrap items-center gap-3"><MoeHttpMethod :method="data.entry.method" /><code class="break-all text-sm">{{ data.entry.path }}</code></div>
    <MoeAlert v-if="operation.deprecated" :title="docs.labels.deprecated" color="warning" />
    <section id="authentication"><h2>{{ docs.labels.authentication }}</h2><p>{{ protocol ? docs.labels.protocolNote : docs.labels.internalNote }}</p><dl v-for="scheme in security" :key="scheme.name"><dt class="font-semibold">{{ scheme.name }}</dt><dd class="mt-2 text-sm text-muted">{{ scheme.description }}</dd></dl></section>
    <section v-if="example"><MoeCodeBlock :title="docs.labels.requestExample" :code="example" :copy-label="common.copy" :copied-label="common.copied" :error-label="common.copyFailed" /><p class="text-xs text-muted">{{ docs.labels.exampleNote }}</p></section>
    <section id="parameters"><h2>{{ docs.labels.parameters }}</h2><MoeSchemaFields :fields="parameters" :required-label="docs.labels.required" :optional-label="docs.labels.optional" :empty-label="docs.labels.none" /></section>
    <section id="body"><h2>{{ docs.labels.body }}</h2><p v-if="operation.requestBody">{{ operation.requestBody.required ? docs.labels.required : docs.labels.optional }} · {{ operation.requestBody.description }}</p><MoeSelect v-if="bodyTypes.length" v-model="bodyMedia" :label="docs.labels.media" :options="bodyTypes.map(value => ({ value, label: value }))" /><MoeSchemaFields :fields="requestFields" :required-label="docs.labels.required" :optional-label="docs.labels.optional" :empty-label="docs.labels.noBody" /></section>
    <section id="responses"><h2>{{ docs.labels.responses }}</h2><div class="grid gap-3 sm:grid-cols-2"><MoeSelect v-model="responseStatus" :label="docs.labels.status" :options="responseTypes.map(value => ({ value, label: value }))" /><MoeSelect v-if="responseMediaTypes.length" v-model="responseMedia" :label="docs.labels.media" :options="responseMediaTypes.map(value => ({ value, label: value }))" /></div><p>{{ response?.description }}</p><template v-if="headerFields.length"><h3 class="font-semibold">{{ docs.labels.headers }}</h3><MoeSchemaFields :fields="headerFields" :required-label="docs.labels.required" :optional-label="docs.labels.optional" :empty-label="docs.labels.none" /></template><MoeSchemaFields :fields="responseFields" :required-label="docs.labels.required" :optional-label="docs.labels.optional" :empty-label="responseMediaTypes.length ? docs.labels.none : docs.labels.noBody" /></section>
    <section id="test"><DocumentationTryIt :detail="data" /></section>
    <section id="raw"><h2>{{ docs.labels.raw }}</h2><details><summary class="moe-focus cursor-pointer text-sm text-primary">{{ data.entry.id }} · JSON</summary><MoeCodeBlock :title="docs.labels.raw" :code="JSON.stringify({ operation, schemas: data.schemas }, null, 2)" :copy-label="common.copy" :copied-label="common.copied" :error-label="common.copyFailed" /></details><p class="text-xs text-muted">{{ docs.labels.source }}</p></section>
  </template>
</MoeDocumentLayout></template>

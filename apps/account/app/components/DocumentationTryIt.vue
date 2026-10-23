<script setup lang="ts">
import { MoeRequestPanel, MoeInput, MoePasswordInput, MoeSelect, MoeTextarea, MoeAlert, MoeButton, MoeSecretDisplay, MoeCodeBlock } from '@moetopia/ui'
import type { DocDetail } from '../../shared/documentation'
import { documentationExamples } from '../utils/documentation-examples'
const props = defineProps<{ detail: DocDetail }>()
const { docs, common } = useDocumentation()
const { values, json, bearer, verifier, response, status, elapsed, error, busy, fields, blocked, navigation, bodyType, run, clear, pkce, cancel } = useDocumentationRequest(computed(() => props.detail))
const options = (values: unknown[]) => values.filter(value => typeof value === 'string' && value !== 'client_credentials').map(value => ({ value: String(value), label: String(value) }))
const secret = (name: string) => ['code', 'code_verifier', 'refresh_token', 'token', 'id_token_hint'].includes(name)
</script>
<template><MoeRequestPanel :title="docs.labels.test" :hint="docs.labels.testHint" :run-label="navigation ? docs.labels.openTest : docs.labels.run" :clear-label="docs.labels.clearTest" :cancel-label="docs.labels.cancelTest" :response-label="docs.labels.result" :elapsed-label="docs.labels.elapsed" :copy-label="common.copy" :copied-label="common.copied" :error-label="common.copyFailed" :busy="busy" :disabled="!!blocked" :response="response" :status="status" :elapsed="elapsed" @run="run" @clear="clear" @cancel="cancel">
  <MoeAlert v-if="blocked" :title="blocked" /><template v-else><div v-for="field in fields" :key="field.key"><MoeSelect v-if="field.schema.enum" :model-value="values[field.key]" :label="`${field.name} · ${field.location}`" :options="options(field.schema.enum)" :required="field.required" @update:model-value="values[field.key] = $event ?? ''" /><MoePasswordInput v-else-if="secret(field.name)" v-model="values[field.key]" :label="`${field.name} · ${field.location}`" autocomplete="off" :required="field.required" :disabled="busy" /><MoeInput v-else v-model="values[field.key]" :label="`${field.name} · ${field.location}${field.name === 'client_id' && detail.entry.path.startsWith('/oauth/') ? ' · Public' : ''}`" :description="field.schema.description" autocomplete="off" :required="field.required" :disabled="busy" /></div>
  <MoeTextarea v-if="bodyType === 'application/json'" v-model="json" :label="docs.labels.requestBody" autocomplete="off" :disabled="busy" :rows="8" spellcheck="false" />
  <MoePasswordInput v-if="detail.entry.path === '/oauth/userinfo'" v-model="bearer" :label="docs.labels.bearer" autocomplete="off" required :disabled="busy" />
  <MoeButton v-if="detail.entry.path === '/oauth/authorize'" variant="bordered" :disabled="busy" @click="pkce">{{ docs.labels.pkce }}</MoeButton>
  <MoeSecretDisplay v-if="verifier" :value="verifier" :label="docs.labels.verifier" :warning="docs.labels.testSecret" :copy-label="common.copy" :copied-label="common.copied" :error-label="common.copyFailed" /></template>
  <MoeAlert v-if="error" :title="error" color="warning" announce />
  <MoeCodeBlock v-if="detail.entry.path === '/oauth/introspect'" :title="docs.labels.terminal" :code="documentationExamples.introspect ?? ''" :copy-label="common.copy" :copied-label="common.copied" :error-label="common.copyFailed" />
</MoeRequestPanel></template>

<script setup lang="ts">
import type { DeveloperApplication } from '@moetopia/api-client'
import { MoeResourceSummary, MoeBadge, MoeButton, MoeAlert, MoeIcon } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const props = defineProps<{ application: DeveloperApplication; status: string; busy: boolean }>()
defineEmits<{ edit: []; delete: []; submit: []; availability: [] }>()
const { locale } = usePortalPreferences()
const p = computed(() => portalMessages[locale.value])
const { text } = useManagementRequest()
const facts = computed(() => [
  { key: 'type', label: text.value.clientType, value: props.application.public ? 'Public' : 'Confidential' },
  { key: 'latest', label: text.value.version, value: `${props.application.latest.version} · ${props.status}` },
  { key: 'active', label: text.value.activeVersion, value: props.application.active_version ? String(props.application.active_version) : text.value.draft },
  { key: 'created', label: text.value.createdAt, value: new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium' }).format(new Date(props.application.created_at)) },
])
</script>
<template><MoeResourceSummary :title="application.latest.config.name" :description="application.latest.config.description" :identifier="application.client_id" :items="facts" :copy-label="p.copy" :copied-label="p.copied" :error-label="p.copyFailed">
  <template #badges><MoeBadge :color="application.suspended ? 'danger' : application.enabled ? 'success' : 'default'">{{ application.suspended ? text.suspended : application.enabled ? text.enabled : text.disabled }}</MoeBadge></template>
  <template #actions><MoeButton size="sm" variant="flat" :aria-label="text.editIdentity" :disabled="busy || application.latest.status === 'pending'" @click="$emit('edit')"><MoeIcon name="pencil" class="size-4" />{{ text.edit }}</MoeButton><MoeButton size="sm" color="danger" variant="flat" :disabled="busy" @click="$emit('delete')"><MoeIcon name="trash" class="size-4" />{{ text.deleteApplication }}</MoeButton></template>
  <MoeAlert v-if="application.suspended" class="mt-4" :title="text.suspensionHint" color="danger">{{ application.suspension_reason }}</MoeAlert>
  <MoeAlert v-if="application.latest.review_reason" class="mt-4" :title="application.latest.review_reason" color="warning" />
  <MoeAlert v-else-if="application.latest.status === 'pending'" class="mt-4" :title="text.reviewPending" color="warning" />
  <div class="mt-4 flex flex-wrap items-center gap-3"><MoeButton v-if="['draft', 'rejected'].includes(application.latest.status)" size="sm" :disabled="busy" @click="$emit('submit')">{{ text.submit }}</MoeButton><MoeButton size="sm" color="default" variant="light" :disabled="busy" @click="$emit('availability')">{{ application.enabled ? text.disable : text.enable }}</MoeButton><span class="text-xs text-muted">{{ text.reviewAccessHint }}</span></div>
</MoeResourceSummary></template>

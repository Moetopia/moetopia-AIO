<script setup lang="ts">
import type { DeveloperApplication } from '@moetopia/api-client'
import { MoeSectionPanel, MoeButton, MoeBadge, MoeCopyButton, MoeAccordion, MoeIcon } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
defineProps<{ application: DeveloperApplication; busy: boolean }>()
defineEmits<{ edit: [] }>()
const { locale } = usePortalPreferences()
const p = computed(() => portalMessages[locale.value])
const { text } = useManagementRequest()
</script>
<template><MoeSectionPanel :title="text.userLogin">
  <template #actions><MoeButton size="sm" color="default" variant="bordered" :aria-label="text.editLogin" :disabled="busy || application.latest.status === 'pending'" @click="$emit('edit')"><MoeIcon name="pencil" class="size-4" />{{ text.edit }}</MoeButton></template>
  <dl class="moe-detail-fields"><div><dt>{{ text.callbacksLabel }}</dt><dd class="grid gap-2"><div v-for="uri in application.latest.config.redirect_uris" :key="uri" class="moe-copy-row"><code>{{ uri }}</code><MoeCopyButton :value="uri" :label="p.copy" :copied-label="p.copied" :error-label="p.copyFailed" /></div></dd></div><div><dt>{{ text.scopes }}</dt><dd class="flex flex-wrap gap-2"><MoeBadge v-for="scope in application.latest.config.scopes" :key="scope" color="default">{{ scope }}</MoeBadge></dd></div></dl>
  <p class="mt-4 text-xs leading-6 text-muted">{{ text.pkceHint }}</p>
  <MoeAccordion v-if="application.approved && application.approved.version !== application.latest.version" class="mt-4" :items="[{ value: 'approved', title: text.approvedConfig }]"><template #approved><dl class="moe-detail-fields"><div><dt>{{ text.callbacksLabel }}</dt><dd v-for="uri in application.approved.config.redirect_uris" :key="uri" class="font-mono">{{ uri }}</dd></div><div><dt>{{ text.scopes }}</dt><dd>{{ application.approved.config.scopes?.join(' ') }}</dd></div></dl></template></MoeAccordion>
</MoeSectionPanel></template>

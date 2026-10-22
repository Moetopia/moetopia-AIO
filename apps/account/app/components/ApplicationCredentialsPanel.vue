<script setup lang="ts">
import type { DeveloperApplication } from '@moetopia/api-client'
import { MoeSectionPanel, MoeButton } from '@moetopia/ui'
defineProps<{ application: DeveloperApplication; busy: boolean }>()
defineEmits<{ rotate: [] }>()
const { text } = useManagementRequest()
const { context } = usePortalContext()
</script>
<template><MoeSectionPanel :title="text.credentials"><template v-if="!application.public" #actions><MoeButton size="sm" :disabled="busy || context?.user?.has_password === false" @click="$emit('rotate')">{{ text.rotate }}</MoeButton></template><p class="py-8 text-center text-sm leading-7 text-muted">{{ application.public ? text.publicSecretHint : text.existingSecretHint }}</p></MoeSectionPanel></template>

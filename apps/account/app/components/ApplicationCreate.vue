<script setup lang="ts">
import { MoeDialog, MoeInput, MoeTextarea, MoeButton } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const open = defineModel<boolean>('open', { default: false })
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
const name = ref('')
const description = ref('')
watch(open, active => { if (active) { name.value = ''; description.value = '' } })
</script>
<template><MoeDialog v-model:open="open" :title="text.createAppTitle" :description="text.createUnavailable" :close-label="text.cancel" :show-close-action="false"><MoeInput v-model="name" :label="text.appIdentity" required /><MoeTextarea v-model="description" :label="text.appDescription" :rows="3" /><template #footer><div class="flex justify-end gap-3"><MoeButton color="default" variant="flat" @click="open = false">{{ text.cancel }}</MoeButton><MoeButton disabled>{{ text.createAppTitle }}</MoeButton></div></template></MoeDialog></template>

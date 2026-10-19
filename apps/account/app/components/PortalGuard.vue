<script setup lang="ts">
import { MoeAlert, MoeButton, MoeSpinner } from '@moetopia/ui'
import { accountMessages } from '@moetopia/locales'
const props = defineProps<{ required: boolean }>()
const route = useRoute()
const { locale } = usePortalPreferences()
const text = computed(() => accountMessages[locale.value])
const { authenticated, loading, busy, navigating, feedback, startLogin, connect } = usePortalContext()
watch([() => props.required, () => route.path, loading, busy], () => {
  if (import.meta.client && props.required && !loading.value && !busy.value && !authenticated.value && !feedback.value) startLogin()
}, { immediate: true })
</script>
<template>
  <slot v-if="!required || authenticated" />
  <div v-else class="moe-access-state">
    <MoeAlert v-if="feedback" :title="text[feedback]" color="danger" announce><template #actions><MoeButton variant="bordered" :loading="loading" @click="connect">{{ text.retry }}</MoeButton></template></MoeAlert>
    <MoeSpinner v-else :label="text.checking" />
    <MoeButton v-if="!loading && !feedback" :loading="navigating" @click="startLogin()">{{ text.login }}</MoeButton>
  </div>
</template>

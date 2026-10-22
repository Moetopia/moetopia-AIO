<script setup lang="ts">
import { MoePasswordInput, MoePasswordFeedback } from '@moetopia/ui'
import { securityMessages, accountMessages } from '@moetopia/locales'
defineOptions({ inheritAttrs: false })
defineProps<{ label: string; description?: string; disabled?: boolean; size?: 'sm' | 'md' | 'lg' }>()
const model = defineModel<string>({ default: '' })
const { locale } = usePortalPreferences()
const text = computed(() => securityMessages[locale.value])
const account = computed(() => accountMessages[locale.value])
const field = ref<HTMLElement>()
const feedbackId = useId()
const labels = computed(() => ({ title: text.value.requirements, minimum: text.value.minimum, maximum: text.value.maximum, policyHint: text.value.policyHint, strength: text.value.strength, levels: [text.value.veryWeak, text.value.weak, text.value.fair, text.value.strong, text.value.veryStrong], suggestion: text.value.suggestion, checking: text.value.checking, unavailable: text.value.unavailable }))
const validate = () => {
  const invalid = model.value && (Array.from(model.value).length < 15 ? text.value.minimum : new TextEncoder().encode(model.value).length > 256 ? text.value.maximum : '')
  field.value?.querySelector('input')?.setCustomValidity(invalid || '')
}
watch([model, locale], () => nextTick(validate))
onMounted(validate)
</script>
<template><div ref="field" class="grid gap-3"><MoePasswordInput v-model="model" v-bind="$attrs" :label="label" :description="description" :disabled="disabled" :size="size" :show-label="account.showPassword" :hide-label="account.hidePassword" autocomplete="new-password" :aria-describedby="`${feedbackId}-requirements`" /><p :id="`${feedbackId}-requirements`" class="sr-only">{{ text.minimum }}。{{ text.maximum }}。{{ text.policyHint }}</p><MoePasswordFeedback :id="feedbackId" :value="model" :min-length="15" :max-bytes="256" :labels="labels" /></div></template>

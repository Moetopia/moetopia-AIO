<script setup lang="ts">
import type { MoeOption } from '../types'
import Checkbox from './Checkbox.vue'
defineProps<{ label: string; options: MoeOption[]; disabled?: boolean; description?: string }>()
const model = defineModel<string[]>({ default: () => [] })
const update = (value: string, checked: boolean | 'indeterminate') => { model.value = checked === true ? Array.from(new Set([...model.value, value])) : model.value.filter(item => item !== value) }
</script>
<template><fieldset class="moe-checkbox-group" :disabled="disabled"><legend class="moe-label mb-1">{{ label }}</legend><p v-if="description" class="moe-description mb-2">{{ description }}</p><Checkbox v-for="option in options" :key="option.value" :label="option.label" :description="option.description" :disabled="disabled || option.disabled" :model-value="model.includes(option.value)" @update:model-value="update(option.value, $event)" /></fieldset></template>

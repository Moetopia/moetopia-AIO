<script setup lang="ts">
withDefaults(defineProps<{ title?: string; description?: string; headingLevel?: 'h2' | 'h3' | 'h4' }>(), { headingLevel: 'h2' })
</script>
<template><section class="moe-card"><header v-if="title || $slots.header" class="border-b border-line px-5 py-4"><slot name="header"><component :is="headingLevel" class="text-base font-semibold">{{ title }}</component><p v-if="description" class="moe-description mt-1">{{ description }}</p></slot></header><div class="p-5"><slot /></div><footer v-if="$slots.footer" class="border-t border-line px-5 py-3"><slot name="footer" /></footer></section></template>

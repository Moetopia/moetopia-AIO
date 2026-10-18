<script setup lang="ts">
import { Primitive } from 'reka-ui'
import Icon from './Icon.vue'
withDefaults(defineProps<{ href?: string; external?: boolean; underline?: 'always' | 'hover' | 'none'; asChild?: boolean }>(), { underline: 'hover' })
</script>
<template><Primitive as="a" :as-child="asChild" :href="href" :target="external ? '_blank' : undefined" :rel="external ? 'noopener noreferrer' : undefined" class="moe-link moe-focus" :data-underline="underline"><slot /><Icon v-if="external && !asChild" name="external" class="size-3.5" /><span v-if="external && !asChild" class="sr-only">（在新标签页打开 / Opens in a new tab）</span></Primitive></template>

<script setup lang="ts">
import { DialogClose, DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle, DialogTrigger } from 'reka-ui'

defineProps<{ title: string; description: string; closeLabel: string }>()
const open = defineModel<boolean>('open', { default: false })
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogTrigger as-child><slot name="trigger" /></DialogTrigger>
    <DialogPortal>
      <DialogOverlay class="moe-dialog-overlay fixed inset-0 z-40 bg-overlay" />
      <DialogContent class="moe-popup moe-dialog fixed top-1/2 left-1/2 z-50 grid max-h-[calc(100svh-2rem)] w-[min(28rem,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 gap-4 overflow-y-auto p-6 focus:outline-none">
        <DialogTitle class="text-lg font-semibold">{{ title }}</DialogTitle>
        <DialogDescription class="text-sm leading-6 text-muted">{{ description }}</DialogDescription>
        <slot />
        <slot name="footer" :close="() => open = false" />
        <DialogClose class="min-h-11 justify-self-end rounded-lg px-3 text-sm font-medium text-primary focus-visible:outline-2 focus-visible:outline-focus">{{ closeLabel }}</DialogClose>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<script setup lang="ts">
import type { MoeIconName } from '../types'
import Card from './Card.vue'
import Icon from './Icon.vue'
defineProps<{ items: { id: string; label: string; value?: string; icon: MoeIconName }[] }>()
</script>
<template><div class="moe-stat-grid"><Card v-for="item in items" :key="item.id" padding="lg"><div class="flex items-center justify-between gap-3"><p class="text-sm text-muted">{{ item.label }}</p><Icon :name="item.icon" class="text-primary" /></div><p class="moe-stat-value">{{ item.value ?? '—' }}</p></Card></div></template>

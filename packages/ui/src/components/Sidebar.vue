<script setup lang="ts">
import { useId } from 'vue'
import type { MoeNavGroup, MoeNavItem } from '../types'
import NavItem from './NavItem.vue'
import Button from './Button.vue'
import Icon from './Icon.vue'
withDefaults(defineProps<{ label: string; groups: MoeNavGroup[]; active?: string; collapsible?: boolean; collapseLabel?: string; expandLabel?: string }>(), { collapseLabel: '收起侧栏', expandLabel: '展开侧栏' })
const collapsed = defineModel<boolean>('collapsed', { default: false })
defineEmits<{ navigate: [event: MouseEvent, item: MoeNavItem] }>()
const id = useId()
</script>

<template>
  <aside class="moe-sidebar" :data-collapsed="collapsed">
    <div v-if="$slots.header || collapsible" class="moe-sidebar-heading"><div v-if="!collapsed" class="min-w-0 flex-1"><slot name="header" /></div><Button v-if="collapsible" icon-only variant="light" color="default" :aria-label="collapsed ? expandLabel : collapseLabel" :aria-expanded="!collapsed" :aria-controls="id" @click="collapsed = !collapsed"><Icon :name="collapsed ? 'chevron-right' : 'chevron-left'" /></Button></div>
    <nav :id="id" :aria-label="label" class="moe-sidebar-groups"><div v-for="group in groups" :key="group.label" class="grid gap-1"><p class="moe-sidebar-group" :class="{ 'sr-only': collapsed }">{{ group.label }}</p><NavItem v-for="item in group.items" :key="item.id" :item="item" :active="active === item.id" :compact="collapsed" @navigate="(event, entry) => $emit('navigate', event, entry)" /></div></nav>
    <div v-if="$slots.footer" class="moe-sidebar-footer"><slot name="footer" :collapsed="collapsed" /></div>
  </aside>
</template>

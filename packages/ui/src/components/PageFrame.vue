<script setup lang="ts">
import { useId } from 'vue'
withDefaults(defineProps<{ skipLabel: string; density?: 'default' | 'compact' }>(), { density: 'default' })
const mainId = useId()
</script>
<template>
  <div class="moe-page-frame" :data-density="density">
    <a :href="`#${mainId}`" class="moe-skip-link moe-focus">{{ skipLabel }}</a>
    <div class="moe-page-header"><div class="moe-page-header-inner"><slot name="header" /></div></div>
    <main :id="mainId" class="moe-page-main" tabindex="-1"><slot /></main>
    <footer class="moe-page-footer"><slot name="footer" /></footer>
  </div>
</template>

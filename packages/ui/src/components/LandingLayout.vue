<script setup lang="ts">
defineProps<{ title: string; description: string }>()
</script>
<template>
  <div class="moe-landing-layout">
    <section class="moe-landing-intro"><div class="moe-landing-inner"><h1>{{ title }}</h1><p>{{ description }}</p><div v-if="$slots.actions" class="moe-landing-actions"><slot name="actions" /></div></div></section>
    <div class="moe-landing-inner moe-landing-content"><slot /></div>
  </div>
</template>

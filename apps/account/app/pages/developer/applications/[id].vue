<script setup lang="ts">
const route = useRoute()
definePageMeta({ validate: route => typeof route.params.id === 'string' && /^[a-f0-9-]{36}$/i.test(route.params.id) })
</script>
<template><ApplicationDetail :key="String(route.params.id)" :id="String(route.params.id)" /></template>

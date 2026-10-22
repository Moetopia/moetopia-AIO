<script setup lang="ts">
const route = useRoute()
definePageMeta({ key: route => route.path })
</script>
<template><ApiOperation :id="String(route.params.operation)" /></template>

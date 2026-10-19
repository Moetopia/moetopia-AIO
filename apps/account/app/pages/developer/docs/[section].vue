<script setup lang="ts">
definePageMeta({ validate: route => typeof route.params.section === 'string' && ['authentication', 'credentials', 'endpoints', 'errors'].includes(route.params.section) })
const route = useRoute()
</script>
<template><DeveloperDocumentation :section="String(route.params.section)" /></template>

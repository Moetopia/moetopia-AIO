<script setup lang="ts">
import { guideIds } from '@moetopia/locales'
definePageMeta({ validate: route => typeof route.params.section === 'string' && [...guideIds, 'endpoints'].includes(route.params.section) })
const route = useRoute()
</script>
<template><DeveloperDocumentation :section="String(route.params.section)" /></template>

<script setup lang="ts">
const route = useRoute()
</script>
<template><PortalHome kind="developer" :workspace="route.path.startsWith('/developer/applications')"><template #account><AccountNavigation /></template><template #workspace-account><AccountNavigation sidebar /></template><slot /></PortalHome></template>

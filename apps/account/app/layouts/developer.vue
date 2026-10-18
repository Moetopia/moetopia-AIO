<script setup lang="ts">
const route = useRoute()
</script>
<template><PortalHome kind="developer" :workspace="route.path !== '/developer'"><slot /></PortalHome></template>

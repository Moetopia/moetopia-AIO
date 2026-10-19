<script setup lang="ts">
const route = useRoute()
onMounted(() => { if (route.query.authorization_request === undefined) void navigateTo('/profile', { replace: true }) })
</script>
<template><AuthorizationPanel v-if="route.query.authorization_request !== undefined" /><AccountPanel v-else section="profile" /></template>

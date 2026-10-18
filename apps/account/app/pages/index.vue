<script setup lang="ts">
const route = useRoute()
</script>
<template><AuthorizationPanel v-if="route.query.authorization_request !== undefined" /><AccountPanel v-else section="profile" /></template>

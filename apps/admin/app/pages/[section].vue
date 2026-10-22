<script setup lang="ts">
const route = useRoute()
const section = computed(() => String(route.params.section))
if (!['users', 'invitations', 'audit', 'principals', 'notifications', 'settings'].includes(section.value)) throw createError({ statusCode: 404 })
</script>
<template><AdminSettings v-if="section === 'settings'" /><AdminCollection v-else :key="section" :section="section as 'users' | 'invitations' | 'audit' | 'principals' | 'notifications'" /></template>

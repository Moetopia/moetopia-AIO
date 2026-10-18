<script setup lang="ts">
import { MoeButton, MoeIcon } from '@moetopia/ui'
const { locale, theme, text, toggleLocale, toggleTheme } = usePortalPreferences()
const ready = ref(false)
onMounted(() => { ready.value = true })
</script>
<template>
  <div class="flex items-center gap-1">
    <MoeButton variant="light" color="default" icon-only :disabled="!ready" :aria-label="text.themeLabel" @click="toggleTheme"><MoeIcon :name="theme === 'light' ? 'moon' : 'sun'" /></MoeButton>
    <MoeButton variant="light" color="default" size="sm" :disabled="!ready" :aria-label="text.languageLabel" @click="toggleLocale">{{ locale === 'en' ? '中文' : 'EN' }}</MoeButton>
  </div>
</template>

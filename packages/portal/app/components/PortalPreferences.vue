<script setup lang="ts">
import { MoeButton, MoeIcon, MoeLanguagePicker } from '@moetopia/ui'
import { supportedLocales } from '@moetopia/locales'
const { locale, theme, text, setLocale, toggleTheme } = usePortalPreferences()
const options = supportedLocales.map(locale => ({ value: locale.code, label: locale.label, shortLabel: locale.shortLabel }))
const ready = ref(false)
onMounted(() => { ready.value = true })
</script>
<template>
  <div class="flex items-center gap-1">
    <MoeButton variant="light" color="default" icon-only :disabled="!ready" :aria-label="text.themeLabel" @click="toggleTheme"><MoeIcon :name="theme === 'light' ? 'moon' : 'sun'" /></MoeButton>
    <MoeLanguagePicker :model-value="locale" :label="text.languageLabel" :options="options" :disabled="!ready" @update:model-value="setLocale" />
  </div>
</template>

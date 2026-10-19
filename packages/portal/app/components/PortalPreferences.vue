<script setup lang="ts">
import { MoeThemePicker, MoeLanguagePicker } from '@moetopia/ui'
import { supportedLocales } from '@moetopia/locales'
const { locale, themeChoice, text, setLocale, setTheme } = usePortalPreferences()
const options = supportedLocales.map(locale => ({ value: locale.code, label: locale.label, shortLabel: locale.shortLabel }))
const ready = ref(false)
onMounted(() => { ready.value = true })
</script>
<template>
  <div class="flex items-center gap-1">
    <MoeThemePicker :model-value="themeChoice" :label="text.themeLabel" :light-label="text.light" :dark-label="text.dark" :system-label="text.system" @update:model-value="setTheme" />
    <MoeLanguagePicker :model-value="locale" :label="text.languageLabel" :options="options" :disabled="!ready" @update:model-value="setLocale" />
  </div>
</template>

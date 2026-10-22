<script setup lang="ts">
import { MoePageHeading, MoeCodeBlock, MoeCopyButton, MoeDocumentLayout, MoeLink, MoeAlert } from '@moetopia/ui'
import { guideIds, type GuideId } from '@moetopia/locales'
import { documentationExamples, documentationSources } from '../utils/documentation-examples'
const props = defineProps<{ section?: string }>()
const { docs, common } = useDocumentation()
const id = computed(() => guideIds.includes(props.section as GuideId) ? props.section as GuideId : 'overview')
const guide = computed(() => docs.value.guides[id.value])
const position = computed(() => guideIds.indexOf(id.value))
const adjacent = computed(() => [guideIds[position.value - 1], guideIds[position.value + 1]].filter((value): value is GuideId => !!value))
const href = (value: string) => value === 'overview' ? '/developer/docs' : `/developer/docs/${value}`
const documentText = computed(() => [guide.value.title, guide.value.description, ...guide.value.sections.flatMap(section => [section.title, ...section.paragraphs, section.code ? documentationExamples[section.code] : ''])].filter(Boolean).join('\n\n'))
useHead(() => ({ title: `${guide.value.title} · Moetopia` }))
</script>
<template>
  <ApiDocumentation v-if="section === 'endpoints'" />
  <MoeDocumentLayout v-else :label="docs.labels.outline" :items="guide.sections.map(section => ({ id: section.id, label: section.title }))">
    <div class="flex flex-wrap items-center justify-between gap-3"><MoeLink as-child><NuxtLink to="/developer/docs/endpoints">{{ docs.labels.reference }}</NuxtLink></MoeLink><MoeCopyButton :value="documentText" :label="common.copy" :copied-label="common.copied" :error-label="common.copyFailed" /></div>
    <MoePageHeading :title="guide.title" :description="guide.description" />
    <MoeAlert v-if="id === 'authentication'" :title="docs.labels.sdk" />
    <section v-for="section in guide.sections" :id="section.id" :key="section.id">
      <h2>{{ section.title }}</h2><p v-for="paragraph in section.paragraphs" :key="paragraph">{{ paragraph }}</p>
      <template v-if="section.code"><MoeCodeBlock :title="docs.labels.requestExample" :code="documentationExamples[section.code] ?? ''" :copy-label="common.copy" :copied-label="common.copied" :error-label="common.copyFailed" /><p class="text-xs text-muted">{{ docs.labels.exampleNote }}</p></template>
      <div v-if="section.operations" class="moe-doc-links"><NuxtLink v-for="operation in section.operations" :key="operation" :to="`/developer/docs/api/${operation}`" class="moe-focus">{{ docs.labels.contract }} · {{ operation }}</NuxtLink></div>
    </section>
    <section id="resources"><h2>{{ docs.labels.related }}</h2><div class="moe-doc-links"><a v-for="source in documentationSources" :key="source.href" :href="source.href" class="moe-focus" target="_blank" rel="noopener noreferrer">{{ source.label }}</a></div></section>
    <nav class="moe-doc-links border-t border-line pt-5" :aria-label="docs.labels.guides"><NuxtLink v-for="value in adjacent" :key="value" :to="href(value)" class="moe-focus">{{ guideIds.indexOf(value) < position ? docs.labels.previous : docs.labels.next }} · {{ docs.guides[value].title }}</NuxtLink></nav>
  </MoeDocumentLayout>
</template>

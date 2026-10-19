<script setup lang="ts">
import { MoePageHeading, MoeCodeBlock, MoeTable, MoeCopyButton, MoeBadge } from '@moetopia/ui'
import { portalMessages } from '@moetopia/locales'
const props = defineProps<{ section?: string }>()
const { locale } = usePortalPreferences()
const text = computed(() => portalMessages[locale.value])
const title = computed(() => props.section === 'authentication' ? text.value.docsAuthentication : props.section === 'credentials' ? text.value.docsCredentials : props.section === 'endpoints' ? text.value.endpointTitle : props.section === 'errors' ? text.value.docsErrors : text.value.docsTitle)
const code = 'GET /.well-known/openid-configuration\n\nresponse_type=code\nscope=openid profile email\ncode_challenge_method=S256'
const endpoints = [
  { id: 'discovery', method: 'GET', path: '/.well-known/openid-configuration', purpose: 'Discovery' },
  { id: 'authorize', method: 'GET', path: '/oauth/authorize', purpose: 'Authorization' },
  { id: 'token', method: 'POST', path: '/oauth/token', purpose: 'Token' },
  { id: 'userinfo', method: 'GET', path: '/oauth/userinfo', purpose: 'UserInfo' },
  { id: 'jwks', method: 'GET', path: '/oauth/jwks', purpose: 'JWKS' },
  { id: 'revoke', method: 'POST', path: '/oauth/revoke', purpose: 'Revocation' },
]
const documentText = computed(() => [text.value.docsTitle, text.value.docsAuthenticationBody, text.value.docsCredentialsBody, text.value.docsErrorsBody, ...endpoints.map(endpoint => `${endpoint.method} ${endpoint.path}`)].join('\n\n'))
</script>
<template>
  <article class="moe-document">
    <div class="flex justify-end"><MoeCopyButton :value="documentText" :label="text.copy" :copied-label="text.copied" :error-label="text.copyFailed" /></div>
    <MoePageHeading :title="title" :description="text.docsDescription" />
    <section v-if="!section" id="overview"><p>{{ text.oauthSupportDescription }}</p><MoeBadge class="justify-self-start">{{ text.protocolBadge }}</MoeBadge><DeveloperGuide /></section>
    <section v-if="!section || section === 'authentication'" id="authentication"><h2 v-if="!section">{{ text.docsAuthentication }}</h2><p>{{ text.docsAuthenticationBody }}</p><MoeCodeBlock :title="text.docsExample" :code="code" :copy-label="text.copy" :copied-label="text.copied" :error-label="text.copyFailed" /></section>
    <section v-if="!section || section === 'credentials'" id="credentials"><h2 v-if="!section">{{ text.docsCredentials }}</h2><p>{{ text.docsCredentialsBody }}</p></section>
    <section v-if="!section || section === 'endpoints'" id="endpoints"><h2 v-if="!section">{{ text.endpointTitle }}</h2><p>{{ text.endpointDescription }}</p><MoeTable :caption="text.endpointTitle" :columns="[{ key: 'method', label: 'HTTP' }, { key: 'path', label: 'Endpoint' }, { key: 'purpose', label: 'Protocol' }]" :rows="endpoints" row-key="id" /></section>
    <section v-if="!section || section === 'errors'" id="errors"><h2 v-if="!section">{{ text.docsErrors }}</h2><p>{{ text.docsErrorsBody }}</p></section>
  </article>
</template>

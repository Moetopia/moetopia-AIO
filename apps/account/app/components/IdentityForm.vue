<script setup lang="ts">
import { MoeButton, MoeInput, MoePasswordInput, MoePageHeading, MoeAlert, MoeLink } from '@moetopia/ui'
import { accountMessages } from '@moetopia/locales'
defineProps<{ embedded?: boolean }>()
const route = useRoute()
const { locale } = usePortalPreferences()
const text = computed(() => accountMessages[locale.value])
const { text: integration } = useIntegrationMessages()
const { account, view, email, password, displayName, invitation, code, challenge, busy, loading, feedback, success, feedbackElement, connect, login, sendCode, register, changeView } = useAccountContext()
const alternative = computed(() => ({ path: view.value === 'login' ? '/auth/register' : '/auth/login', query: route.query.continue === 'developer-applications' ? { continue: 'developer-applications' } : {} }))
</script>
<template>
  <div class="grid gap-6">
    <MoePageHeading v-if="!embedded" :title="view === 'login' ? text.welcome : text.register" :description="view === 'login' ? text.subtitle : text.registerSubtitle" />
    <form class="grid gap-4" @submit.prevent="view === 'login' ? login() : register()">
      <p v-if="loading" class="text-xs text-muted">{{ text.checking }}</p>
      <MoeAlert v-if="view === 'register'" :title="account?.registration_mode === 'open' ? text.openHint : text.invitationHint" color="info" />
      <MoeInput id="account-email" v-model="email" :label="text.email" type="email" autocomplete="email" placeholder="you@example.com" :maxlength="254" size="lg" required :disabled="busy || loading" />
      <MoeInput v-if="view === 'register'" id="account-name" v-model="displayName" :label="text.name" :placeholder="text.namePlaceholder" autocomplete="nickname" :maxlength="80" size="lg" required :disabled="busy || loading" />
      <MoeInput v-if="view === 'register' && account?.registration_mode !== 'open'" id="account-invitation" v-model="invitation" :label="text.invitation" :placeholder="text.invitationPlaceholder" autocomplete="off" :maxlength="128" size="lg" required :disabled="busy || loading" />
      <div class="grid gap-2">
        <MoePasswordInput id="account-password" v-model="password" :label="text.password" :placeholder="text.passwordPlaceholder" :description="view === 'register' ? text.passwordHint : undefined" :autocomplete="view === 'login' ? 'current-password' : 'new-password'" :minlength="view === 'register' ? 15 : undefined" :maxlength="256" size="lg" required :disabled="busy || loading" :show-label="text.showPassword" :hide-label="text.hidePassword" />
        <MoeLink v-if="view === 'login'" as-child class="justify-self-end text-xs"><NuxtLink :to="{ path: '/auth/forgot-password', query: typeof route.query.authorization_request === 'string' ? { authorization_request: route.query.authorization_request } : {} }">{{ text.forgotPassword }}</NuxtLink></MoeLink>
      </div>
      <div v-if="view === 'register'" class="grid gap-1.5">
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-end gap-2">
          <MoeInput id="account-code" v-model="code" :label="text.code" :placeholder="text.codePlaceholder" autocomplete="one-time-code" inputmode="numeric" pattern="[0-9]{8}" :maxlength="8" size="lg" required :disabled="busy || !challenge" />
          <MoeButton variant="bordered" size="lg" :loading="busy" :disabled="!account || !email" @click="sendCode">{{ challenge ? text.resend : text.sendCode }}</MoeButton>
        </div>
        <p class="text-xs leading-5 text-muted">{{ text.codeHint }}</p>
      </div>
      <div v-if="feedback" ref="feedbackElement" tabindex="-1" class="outline-none"><MoeAlert :title="text[feedback]" :color="success ? 'success' : 'danger'" announce /></div>
      <MoeButton type="submit" size="lg" class="mt-2 w-full" :loading="busy" :disabled="!account || loading || view === 'register' && !challenge">{{ view === 'login' ? text.login : text.continue }}</MoeButton>
    </form>
    <FederationSignIn />
    <MoeLink v-if="view === 'login'" as-child class="justify-self-center text-xs"><NuxtLink to="/auth/cancel-deletion">{{ integration.cancelDeletion }}</NuxtLink></MoeLink>
    <p class="moe-auth-alternative"><span>{{ view === 'login' ? text.noAccount : text.haveAccount }}</span><MoeButton v-if="route.query.authorization_request !== undefined" variant="light" size="sm" :disabled="busy || loading" @click="changeView(view === 'login' ? 'register' : 'login')">{{ view === 'login' ? text.registerLink : text.loginLink }}</MoeButton><MoeLink v-else as-child><NuxtLink :to="alternative">{{ view === 'login' ? text.registerLink : text.loginLink }}</NuxtLink></MoeLink></p>
    <MoeButton v-if="!account && !loading || feedback === 'forbidden'" variant="bordered" :loading="loading" @click="connect">{{ text.retry }}</MoeButton>
  </div>
</template>

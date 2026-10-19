<script setup lang="ts">
import { MoeButton, MoeIcon, MoeInput, MoePasswordInput, MoePageHeading, MoeCard, MoeDescriptionList, MoeSettingRow, MoeBadge, MoeAlert, MoeEmptyState, MoeLink, MoeProfileLayout, MoeProfileSummary } from '@moetopia/ui'
import { accountMessages, portalMessages } from '@moetopia/locales'

const props = withDefaults(defineProps<{ section?: 'profile' | 'sessions' | 'security'; forceLogin?: boolean; embedded?: boolean }>(), { section: 'profile', forceLogin: false, embedded: false })
const { locale } = usePortalPreferences()
const route = useRoute()
const text = computed(() => accountMessages[locale.value])
const pageText = computed(() => portalMessages[locale.value])
const { account, sessionList, sessions, view, email, password, displayName, invitation, code, challenge, busy, loading, feedback, success, feedbackElement, formatTime, connect, perform, login, sendCode, register, revoke, changeView, loadSessions } = useAccountContext()
const title = computed(() => props.section === 'sessions' ? text.value.sessions : props.section === 'security' ? pageText.value.security : pageText.value.profile)
const description = computed(() => props.section === 'sessions' ? pageText.value.sessionCardDescription : props.section === 'security' ? pageText.value.securityOverview : pageText.value.profileDescription)
const alternative = computed(() => ({ path: view.value === 'login' ? '/auth/register' : '/auth/login', query: route.query.continue === 'developer-applications' ? { continue: 'developer-applications' } : {} }))
</script>

<template>
  <div class="grid gap-6">
    <template v-if="!account?.user || forceLogin">
      <MoePageHeading v-if="!embedded" :title="view === 'login' ? text.welcome : text.register" :description="view === 'login' ? text.subtitle : text.registerSubtitle" />
      <form class="grid gap-4" @submit.prevent="view === 'login' ? login() : register()">
        <p v-if="loading" class="text-xs text-muted">{{ text.checking }}</p>
        <MoeAlert v-if="view === 'register'" :title="account?.registration_mode === 'open' ? text.openHint : text.invitationHint" color="info" />
        <MoeInput id="account-email" v-model="email" :label="text.email" type="email" autocomplete="email" placeholder="you@example.com" :maxlength="254" required :disabled="busy || loading" />
        <div v-if="view === 'register'" class="grid gap-4" :class="account?.registration_mode !== 'open' ? 'sm:grid-cols-2' : ''">
          <MoeInput id="account-name" v-model="displayName" :label="text.name" :placeholder="text.namePlaceholder" autocomplete="nickname" :maxlength="80" required :disabled="busy || loading" />
          <MoeInput v-if="account?.registration_mode !== 'open'" id="account-invitation" v-model="invitation" :label="text.invitation" :placeholder="text.invitationPlaceholder" autocomplete="off" :maxlength="128" required :disabled="busy || loading" />
        </div>
        <MoePasswordInput id="account-password" v-model="password" :label="text.password" :placeholder="text.passwordPlaceholder" :description="view === 'register' ? text.passwordHint : undefined" :autocomplete="view === 'login' ? 'current-password' : 'new-password'" :minlength="view === 'register' ? 15 : undefined" :maxlength="256" required :disabled="busy || loading" :show-label="text.showPassword" :hide-label="text.hidePassword" />
        <div v-if="view === 'register'" class="grid gap-1.5">
          <div class="grid grid-cols-[minmax(0,1fr)_auto] items-end gap-2">
            <MoeInput id="account-code" v-model="code" :label="text.code" :placeholder="text.codePlaceholder" autocomplete="one-time-code" inputmode="numeric" pattern="[0-9]{8}" :maxlength="8" required :disabled="busy || !challenge" />
            <MoeButton variant="bordered" :loading="busy" :disabled="!account || !email" @click="sendCode">{{ challenge ? text.resend : text.sendCode }}</MoeButton>
          </div>
          <p class="text-xs leading-5 text-muted">{{ text.codeHint }}</p>
        </div>
        <MoeButton type="submit" class="mt-2 w-full" :loading="busy" :disabled="!account || loading || view === 'register' && !challenge">{{ view === 'login' ? text.login : text.continue }}<MoeIcon name="arrow-right" /></MoeButton>
      </form>
      <p class="border-t border-line pt-4 text-center text-xs leading-6 text-muted">{{ text.providerHint }}</p>
      <p class="moe-auth-alternative"><span>{{ view === 'login' ? text.noAccount : text.haveAccount }}</span><MoeButton v-if="route.query.authorization_request !== undefined" variant="light" size="sm" :disabled="busy || loading" @click="changeView(view === 'login' ? 'register' : 'login')">{{ view === 'login' ? text.registerLink : text.loginLink }}</MoeButton><MoeLink v-else as-child><NuxtLink :to="alternative">{{ view === 'login' ? text.registerLink : text.loginLink }}</NuxtLink></MoeLink></p>
    </template>
    <template v-else>
      <MoePageHeading :title="title" :description="description" />
      <MoeProfileLayout>
      <template v-if="section === 'profile'" #summary><MoeProfileSummary :name="account.user.display_name" :email="account.user.email" :description="pageText.identityDescription" /></template>
      <MoeCard v-if="section === 'profile'" layout="form" :title="pageText.profileTitle" :description="pageText.profileReadOnly"><div class="grid gap-4"><MoeInput :model-value="account.user.display_name" :label="text.name" readonly /><MoeButton class="w-full" disabled>{{ pageText.saveChanges }}</MoeButton></div></MoeCard>
      <MoeCard v-if="section === 'profile'" layout="form" :title="pageText.emailSettings" :description="pageText.emailSettingsDescription"><MoeInput :model-value="account.user.email" :label="text.email" type="email" readonly /></MoeCard>
      <MoeCard v-if="section === 'sessions'" id="sessions" :title="text.sessions" :description="pageText.sessionCardDescription">
        <template v-if="sessions.length"><MoeSettingRow v-for="session in sessions" :key="session.id" :title="session.id === sessionList?.current_id ? text.current : pageText.sessionTitle" icon="shield"><MoeDescriptionList :items="[{ key: 'created', label: text.created, value: formatTime(session.created_at) }, { key: 'expires', label: text.expires, value: formatTime(session.expires_at) }]" /><template #actions><MoeBadge v-if="session.id === sessionList?.current_id">{{ text.current }}</MoeBadge><MoeButton variant="light" color="danger" :disabled="busy || loading" @click="revoke(session.id)">{{ text.revoke }}</MoeButton></template></MoeSettingRow></template>
        <MoeEmptyState v-else :title="sessionList ? pageText.emptySessions : text.sessionListFailed" icon="shield"><MoeButton v-if="!sessionList" variant="bordered" :loading="busy" @click="perform(loadSessions)">{{ text.retry }}</MoeButton></MoeEmptyState>
      </MoeCard>
      <MoeCard v-if="section === 'profile' || section === 'security'" layout="form" :title="section === 'profile' ? text.sessions : pageText.accountAccess">
        <MoeSettingRow :title="pageText.viewSessions" :description="text.sessionHint" icon="shield"><template #actions><MoeLink as-child><NuxtLink to="/sessions">{{ pageText.viewSessions }}<MoeIcon name="arrow-right" class="size-4" /></NuxtLink></MoeLink></template></MoeSettingRow>
        <template v-if="section === 'security'">
          <MoeSettingRow :title="pageText.passwordAccess" :description="pageText.passwordAccessDescription" icon="user"><template #actions><MoeBadge color="secondary">{{ pageText.passwordChangesLater }}</MoeBadge></template></MoeSettingRow>
          <MoeSettingRow :title="pageText.providerAccess" :description="pageText.providerAccessDescription" icon="code"><template #actions><MoeBadge color="secondary">{{ pageText.planned }}</MoeBadge></template></MoeSettingRow>
          <MoeSettingRow :title="pageText.accessManagement" :description="pageText.accessManagementDescription" icon="grid"><template #actions><MoeBadge color="secondary">{{ pageText.planned }}</MoeBadge></template></MoeSettingRow>
        </template>
      </MoeCard>
      <MoeCard v-if="section === 'profile'" layout="form" :title="pageText.passwordAccess"><MoeSettingRow :title="pageText.passwordChangesLater" :description="pageText.passwordAccessDescription" icon="shield"><template #actions><MoeBadge color="secondary">{{ pageText.planned }}</MoeBadge></template></MoeSettingRow><MoeLink as-child><NuxtLink to="/security">{{ pageText.security }}<MoeIcon name="arrow-right" class="size-4" /></NuxtLink></MoeLink></MoeCard>
      </MoeProfileLayout>
    </template>
    <div v-if="feedback" ref="feedbackElement" tabindex="-1" class="outline-none"><MoeAlert :title="text[feedback]" :color="success ? 'success' : 'danger'" announce /></div>
    <MoeButton v-if="!account && !loading || feedback === 'forbidden' || feedback === 'sessionExpired'" variant="bordered" :loading="loading" @click="connect">{{ text.retry }}</MoeButton>
  </div>
</template>


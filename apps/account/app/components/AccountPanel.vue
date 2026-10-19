<script setup lang="ts">
import { MoeButton, MoeIcon, MoePageHeading, MoeCard, MoeDescriptionList, MoeSettingRow, MoeBadge, MoeAlert, MoeEmptyState, MoeLink, MoeProfileLayout, MoeProfileSummary, MoeDialog, MoeSpinner } from '@moetopia/ui'
import { accountMessages, portalMessages } from '@moetopia/locales'
const props = withDefaults(defineProps<{ section?: 'profile' | 'sessions' | 'security' }>(), { section: 'profile' })
const { locale } = usePortalPreferences()
const text = computed(() => accountMessages[locale.value])
const pageText = computed(() => portalMessages[locale.value])
const { context, sessionList, sessions, sessionsLoading, authenticated, busy, feedback, formatTime, connect, loadSessions, revoke } = usePortalContext()
const selected = ref<string>()
const revoking = ref(false)
const title = computed(() => props.section === 'sessions' ? text.value.sessions : props.section === 'security' ? pageText.value.security : pageText.value.profile)
const description = computed(() => props.section === 'sessions' ? pageText.value.sessionCardDescription : props.section === 'security' ? pageText.value.securityOverview : pageText.value.profileDescription)
watch([() => props.section, authenticated], () => { if (props.section === 'sessions' && authenticated.value) void loadSessions() }, { immediate: true })
const confirmRevoke = async () => { if (!selected.value) return; await revoke(selected.value); if (!feedback.value) revoking.value = false }
</script>
<template>
  <div v-if="context?.user" class="grid gap-8">
    <MoePageHeading :title="title" :description="description" />
    <MoeProfileLayout>
      <template v-if="section === 'profile'" #summary><MoeProfileSummary :name="context.user.display_name" :email="context.user.email" :src="context.user.avatar_url" :description="context.user.bio || pageText.identityDescription" /></template>
      <ProfileSettings v-if="section === 'profile'" :user="context.user" />
      <MoeCard v-if="section === 'sessions'" :title="text.sessions" :description="pageText.sessionCardDescription" layout="form">
        <MoeSpinner v-if="sessionsLoading" :label="text.checking" /><template v-else-if="sessions.length"><MoeSettingRow v-for="session in sessions" :key="session.id" :title="session.id === sessionList?.current_id ? text.current : pageText.sessionTitle" icon="shield"><MoeDescriptionList :items="[{ key: 'created', label: text.created, value: formatTime(session.created_at) }, { key: 'expires', label: text.expires, value: formatTime(session.expires_at) }]" /><template #actions><MoeBadge v-if="session.id === sessionList?.current_id">{{ text.current }}</MoeBadge><MoeButton variant="light" color="danger" :disabled="busy" @click="selected = session.id; revoking = true">{{ text.revoke }}</MoeButton></template></MoeSettingRow></template>
        <MoeEmptyState v-else :title="sessionList ? pageText.emptySessions : text.sessionListFailed" icon="shield"><MoeButton v-if="!sessionList" variant="bordered" :disabled="busy" @click="loadSessions">{{ text.retry }}</MoeButton></MoeEmptyState>
      </MoeCard>
      <MoeCard v-if="section === 'profile' || section === 'security'" layout="form" :title="text.sessions">
        <MoeSettingRow :title="pageText.viewSessions" :description="text.sessionHint" icon="shield"><template #actions><MoeLink as-child><NuxtLink to="/sessions">{{ pageText.viewSessions }}<MoeIcon name="arrow-right" class="size-4" /></NuxtLink></MoeLink></template></MoeSettingRow>
      </MoeCard>
      <ProfileSettings v-if="section === 'security'" :user="context.user" security-only />
    </MoeProfileLayout>
    <MoeAlert v-if="feedback" :title="text[feedback]" color="danger" announce><template #actions><MoeButton variant="bordered" :disabled="busy" @click="connect">{{ text.retry }}</MoeButton></template></MoeAlert>
    <MoeDialog v-model:open="revoking" :title="text.revoke" :description="pageText.revokeConfirmation" :close-label="pageText.cancel" :show-close-action="false"><template #footer><div class="flex justify-end gap-3"><MoeButton variant="flat" color="default" :disabled="busy" @click="revoking = false">{{ pageText.cancel }}</MoeButton><MoeButton color="danger" :loading="busy" @click="confirmRevoke">{{ text.revoke }}</MoeButton></div></template></MoeDialog>
  </div>
</template>


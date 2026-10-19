<script setup lang="ts">
import type { ProfileResponse, ProfileUpdateRequest, AvatarUpdateRequest } from '@moetopia/api-client'
import { MoeCard, MoeInput, MoeTextarea, MoeAvatar, MoeButton, MoeAlert } from '@moetopia/ui'
import { accountMessages, portalMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const accountText = computed(() => accountMessages[locale.value])
const pageText = computed(() => portalMessages[locale.value])
const { context } = usePortalContext()
const { text, busy, error, success, headers, run } = useManagementRequest()
const name = ref('')
const bio = ref('')
const fileInput = ref<HTMLInputElement>()
const revision = ref(0)
watch(() => context.value?.user, user => { name.value = user?.display_name ?? ''; bio.value = user?.bio ?? ''; revision.value = user?.revision ?? 0 }, { immediate: true })
const update = (user: ProfileResponse) => { if (context.value) context.value = { ...context.value, user } }
const save = () => run(async () => {
  const body: ProfileUpdateRequest = { display_name: name.value, bio: bio.value, revision: revision.value }
  update(await $fetch<ProfileResponse>('/api/v1/portal/profile', { method: 'PATCH', headers: headers.value, body }))
})
const avatar = (image_base64: string) => run(async () => {
  const body: AvatarUpdateRequest = { image_base64, revision: revision.value }
  update(await $fetch<ProfileResponse>('/api/v1/portal/avatar', { method: 'PUT', headers: headers.value, body }))
})
const upload = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!['image/png', 'image/jpeg'].includes(file.type) || file.size > 512 * 1024) { error.value = text.value.avatarHint; return }
  try {
    const data = await new Promise<string>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(String(reader.result).split(',')[1] ?? ''); reader.onerror = reject; reader.readAsDataURL(file) })
    await avatar(data)
  } catch { error.value = text.value.invalid }
}
</script>
<template>
  <MoeCard layout="form" :title="pageText.profileTitle">
    <form class="grid gap-4" @submit.prevent="save">
      <div class="flex flex-wrap items-center gap-4"><MoeAvatar :name="name" :src="context?.user?.avatar_url" size="lg" /><input ref="fileInput" type="file" accept="image/png,image/jpeg" class="sr-only" :aria-label="pageText.changeAvatar" :disabled="busy" @change="upload" /><MoeButton type="button" variant="bordered" size="sm" :disabled="busy" @click="fileInput?.click()">{{ pageText.changeAvatar }}</MoeButton><MoeButton v-if="context?.user?.avatar_url" type="button" variant="light" color="danger" :disabled="busy" @click="avatar('')">{{ text.removeAvatar }}</MoeButton></div>
      <p class="text-sm text-muted">{{ text.avatarHint }}</p>
      <MoeInput v-model="name" :label="accountText.name" required :maxlength="80" :disabled="busy" />
      <MoeTextarea v-model="bio" :label="pageText.biography" :placeholder="pageText.biographyPlaceholder" :rows="3" :maxlength="500" :disabled="busy" />
      <MoeAlert v-if="error" :title="error" color="danger" announce /><MoeAlert v-else-if="success" :title="text.saved" color="success" announce />
      <MoeButton class="w-full" type="submit" :loading="busy">{{ pageText.saveChanges }}</MoeButton>
    </form>
  </MoeCard>
</template>

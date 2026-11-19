<script setup lang="ts">
import type { ProfileResponse, ProfileUpdateRequest, AvatarUpdateRequest } from '@moetopia/api-client'
import { MoeCard, MoeInput, MoeTextarea, MoeAvatar, MoeAvatarCrop, MoeButton, MoeAlert } from '@moetopia/ui'
import { accountMessages, portalMessages } from '@moetopia/locales'
const { locale } = usePortalPreferences()
const accountText = computed(() => accountMessages[locale.value])
const pageText = computed(() => portalMessages[locale.value])
const { context } = usePortalContext()
const { text, busy, error, success, headers, run } = useManagementRequest()
const name = ref('')
const bio = ref('')
const fileInput = ref<HTMLInputElement>()
const cropFile = shallowRef<File>()
const draggingFile = ref(false)
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
const acceptAvatar = (file: File | undefined) => {
  if (!file) return
  if (!['image/png', 'image/jpeg'].includes(file.type) || file.size > 20 * 1024 * 1024) { error.value = text.value.avatarHint; return }
  error.value = ''
  cropFile.value = file
}
const selectAvatar = (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  acceptAvatar(file)
}
const dropAvatar = (event: DragEvent) => {
  draggingFile.value = false
  if (busy.value) return
  acceptAvatar(event.dataTransfer?.files?.[0])
}
const uploadCrop = (data: string) => {
  cropFile.value = undefined
  return avatar(data)
}
</script>
<template>
  <MoeCard layout="form" :title="pageText.profileTitle">
    <form class="grid gap-4" @submit.prevent="save">
      <div
        class="flex flex-wrap items-center gap-4 rounded-xl border border-dashed p-3 transition-colors"
        :class="draggingFile ? 'border-primary bg-accent/40' : 'border-line bg-surface-raised/40'"
        @dragover.prevent="draggingFile = !busy"
        @dragleave="draggingFile = false"
        @drop.prevent="dropAvatar"
      >
        <MoeAvatar :name="name" :src="context?.user?.avatar_url" size="lg" />
        <input ref="fileInput" type="file" accept="image/png,image/jpeg" class="sr-only" :aria-label="pageText.changeAvatar" :disabled="busy" @change="selectAvatar" />
        <div class="min-w-0 flex-1 grid gap-2">
          <div class="flex flex-wrap items-center gap-2">
            <MoeButton type="button" variant="bordered" size="sm" :disabled="busy" @click="fileInput?.click()">{{ pageText.changeAvatar }}</MoeButton>
            <MoeButton v-if="context?.user?.avatar_url" type="button" variant="light" color="danger" size="sm" :disabled="busy" @click="avatar('')">{{ text.removeAvatar }}</MoeButton>
          </div>
          <p class="text-xs text-muted">{{ text.avatarHint }}</p>
        </div>
      </div>
      <MoeInput v-model="name" :label="accountText.name" required :maxlength="80" :disabled="busy" />
      <MoeTextarea v-model="bio" :label="pageText.biography" :placeholder="pageText.biographyPlaceholder" :rows="3" :maxlength="500" :disabled="busy" />
      <MoeAlert v-if="error" :title="error" color="danger" announce /><MoeAlert v-else-if="success" :title="text.saved" color="success" announce />
      <MoeButton class="w-full" type="submit" :loading="busy">{{ pageText.saveChanges }}</MoeButton>
    </form>
    <MoeAvatarCrop
      v-if="cropFile"
      :file="cropFile"
      :title="text.avatarCrop"
      :zoom-label="text.avatarZoom"
      :horizontal-label="text.avatarHorizontal"
      :vertical-label="text.avatarVertical"
      :drag-hint="text.avatarDragHint"
      :reset-label="text.avatarReset"
      :confirm-label="text.avatarUpload"
      :cancel-label="text.close"
      :error-label="text.invalid"
      :loading="busy"
      @cancel="cropFile = undefined"
      @confirm="uploadCrop"
    />
  </MoeCard>
</template>

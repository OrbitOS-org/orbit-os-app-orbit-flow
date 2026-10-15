<script setup lang="ts">
import AppMark from '@/components/AppMark.vue'
import { computed, ref } from 'vue'

import { useAuthStore } from '@/stores/authStore'
import { useApi, type ApiError } from '@/composables/useApi'

const auth = useAuthStore()
const api = useApi()

const username = ref('')
const password = ref('')
const passwordConfirm = ref('')
const submitting = ref(false)
const errorMessage = ref<string | null>(null)

const usernameTrimmed = computed(() => username.value.trim())
const passwordTooShort = computed(
  () => password.value.length > 0 && password.value.length < 8,
)
const passwordsDiffer = computed(
  () => passwordConfirm.value.length > 0 && passwordConfirm.value !== password.value,
)
const canSubmit = computed(
  () =>
    !submitting.value &&
    usernameTrimmed.value.length > 0 &&
    password.value.length >= 8 &&
    passwordConfirm.value === password.value,
)

async function handleSubmit() {
  if (!canSubmit.value) return
  submitting.value = true
  errorMessage.value = null
  try {
    await auth.setup(usernameTrimmed.value, password.value)
  } catch (err) {
    const apiErr = err as ApiError
    errorMessage.value = api.isApiError(err)
      ? apiErr.message
      : 'Setup failed; please try again.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="auth-screen">
    <form class="auth-card" @submit.prevent="handleSubmit">
      <div class="auth-icon"><AppMark class="w-9 h-9" /></div>
      <h1 class="auth-title">Welcome to Orbit Flow</h1>
      <p class="auth-sub">Create the administrator account to get started.</p>

      <div class="auth-fields">
        <input
          v-model="username"
          type="text"
          placeholder="Username"
          aria-label="Username"
          autocomplete="username"
          autofocus
          :disabled="submitting"
        />
        <input
          v-model="password"
          type="password"
          placeholder="Password"
          aria-label="Password"
          autocomplete="new-password"
          :disabled="submitting"
          :class="{ bad: passwordTooShort }"
        />
        <input
          v-model="passwordConfirm"
          type="password"
          placeholder="Confirm password"
          aria-label="Confirm password"
          autocomplete="new-password"
          :disabled="submitting"
          :class="{ bad: passwordsDiffer }"
        />
      </div>
      <p v-if="passwordTooShort" class="auth-hint bad">Password must be at least 8 characters.</p>
      <p v-else-if="passwordsDiffer" class="auth-hint bad">Passwords do not match.</p>
      <p v-else class="auth-hint">At least 8 characters.</p>

      <p v-if="errorMessage" class="settings-error mt-3" role="alert">{{ errorMessage }}</p>

      <button type="submit" class="auth-btn" :disabled="!canSubmit">
        {{ submitting ? 'Creating…' : 'Create account' }}
      </button>
      <p class="auth-foot">Orbit OS</p>
    </form>
  </div>
</template>

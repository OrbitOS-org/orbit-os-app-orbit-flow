<script setup lang="ts">
import AppMark from '@/components/AppMark.vue'
import { computed, ref } from 'vue'

import { useAuthStore } from '@/stores/authStore'
import { useApi, type ApiError } from '@/composables/useApi'

const auth = useAuthStore()
const api = useApi()

const username = ref('')
const password = ref('')
const submitting = ref(false)
const errorMessage = ref<string | null>(null)

const canSubmit = computed(
  () => !submitting.value && username.value.trim().length > 0 && password.value.length > 0,
)

async function handleSubmit() {
  if (!canSubmit.value) return
  submitting.value = true
  errorMessage.value = null
  try {
    await auth.login(username.value.trim(), password.value)
  } catch (err) {
    if (api.isApiError(err)) {
      const apiErr = err as ApiError
      // Map server status codes to user-facing messages. The server
      // intentionally returns the same "invalid credentials" text for
      // wrong password, missing user, and disabled user — so the
      // attacker cannot enumerate accounts.
      if (apiErr.status === 401) {
        errorMessage.value = 'Invalid username or password.'
      } else if (apiErr.status === 429) {
        errorMessage.value =
          'Too many failed attempts. Please wait a few minutes before trying again.'
      } else {
        errorMessage.value = apiErr.message || 'Login failed.'
      }
    } else {
      errorMessage.value = "Can't reach Orbit Flow. Check that the app is running on the device."
    }
    // Clear the password field on any error so a stuck-keys keystroke
    // does not slowly walk through wrong passwords.
    password.value = ''
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="auth-screen">
    <form class="auth-card" @submit.prevent="handleSubmit">
      <div class="auth-icon"><AppMark class="w-9 h-9" /></div>
      <h1 class="auth-title">Orbit Flow</h1>
      <p class="auth-sub">Sign in to edit and deploy flows on this device.</p>

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
          autocomplete="current-password"
          :disabled="submitting"
        />
      </div>

      <p v-if="errorMessage" class="settings-error mt-3" role="alert">{{ errorMessage }}</p>

      <button type="submit" class="auth-btn" :disabled="!canSubmit">
        {{ submitting ? 'Signing in…' : 'Sign in' }}
      </button>
    </form>
  </div>
</template>

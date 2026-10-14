<script setup lang="ts">
import { useSettingsAction } from '@/views/settings/action'
import { computed, onMounted, ref } from 'vue'

import { useAuthStore } from '@/stores/authStore'
import { useApi, type ApiError } from '@/composables/useApi'
import type { Role, User } from '@/types/auth'

const auth = useAuthStore()
const api = useApi()

// ── Table data ───────────────────────────────────────────────────────────
const users = ref<User[]>([])
const loading = ref(true)
const tableError = ref<string | null>(null)

async function reload() {
  loading.value = true
  tableError.value = null
  try {
    users.value = await api.listUsers()
  } catch (err) {
    tableError.value = api.isApiError(err)
      ? (err as ApiError).message
      : 'Failed to load users.'
  } finally {
    loading.value = false
  }
}

onMounted(reload)

// ── Per-row inline actions ───────────────────────────────────────────────
async function applyUpdate(u: User, patch: { role?: Role; disabled?: boolean }) {
  try {
    const updated = await api.updateUser(u.id, patch)
    const i = users.value.findIndex((x) => x.id === u.id)
    if (i !== -1) users.value[i] = updated
  } catch (err) {
    const apiErr = err as ApiError
    const msg = api.isApiError(err) ? apiErr.message : 'Update failed.'
    // Re-fetch on conflict so the UI reflects the actual server state.
    if (apiErr.status === 409) await reload()
    window.alert(msg)
  }
}

async function changeRole(u: User, newRole: Role) {
  if (newRole === u.role) return
  // Promoting to admin is a meaningful action; require an explicit
  // confirmation. Demoting / lateral changes go through silently.
  if (newRole === 'admin' && !window.confirm(
    `Promote "${u.username}" to admin? Admins can manage all users.`,
  )) {
    return
  }
  await applyUpdate(u, { role: newRole })
}

async function toggleDisabled(u: User) {
  await applyUpdate(u, { disabled: !u.disabled })
}

// ── Create-user modal ────────────────────────────────────────────────────
const showCreate = ref(false)
const createUsername = ref('')
const createPassword = ref('')
const createRole = ref<Role>('viewer')
const createSubmitting = ref(false)
const createError = ref<string | null>(null)

const canSubmitCreate = computed(
  () =>
    !createSubmitting.value &&
    createUsername.value.trim().length > 0 &&
    createPassword.value.length >= 8,
)

function openCreate() {
  createUsername.value = ''
  createPassword.value = ''
  createRole.value = 'viewer'
  createError.value = null
  showCreate.value = true
}

async function submitCreate() {
  if (!canSubmitCreate.value) return
  if (createRole.value === 'admin' && !window.confirm(
    'Create as admin? Admins can manage all users.',
  )) {
    return
  }
  createSubmitting.value = true
  createError.value = null
  try {
    const u = await api.createUser(
      createUsername.value.trim(),
      createPassword.value,
      createRole.value,
    )
    users.value.push(u)
    users.value.sort((a, b) => a.username.localeCompare(b.username))
    showCreate.value = false
  } catch (err) {
    createError.value = api.isApiError(err)
      ? (err as ApiError).message
      : 'Failed to create user.'
  } finally {
    createSubmitting.value = false
  }
}

// ── Set-password modal ───────────────────────────────────────────────────
const passwordTarget = ref<User | null>(null)
const passwordValue = ref('')
const passwordSubmitting = ref(false)
const passwordError = ref<string | null>(null)

const canSubmitPassword = computed(
  () => !passwordSubmitting.value && passwordValue.value.length >= 8,
)

function openPasswordReset(u: User) {
  passwordTarget.value = u
  passwordValue.value = ''
  passwordError.value = null
}

async function submitPassword() {
  if (!passwordTarget.value || !canSubmitPassword.value) return
  passwordSubmitting.value = true
  passwordError.value = null
  try {
    await api.setUserPassword(passwordTarget.value.id, passwordValue.value)
    passwordTarget.value = null
  } catch (err) {
    passwordError.value = api.isApiError(err)
      ? (err as ApiError).message
      : 'Failed to set password.'
  } finally {
    passwordSubmitting.value = false
  }
}

const ROLE_OPTIONS: Role[] = ['viewer', 'editor', 'admin']

function isSelf(u: User): boolean {
  return auth.user?.id === u.id
}

// The section's main action, shown at the top right of the Settings bar.
useSettingsAction({ label: 'New user', run: () => openCreate() })
</script>

<template>
  <div class="h-full flex flex-col bg-terminal-bg text-terminal-text overflow-hidden">

    <div class="flex-1 overflow-auto p-6 space-y-2">
      <div class="settings-column space-y-2">
      <div v-if="tableError" class="settings-error">{{ tableError }}</div>

      <div v-if="loading" class="px-3 text-[15px] text-terminal-text-dim">Loading…</div>

      <template v-else>
        <div class="info-title">Accounts</div>
        <div class="info-list">
          <div v-for="u in users" :key="u.id" class="info-row">
            <div class="min-w-0">
              <div class="text-terminal-text">
                {{ u.username }}
                <span v-if="isSelf(u)" class="text-terminal-text-dim">(you)</span>
              </div>
              <div class="text-[12.5px] text-terminal-text-dim">
                {{ u.authProvider }} · created {{ new Date(u.createdAt).toISOString().slice(0, 10) }}
              </div>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <select
                :value="u.role"
                class="pill-select"
                :aria-label="`Role of ${u.username}`"
                @change="changeRole(u, ($event.target as HTMLSelectElement).value as Role)"
              >
                <option v-for="r in ROLE_OPTIONS" :key="r" :value="r">{{ r[0].toUpperCase() + r.slice(1) }}</option>
              </select>
              <button
                class="pill"
                :class="u.disabled ? 'pill-bad' : 'pill-ok'"
                :title="u.disabled ? 'Disabled — click to enable' : 'Active — click to disable'"
                @click="toggleDisabled(u)"
              >
                {{ u.disabled ? 'Disabled' : 'Active' }}
              </button>
              <button class="pill pill-plain" @click="openPasswordReset(u)">Reset password</button>
            </div>
          </div>
          <div v-if="users.length === 0" class="info-row justify-center text-terminal-text-dim">No users yet.</div>
        </div>
        <p class="settings-note">
          Admins manage users and certificates, editors change and deploy flows, viewers only look.
          A disabled user cannot sign in; disabling a user or resetting their password ends their sessions.
        </p>
      </template>
      </div>
    </div>

    <!-- Create-user modal -->
    <div
      v-if="showCreate"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
    >
      <form
        class="ios-dialog w-[400px] max-w-[90vw]"
        @submit.prevent="submitCreate"
      >
        <header class="px-6 pt-5 pb-2">
          <h2>New user</h2>
        </header>
        <div class="px-6 py-5 space-y-4">
          <div>
            <label class="block text-[12px] text-terminal-text-dim mb-1">Username</label>
            <input
              v-model="createUsername"
              type="text"
              autofocus
              :disabled="createSubmitting"
              class="w-full bg-terminal-surface-alt/70 border border-transparent rounded-lg text-terminal-text px-3 py-2 text-[15px] outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25 disabled:opacity-50"
            />
          </div>
          <div>
            <label class="block text-[12px] text-terminal-text-dim mb-1">Password</label>
            <input
              v-model="createPassword"
              type="password"
              autocomplete="new-password"
              :disabled="createSubmitting"
              class="w-full bg-terminal-surface-alt/70 border border-transparent rounded-lg text-terminal-text px-3 py-2 text-[15px] outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25 disabled:opacity-50"
            />
            <p class="mt-1 text-[12.5px] text-terminal-text-dim">Minimum 8 characters.</p>
          </div>
          <div>
            <label class="block text-[12px] text-terminal-text-dim mb-1">Role</label>
            <select
              v-model="createRole"
              :disabled="createSubmitting"
              class="w-full bg-terminal-surface-alt/70 border border-transparent rounded-lg text-terminal-text px-3 py-2 text-[15px] outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25 disabled:opacity-50"
            >
              <option value="viewer">viewer — read-only</option>
              <option value="editor">editor — deploy & run</option>
              <option value="admin">admin — full control</option>
            </select>
          </div>
          <div v-if="createError" class="text-[12px] text-status-error bg-status-error/10 rounded-lg px-3 py-2 rounded-lg">
            {{ createError }}
          </div>
        </div>
        <footer class="px-6 py-4 border-t border-terminal-border flex justify-end gap-2">
          <button
            type="button"
            :disabled="createSubmitting"
            class="pill pill-plain disabled:opacity-50"
            @click="showCreate = false"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="!canSubmitCreate"
            class="pill pill-primary disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {{ createSubmitting ? 'Creating…' : 'Create' }}
          </button>
        </footer>
      </form>
    </div>

    <!-- Reset-password modal -->
    <div
      v-if="passwordTarget"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
    >
      <form
        class="ios-dialog w-[400px] max-w-[90vw]"
        @submit.prevent="submitPassword"
      >
        <header class="px-6 pt-5 pb-2">
          <h2>
            Reset password
          </h2>
          <p class="text-xs text-terminal-text-dim mt-0.5">
            For <span class="font-mono">{{ passwordTarget.username }}</span>.
            Active sessions will be terminated.
          </p>
        </header>
        <div class="px-6 py-5 space-y-4">
          <div>
            <label class="block text-[12px] text-terminal-text-dim mb-1">New password</label>
            <input
              v-model="passwordValue"
              type="password"
              autocomplete="new-password"
              autofocus
              :disabled="passwordSubmitting"
              class="w-full bg-terminal-surface-alt/70 border border-transparent rounded-lg text-terminal-text px-3 py-2 text-[15px] outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25 disabled:opacity-50"
            />
            <p class="mt-1 text-[12.5px] text-terminal-text-dim">Minimum 8 characters.</p>
          </div>
          <div v-if="passwordError" class="text-[12px] text-status-error bg-status-error/10 rounded-lg px-3 py-2 rounded-lg">
            {{ passwordError }}
          </div>
        </div>
        <footer class="px-6 py-4 border-t border-terminal-border flex justify-end gap-2">
          <button
            type="button"
            :disabled="passwordSubmitting"
            class="pill pill-plain disabled:opacity-50"
            @click="passwordTarget = null"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="!canSubmitPassword"
            class="pill pill-primary disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {{ passwordSubmitting ? 'Saving…' : 'Save' }}
          </button>
        </footer>
      </form>
    </div>
  </div>
</template>

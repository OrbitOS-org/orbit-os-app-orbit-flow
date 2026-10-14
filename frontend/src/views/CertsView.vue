<script setup lang="ts">
import { useSettingsAction } from '@/views/settings/action'
import { computed, onMounted, reactive, ref } from 'vue'

import { useApi, type ApiError } from '@/composables/useApi'
import { useAuthStore } from '@/stores/authStore'
import { useCertsStore } from '@/stores/certsStore'
import {
  CERT_SOURCE_OPTIONS,
  CERT_TYPE_OPTIONS,
  type CertEntryInput,
  type CertEntrySummary,
  type CertReference,
  type CertSource,
  type CertType,
} from '@/types/cert'

const auth = useAuthStore()
const api = useApi()
const store = useCertsStore()

const tableError = ref<string | null>(null)
onMounted(async () => {
  await store.reload()
  if (store.error) tableError.value = store.error
})

const canEdit = computed(() => auth.can('manageCerts'))

// ── Create / edit modal ─────────────────────────────────────────────────

interface FormState {
  open: boolean
  mode: 'create' | 'edit'
  editingId: string | null
  id: string
  name: string
  type: CertType
  source: CertSource
  certPem: string
  keyPem: string
  certPath: string
  keyPath: string
  notes: string
  submitting: boolean
  error: string | null
}

const form = reactive<FormState>({
  open: false,
  mode: 'create',
  editingId: null,
  id: '',
  name: '',
  type: 'ca-bundle',
  source: 'inline',
  certPem: '',
  keyPem: '',
  certPath: '',
  keyPath: '',
  notes: '',
  submitting: false,
  error: null,
})

const requiresKey = computed(
  () => form.type === 'client-pair' || form.type === 'server-pair',
)

const canSubmit = computed(() => {
  if (form.submitting) return false
  if (!form.id || !/^[a-z0-9][a-z0-9_-]{0,63}$/.test(form.id)) return false
  // In edit mode the backend inherits any field left blank from the
  // existing entry, so we only enforce material presence on create.
  // (Switching source is checked separately by the backend; we let the
  // server return a 400 if the user attempts that without supplying
  // fresh material.)
  if (form.mode === 'create') {
    if (form.source === 'inline') {
      if (!form.certPem.trim()) return false
      if (requiresKey.value && !form.keyPem.trim()) return false
    } else {
      if (!form.certPath.trim().startsWith('/')) return false
      if (requiresKey.value && !form.keyPath.trim().startsWith('/')) return false
    }
  }
  return true
})

function resetForm() {
  form.open = false
  form.editingId = null
  form.id = ''
  form.name = ''
  form.type = 'ca-bundle'
  form.source = 'inline'
  form.certPem = ''
  form.keyPem = ''
  form.certPath = ''
  form.keyPath = ''
  form.notes = ''
  form.submitting = false
  form.error = null
}

function openCreate() {
  resetForm()
  form.mode = 'create'
  form.open = true
}

function openEdit(entry: CertEntrySummary) {
  resetForm()
  form.mode = 'edit'
  form.editingId = entry.id
  form.id = entry.id
  form.name = entry.name
  form.type = entry.type
  form.source = entry.source
  form.certPath = entry.certPath ?? ''
  form.keyPath = entry.keyPath ?? ''
  form.notes = entry.notes ?? ''
  form.open = true
}

function entryFromForm(): CertEntryInput {
  const base: CertEntryInput = {
    id: form.id || undefined,
    name: form.name || undefined,
    type: form.type,
    source: form.source,
    notes: form.notes || undefined,
  }
  if (form.source === 'inline') {
    base.certPem = form.certPem
    if (requiresKey.value) base.keyPem = form.keyPem
  } else {
    base.certPath = form.certPath
    if (requiresKey.value) base.keyPath = form.keyPath
  }
  return base
}

async function submitForm() {
  if (!canSubmit.value) return
  form.submitting = true
  form.error = null
  try {
    if (form.mode === 'create') {
      await store.create(entryFromForm())
    } else if (form.editingId) {
      await store.update(form.editingId, entryFromForm())
    }
    resetForm()
  } catch (err) {
    form.error = api.isApiError(err)
      ? (err as ApiError).message
      : 'Failed to save certificate.'
  } finally {
    form.submitting = false
  }
}

// ── Validate-without-save preview ───────────────────────────────────────

const probe = ref<CertEntrySummary | null>(null)
const probeError = ref<string | null>(null)
const probing = ref(false)

async function previewParse() {
  probe.value = null
  probeError.value = null
  probing.value = true
  try {
    probe.value = await api.validateCert(entryFromForm())
  } catch (err) {
    probeError.value = api.isApiError(err)
      ? (err as ApiError).message
      : 'Validation failed.'
  } finally {
    probing.value = false
  }
}

// ── Delete with reference list ──────────────────────────────────────────

const deleteTarget = ref<CertEntrySummary | null>(null)
const deleteRefs = ref<CertReference[]>([])
const deleteError = ref<string | null>(null)
const deleting = ref(false)

async function confirmDelete(entry: CertEntrySummary) {
  deleteTarget.value = entry
  deleteRefs.value = []
  deleteError.value = null
}

async function performDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  deleteError.value = null
  deleteRefs.value = []
  try {
    await store.remove(deleteTarget.value.id)
    deleteTarget.value = null
  } catch (err) {
    if (api.isApiError(err)) {
      const apiErr = err as ApiError
      if (apiErr.status === 409 && apiErr.details) {
        const details = apiErr.details as { references?: CertReference[] }
        deleteRefs.value = details.references ?? []
      }
      deleteError.value = apiErr.message
    } else {
      deleteError.value = 'Failed to delete certificate.'
    }
  } finally {
    deleting.value = false
  }
}

function formatDate(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toISOString().slice(0, 10)
}

function fingerprintShort(fp?: string): string {
  if (!fp) return '—'
  return `sha256:${fp.slice(0, 16)}…`
}

// The section's main action, shown at the top right of the Settings bar.
useSettingsAction(canEdit.value ? { label: 'New certificate', run: () => openCreate() } : null)
</script>

<template>
  <div class="h-full flex flex-col bg-terminal-bg text-terminal-text overflow-hidden">

    <div class="flex-1 overflow-auto p-6 space-y-2">
      <div v-if="tableError" class="settings-error">{{ tableError }}</div>

      <div v-if="store.loading" class="px-3 text-[15px] text-terminal-text-dim">Loading…</div>

      <div v-else-if="store.certs.length === 0" class="settings-empty">
        <svg viewBox="0 0 24 24" class="w-9 h-9 text-terminal-text-dim/60" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" aria-hidden="true">
          <path d="M12 3 5 6v5c0 4.5 3 8 7 10 4-2 7-5.5 7-10V6z" /><path d="m9 12 2 2 4-4" stroke-linecap="round" />
        </svg>
        <div class="text-[16px] font-semibold text-terminal-text-bright">No certificates yet</div>
        <p class="text-[13.5px] text-terminal-text-dim max-w-md">
          Add the TLS certificates and keys that MQTT, HTTP, TCP and OPC UA connections use.
          Keys pasted here are stored encrypted on the device.
        </p>
        <button v-if="canEdit" class="pill pill-primary" @click="openCreate">New certificate</button>
      </div>

      <template v-else>
        <div class="info-title">Stored certificates</div>
        <div class="info-list">
          <div v-for="c in store.certs" :key="c.id" class="info-row">
            <div class="min-w-0">
              <div class="text-terminal-text truncate">{{ c.name || c.id }}</div>
              <div class="text-[12.5px] text-terminal-text-dim truncate">
                {{ c.type }} · {{ c.source }} · expires {{ formatDate(c.notAfter) }}
                <span v-if="c.subject"> · {{ c.subject }}</span>
              </div>
              <div class="text-[11.5px] font-mono text-terminal-text-dim/80 truncate">
                {{ c.id }} · {{ fingerprintShort(c.fingerprint) }}
              </div>
            </div>
            <div v-if="canEdit" class="flex items-center gap-2 shrink-0">
              <button class="pill pill-plain" @click="openEdit(c)">Edit</button>
              <button class="pill pill-bad" @click="confirmDelete(c)">Delete</button>
            </div>
          </div>
        </div>
      </template>
      <p class="settings-note">
        Connection nodes refer to a stored certificate by its ID. Pasted keys are encrypted on the device;
        a certificate taken from a file stays in that file.
      </p>
    </div>

    <!-- Create / edit modal ─────────────────────────────────────────────── -->
    <div
      v-if="form.open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      @click.self="resetForm"
    >
      <div class="ios-dialog w-[40rem] max-w-[92vw] max-h-[90vh] overflow-auto p-6">
        <h2 class="mb-1">
          {{ form.mode === 'create' ? 'New certificate' : `Edit ${form.editingId}` }}
        </h2>
        <p class="text-[12.5px] text-terminal-text-dim mb-4">
          IDs must match <code>^[a-z0-9][a-z0-9_-]{0,63}$</code> and are immutable.
        </p>

        <form class="flex flex-col gap-3" @submit.prevent="submitForm">
          <div class="grid grid-cols-2 gap-3">
            <label class="flex flex-col gap-1.5 text-[13px]">
              <span class="text-terminal-text-dim">ID</span>
              <input
                v-model="form.id"
                :disabled="form.mode === 'edit'"
                class="bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 font-mono text-[13px] text-terminal-text outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25 disabled:opacity-50"
                placeholder="ca-internal-root"
              />
            </label>
            <label class="flex flex-col gap-1.5 text-[13px]">
              <span class="text-terminal-text-dim">Name</span>
              <input
                v-model="form.name"
                class="bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25 text-terminal-text"
                placeholder="Internal Root CA"
              />
            </label>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <label class="flex flex-col gap-1.5 text-[13px]">
              <span class="text-terminal-text-dim">Type</span>
              <select
                v-model="form.type"
                class="bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25 text-terminal-text"
              >
                <option v-for="o in CERT_TYPE_OPTIONS" :key="o.value" :value="o.value">
                  {{ o.label }}
                </option>
              </select>
            </label>
            <label class="flex flex-col gap-1.5 text-[13px]">
              <span class="text-terminal-text-dim">Source</span>
              <select
                v-model="form.source"
                class="bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25 text-terminal-text"
              >
                <option v-for="o in CERT_SOURCE_OPTIONS" :key="o.value" :value="o.value">
                  {{ o.label }}
                </option>
              </select>
            </label>
          </div>

          <p
            v-if="form.mode === 'edit'"
            class="text-[12px] text-terminal-text-dim leading-tight bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25.5"
          >
            Existing PEM and paths are not echoed back for security. Leave the
            fields below empty to keep the current material; fill them only
            to replace it.
          </p>

          <template v-if="form.source === 'inline'">
            <label class="flex flex-col gap-1.5 text-[13px]">
              <span class="text-terminal-text-dim">Certificate (PEM)</span>
              <textarea
                v-model="form.certPem"
                rows="6"
                class="font-mono text-[12px] bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 resize-y outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25"
                :placeholder="form.mode === 'edit'
                  ? 'Leave empty to keep current certificate'
                  : '-----BEGIN CERTIFICATE-----\n…\n-----END CERTIFICATE-----'"
              />
            </label>
            <label v-if="requiresKey" class="flex flex-col gap-1.5 text-[13px]">
              <span class="text-terminal-text-dim">Private key (PEM)</span>
              <textarea
                v-model="form.keyPem"
                rows="6"
                class="font-mono text-[12px] bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 resize-y outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25"
                :placeholder="form.mode === 'edit'
                  ? 'Leave empty to keep current key'
                  : '-----BEGIN PRIVATE KEY-----\n…\n-----END PRIVATE KEY-----'"
              />
            </label>
          </template>

          <template v-else>
            <label class="flex flex-col gap-1.5 text-[13px]">
              <span class="text-terminal-text-dim">Certificate file (absolute path)</span>
              <input
                v-model="form.certPath"
                class="bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 font-mono text-[13px] text-terminal-text outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25"
                placeholder="/path/to/ca.pem"
              />
            </label>
            <label v-if="requiresKey" class="flex flex-col gap-1.5 text-[13px]">
              <span class="text-terminal-text-dim">Private key file (absolute path)</span>
              <input
                v-model="form.keyPath"
                class="bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 font-mono text-[13px] text-terminal-text outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25"
                placeholder="/path/to/client.key"
              />
            </label>
            <p class="text-[12px] text-terminal-text-dim leading-tight">
              File contents are read fresh on every connection init, so external
              tools (cert-manager, Let's Encrypt) can rotate without touching
              this entry.
            </p>
          </template>

          <label class="flex flex-col gap-1.5 text-[13px]">
            <span class="text-terminal-text-dim">Notes</span>
            <input
              v-model="form.notes"
              class="bg-terminal-surface-alt/70 border border-transparent rounded-lg px-2.5 py-1.5 outline-none focus:border-accent/60 focus:ring-2 focus:ring-accent/25 text-terminal-text"
              placeholder="Provisioned by ops on 2026-05-09"
            />
          </label>

          <div v-if="probe" class="text-[12.5px] border border-status-success/30 bg-status-success/10 text-terminal-text rounded px-3 py-2 font-mono">
            <div>Subject: {{ probe.subject }}</div>
            <div>Issuer: {{ probe.issuer }}</div>
            <div>Expires: {{ formatDate(probe.notAfter) }}</div>
            <div class="truncate">{{ fingerprintShort(probe.fingerprint) }}</div>
          </div>
          <div v-if="probeError" class="text-[12.5px] text-status-error">{{ probeError }}</div>
          <div v-if="form.error" class="text-[12.5px] text-status-error">{{ form.error }}</div>

          <div class="flex justify-end gap-2 mt-1">
            <button
              type="button"
              class="pill pill-plain"
              @click="resetForm"
            >
              Cancel
            </button>
            <button
              type="button"
              class="pill pill-plain"
              :disabled="probing"
              @click="previewParse"
            >
              {{ probing ? 'Validating…' : 'Validate' }}
            </button>
            <button
              type="submit"
              class="pill pill-primary disabled:opacity-50"
              :disabled="!canSubmit"
            >
              {{ form.mode === 'create' ? 'Create' : 'Save' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete confirmation ───────────────────────────────────────────── -->
    <div
      v-if="deleteTarget"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      @click.self="deleteTarget = null"
    >
      <div class="ios-dialog w-[34rem] max-w-[92vw] p-6">
        <h2 class="mb-2">
          Delete {{ deleteTarget.id }}?
        </h2>
        <p class="text-xs text-terminal-text-dim mb-3">
          The cert is removed from the store immediately. File-source paths
          on disk are NOT touched.
        </p>

        <div
          v-if="deleteRefs.length"
          class="text-[12.5px] border border-status-error/40 bg-status-error/10 text-terminal-text rounded px-3 py-2 mb-3"
        >
          <div class="text-status-error font-medium mb-1">
            Cert is still referenced:
          </div>
          <ul class="font-mono">
            <li v-for="(ref, i) in deleteRefs" :key="i" class="truncate">
              {{ ref.flowId ? `${ref.flowId} / ` : '' }}{{ ref.nodeType }} {{ ref.nodeId }} — {{ ref.field }}
            </li>
          </ul>
        </div>

        <div v-if="deleteError" class="text-[12.5px] text-status-error mb-3">
          {{ deleteError }}
        </div>

        <div class="flex justify-end gap-2">
          <button
            class="pill pill-plain"
            @click="deleteTarget = null"
          >
            Cancel
          </button>
          <button
            class="pill pill-bad disabled:opacity-50"
            :disabled="deleting"
            @click="performDelete"
          >
            {{ deleting ? 'Deleting…' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

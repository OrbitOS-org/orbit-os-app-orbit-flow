<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useFlowStore } from '@/stores/flowStore'
import { useUiStore } from '@/stores/uiStore'
import { useApi } from '@/composables/useApi'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import FormCheckbox from '@/components/ui/FormCheckbox.vue'

const props = defineProps<{
  configId?: string
}>()

const flowStore = useFlowStore()
const ui = useUiStore()
const api = useApi()

const name = ref('')
const bus = ref<number | null>(null)
const clockHz = ref(100_000)
const tenBitAddr = ref(false)
const clockStretching = ref(false)

const isEditing = ref(false)

// Buses OrbitOS's I2CManager reports for this device. Same "never show an
// empty dropdown" rule as the other hardware pickers: defaults to manual
// entry, only switches to a dropdown once there's something real to list.
const detectedBuses = ref<number[]>([])
const manualEntry = ref(true)
const CUSTOM_BUS_VALUE = '__custom__'

const busOptions = computed(() => [
  ...detectedBuses.value.map((b) => ({ value: String(b), label: `Bus ${b}` })),
  { value: CUSTOM_BUS_VALUE, label: 'Custom…' },
])

const selectedBus = computed<string>({
  get: () => (manualEntry.value || bus.value === null ? CUSTOM_BUS_VALUE : String(bus.value)),
  set: (value) => {
    if (value === CUSTOM_BUS_VALUE) {
      manualEntry.value = true
      return
    }
    manualEntry.value = false
    bus.value = Number(value)
  },
})

// Scan is independent of saving the config — it opens a transient bus with
// whatever settings are currently in the form and reports what answered.
const scanning = ref(false)
const scanError = ref('')
const scanResults = ref<number[] | null>(null)

async function scan() {
  if (bus.value === null) {
    scanError.value = 'Set a bus number first'
    return
  }
  scanning.value = true
  scanError.value = ''
  scanResults.value = null
  try {
    scanResults.value = await api.scanI2CBus({
      bus: bus.value,
      clockHz: clockHz.value,
      tenBitAddr: tenBitAddr.value,
      clockStretching: clockStretching.value,
    })
  } catch (err) {
    scanError.value = err instanceof Error ? err.message : 'Scan failed'
  } finally {
    scanning.value = false
  }
}

onMounted(async () => {
  if (props.configId) {
    const existing = flowStore.configs.find((c) => c.id === props.configId)
    if (existing) {
      isEditing.value = true
      name.value = existing.name ?? ''
      const cfg = (existing.config ?? {}) as Record<string, unknown>
      bus.value = (cfg.bus as number) ?? null
      clockHz.value = (cfg.clockHz as number) ?? 100_000
      tenBitAddr.value = (cfg.tenBitAddr as boolean) ?? false
      clockStretching.value = (cfg.clockStretching as boolean) ?? false
    }
  }

  try {
    detectedBuses.value = await api.listI2CBuses()
  } catch {
    detectedBuses.value = []
  }
  const hasUsableBuses =
    detectedBuses.value.length > 0 &&
    (bus.value === null || detectedBuses.value.includes(bus.value))
  if (hasUsableBuses) {
    manualEntry.value = false
  }
})

function generateId(): string {
  return crypto.randomUUID()
}

function save() {
  const config = {
    bus: bus.value,
    clockHz: clockHz.value,
    tenBitAddr: tenBitAddr.value,
    clockStretching: clockStretching.value,
  }

  if (isEditing.value && props.configId) {
    flowStore.updateConfig(props.configId, { name: name.value, config })
  } else {
    flowStore.addConfig({
      id: generateId(),
      type: 'i2c-bus',
      name: name.value,
      config,
    })
  }

  cancel()
}

function cancel() {
  ui.clearConfigEditor()
}

const canSave = computed(() => !!name.value && bus.value !== null && bus.value >= 0)
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-3">
      <div class="flex items-center gap-2 mb-1">
        <span class="text-accent text-sm font-bold uppercase tracking-wider">
          {{ isEditing ? 'Edit' : 'New' }} I2C Bus
        </span>
      </div>

      <FormField label="Name" :error="!name ? 'Name required' : ''">
        <FormInput v-model="name" placeholder="e.g. Sensor Bus" :invalid="!name" />
      </FormField>

      <FormField label="Bus" :error="bus === null ? 'Bus required' : ''">
        <template v-if="detectedBuses.length > 0" #action>
          <button
            type="button"
            class="text-[9px] text-terminal-text-dim hover:text-accent underline underline-offset-2"
            @click="manualEntry = !manualEntry"
          >
            {{ manualEntry ? 'Choose from detected buses' : 'Type manually' }}
          </button>
        </template>

        <FormSelect
          v-if="!manualEntry"
          v-model="selectedBus"
          :options="busOptions"
          placeholder="Select a bus…"
          width="100%"
        />
        <NumberInput
          v-else
          :model-value="bus ?? 0"
          :min="0"
          @update:model-value="bus = $event"
        />
      </FormField>

      <FormField label="Clock Speed" hint="Standard 100 kHz, Fast mode 400 kHz">
        <NumberInput v-model="clockHz" :min="1" unit="Hz" />
      </FormField>

      <FormCheckbox v-model="tenBitAddr" label="10-bit addressing" />
      <FormCheckbox v-model="clockStretching" label="Clock stretching" />

      <FormField label="Scan">
        <div class="flex flex-col gap-2">
          <button
            type="button"
            class="px-3 py-1.5 text-[10px] uppercase tracking-wider font-bold
                   bg-accent/10 text-accent border border-accent/30
                   hover:bg-accent/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed self-start"
            :disabled="scanning || bus === null"
            @click="scan"
          >
            {{ scanning ? 'Scanning…' : 'Scan for devices' }}
          </button>
          <div v-if="scanError" class="text-[10px] text-status-error">{{ scanError }}</div>
          <div v-else-if="scanResults !== null" class="text-[10px] text-terminal-text-dim leading-relaxed">
            <template v-if="scanResults.length > 0">
              Found: <span class="font-mono text-accent">{{ scanResults.map(a => '0x' + a.toString(16).toUpperCase().padStart(2, '0')).join(', ') }}</span>
            </template>
            <template v-else>No devices responded.</template>
          </div>
        </div>
      </FormField>
    </div>

    <div class="shrink-0 flex items-center gap-2 px-4 py-2.5 border-t border-terminal-border bg-terminal-surface">
      <button
        class="px-3 py-1 text-[10px] uppercase tracking-wider font-bold
               bg-accent/10 text-accent border border-accent/30
               hover:bg-accent/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="!canSave"
        @click="save"
      >
        {{ isEditing ? 'Update' : 'Create' }}
      </button>
      <button
        class="px-3 py-1 text-[10px] uppercase tracking-wider
               text-terminal-text-dim border border-terminal-border
               hover:text-terminal-text hover:border-terminal-text-dim transition-colors"
        @click="cancel"
      >
        Cancel
      </button>
    </div>
  </div>
</template>

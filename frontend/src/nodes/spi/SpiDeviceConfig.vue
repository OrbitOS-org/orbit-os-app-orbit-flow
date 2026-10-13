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

const MODES = [
  { value: 0, label: 'Mode 0 (CPOL=0, CPHA=0)' },
  { value: 1, label: 'Mode 1 (CPOL=0, CPHA=1)' },
  { value: 2, label: 'Mode 2 (CPOL=1, CPHA=0)' },
  { value: 3, label: 'Mode 3 (CPOL=1, CPHA=1)' },
]

const name = ref('')
const bus = ref(0)
const chipSelect = ref(0)
const maxSpeedHz = ref(1_000_000)
const bitsPerWord = ref(8)
const mode = ref(0)
const lsbFirst = ref(false)

const isEditing = ref(false)

// Devices OrbitOS's SpiManager reports (e.g. "spidev0.0") — same "never
// show an empty dropdown" rule as the other hardware pickers. Selecting one
// fills bus/chipSelect together; manual entry is always available too.
const detectedDevices = ref<string[]>([])
const manualEntry = ref(true)
const CUSTOM_DEVICE_VALUE = '__custom__'

const deviceOptions = computed(() => [
  ...detectedDevices.value.map((d) => ({ value: d, label: d })),
  { value: CUSTOM_DEVICE_VALUE, label: 'Custom…' },
])

const selectedDevice = computed<string>({
  get: () => (manualEntry.value ? CUSTOM_DEVICE_VALUE : `spidev${bus.value}.${chipSelect.value}`),
  set: (value) => {
    if (value === CUSTOM_DEVICE_VALUE) {
      manualEntry.value = true
      return
    }
    manualEntry.value = false
    const m = /^spidev(\d+)\.(\d+)$/.exec(value)
    if (m) {
      bus.value = Number(m[1])
      chipSelect.value = Number(m[2])
    }
  },
})

onMounted(async () => {
  if (props.configId) {
    const existing = flowStore.configs.find((c) => c.id === props.configId)
    if (existing) {
      isEditing.value = true
      name.value = existing.name ?? ''
      const cfg = (existing.config ?? {}) as Record<string, unknown>
      bus.value = (cfg.bus as number) ?? 0
      chipSelect.value = (cfg.chipSelect as number) ?? 0
      maxSpeedHz.value = (cfg.maxSpeedHz as number) ?? 1_000_000
      bitsPerWord.value = (cfg.bitsPerWord as number) ?? 8
      mode.value = (cfg.mode as number) ?? 0
      lsbFirst.value = (cfg.lsbFirst as boolean) ?? false
    }
  }

  try {
    detectedDevices.value = await api.listSpiDevices()
  } catch {
    detectedDevices.value = []
  }
  const current = `spidev${bus.value}.${chipSelect.value}`
  const hasUsableDevices =
    detectedDevices.value.length > 0 &&
    (!props.configId || detectedDevices.value.includes(current))
  if (hasUsableDevices) {
    manualEntry.value = false
  }
})

function generateId(): string {
  return crypto.randomUUID()
}

function save() {
  const config = {
    bus: bus.value,
    chipSelect: chipSelect.value,
    maxSpeedHz: maxSpeedHz.value,
    bitsPerWord: bitsPerWord.value,
    mode: mode.value,
    lsbFirst: lsbFirst.value,
  }

  if (isEditing.value && props.configId) {
    flowStore.updateConfig(props.configId, { name: name.value, config })
  } else {
    flowStore.addConfig({
      id: generateId(),
      type: 'spi-device',
      name: name.value,
      config,
    })
  }

  cancel()
}

function cancel() {
  ui.clearConfigEditor()
}

const canSave = computed(() => !!name.value)
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-3">
      <div class="flex items-center gap-2 mb-1">
        <span class="text-accent text-sm font-bold uppercase tracking-wider">
          {{ isEditing ? 'Edit' : 'New' }} SPI Device
        </span>
      </div>

      <FormField label="Name" :error="!name ? 'Name required' : ''">
        <FormInput v-model="name" placeholder="e.g. Flash Chip" :invalid="!name" />
      </FormField>

      <FormField label="Device">
        <template v-if="detectedDevices.length > 0" #action>
          <button
            type="button"
            class="text-[9px] text-terminal-text-dim hover:text-accent underline underline-offset-2"
            @click="manualEntry = !manualEntry"
          >
            {{ manualEntry ? 'Choose from detected devices' : 'Type manually' }}
          </button>
        </template>

        <FormSelect
          v-if="!manualEntry"
          v-model="selectedDevice"
          :options="deviceOptions"
          placeholder="Select a device…"
          width="100%"
        />
        <div v-else class="flex items-stretch gap-2">
          <div class="flex-1">
            <FormField label="Bus">
              <NumberInput v-model="bus" :min="0" />
            </FormField>
          </div>
          <div class="flex-1">
            <FormField label="Chip Select">
              <NumberInput v-model="chipSelect" :min="0" />
            </FormField>
          </div>
        </div>
      </FormField>

      <FormField label="Max Speed" hint="Upper bound; the driver may clock slower">
        <NumberInput v-model="maxSpeedHz" :min="1" unit="Hz" />
      </FormField>

      <div class="flex items-stretch gap-2">
        <div class="flex-1">
          <FormField label="Bits per Word">
            <NumberInput v-model="bitsPerWord" :min="1" :max="32" />
          </FormField>
        </div>
        <div class="flex-1">
          <FormField label="Mode">
            <FormSelect v-model="mode" :options="MODES" width="100%" />
          </FormField>
        </div>
      </div>

      <FormCheckbox v-model="lsbFirst" label="LSB first" />
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

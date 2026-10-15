<script setup lang="ts">
import { newId } from '@/lib/id'
import { ref, computed, onMounted } from 'vue'
import { useFlowStore } from '@/stores/flowStore'
import { useUiStore } from '@/stores/uiStore'
import { useApi } from '@/composables/useApi'
import type { GpioPinInfo } from '@/composables/useApi'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'

const props = defineProps<{
  configId?: string
}>()

const flowStore = useFlowStore()
const ui = useUiStore()
const api = useApi()

const DIRECTIONS = [
  { value: 'input', label: 'Input' },
  { value: 'output', label: 'Output' },
]

const name = ref('')
const chip = ref(0)
const line = ref<number | null>(null)
const pinName = ref('') // display name reported by ListPins, if any
const direction = ref<string>('input')

const isEditing = ref(false)

// Pins OrbitOS's GpioManager reports for this device (empty when running
// outside OrbitOS, or when nothing was detected). Defaults to manual
// chip/line entry — same "never show an empty dropdown" rule as the Modbus
// Server's Serial Port field, and for the same reason: a slow or failed
// fetch must not look like a broken UI.
const detectedPins = ref<GpioPinInfo[]>([])
const manualEntry = ref(true)

const CUSTOM_PIN_VALUE = '__custom__'

function pinKey(p: GpioPinInfo): string {
  return `${p.chip}:${p.line}`
}

const pinOptions = computed(() => [
  ...detectedPins.value.map((p) => ({
    value: pinKey(p),
    label: p.name ? `${p.name} (chip ${p.chip}, line ${p.line})` : `chip ${p.chip}, line ${p.line}`,
  })),
  { value: CUSTOM_PIN_VALUE, label: 'Custom…' },
])

const selectedPin = computed<string>({
  get: () => (manualEntry.value || line.value === null ? CUSTOM_PIN_VALUE : pinKey({ chip: chip.value, line: line.value, name: pinName.value })),
  set: (value) => {
    if (value === CUSTOM_PIN_VALUE) {
      manualEntry.value = true
      return
    }
    manualEntry.value = false
    const found = detectedPins.value.find((p) => pinKey(p) === value)
    if (found) {
      chip.value = found.chip
      line.value = found.line
      pinName.value = found.name
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
      chip.value = (cfg.chip as number) ?? 0
      line.value = (cfg.line as number) ?? null
      pinName.value = (cfg.name as string) ?? ''
      direction.value = (cfg.direction as string) ?? 'input'
    }
  }

  try {
    detectedPins.value = await api.listGpioPins()
  } catch {
    detectedPins.value = []
  }
  const hasUsablePins =
    detectedPins.value.length > 0 &&
    (line.value === null || detectedPins.value.some((p) => p.chip === chip.value && p.line === line.value))
  if (hasUsablePins) {
    manualEntry.value = false
  }
})

function generateId(): string {
  return newId()
}

function save() {
  const config = {
    chip: chip.value,
    line: line.value,
    name: pinName.value,
    direction: direction.value,
  }

  if (isEditing.value && props.configId) {
    flowStore.updateConfig(props.configId, { name: name.value, config })
  } else {
    flowStore.addConfig({
      id: generateId(),
      type: 'gpio-pin',
      name: name.value,
      config,
    })
  }

  cancel()
}

function cancel() {
  ui.clearConfigEditor()
}

const canSave = computed(() => !!name.value && line.value !== null && line.value >= 0)
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-3">
      <div class="flex items-center gap-2 mb-1">
        <span class="text-accent text-sm font-bold uppercase tracking-wider">
          {{ isEditing ? 'Edit' : 'New' }} GPIO Pin
        </span>
      </div>

      <FormField label="Name" :error="!name ? 'Name required' : ''">
        <FormInput v-model="name" placeholder="e.g. Relay 1" :invalid="!name" />
      </FormField>

      <FormField label="Pin" :error="line === null ? 'Pin required' : ''">
        <template v-if="detectedPins.length > 0" #action>
          <button
            type="button"
            class="text-[9px] text-terminal-text-dim hover:text-accent underline underline-offset-2"
            @click="manualEntry = !manualEntry"
          >
            {{ manualEntry ? 'Choose from detected pins' : 'Type manually' }}
          </button>
        </template>

        <FormSelect
          v-if="!manualEntry"
          v-model="selectedPin"
          :options="pinOptions"
          placeholder="Select a pin…"
          width="100%"
        />
        <div v-else class="flex items-stretch gap-2">
          <div class="flex-1">
            <FormField label="Chip">
              <NumberInput v-model="chip" :min="0" />
            </FormField>
          </div>
          <div class="flex-1">
            <FormField label="Line">
              <NumberInput :model-value="line ?? 0" :min="0" @update:model-value="line = $event" />
            </FormField>
          </div>
        </div>
      </FormField>

      <FormField label="Direction" hint="Set once when this pin is deployed; shared by every Read/Write node using it">
        <ToggleGroup v-model="direction" :options="DIRECTIONS" />
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

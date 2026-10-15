<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useFlowStore } from '@/stores/flowStore'
import { useUiStore } from '@/stores/uiStore'
import { useApi } from '@/composables/useApi'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'

const props = defineProps<{
  configId?: string
}>()

const flowStore = useFlowStore()
const ui = useUiStore()
const api = useApi()

const BAUD_RATES = [1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200, 230400].map((b) => ({ value: b, label: String(b) }))
const DATA_BITS = [{ value: 7, label: '7' }, { value: 8, label: '8' }]
const PARITY = [
  { value: 'none', label: 'None' },
  { value: 'even', label: 'Even' },
  { value: 'odd', label: 'Odd' },
]
const STOP_BITS = [{ value: 1, label: '1' }, { value: 2, label: '2' }]

const name = ref('')
const port = ref('')
const baudRate = ref(9600)
const dataBits = ref(8)
const parity = ref('none')
const stopBits = ref(1)
const isEditing = ref(false)

// Ports the device reports. Starts as a text field, as the other hardware
// pickers do, and becomes a list once the device has answered with ports.
const devicePorts = ref<string[]>([])
const manualEntry = ref(true)
const CUSTOM = '__custom__'

const bare = (p: string) => p.trim().replace(/^\/dev\//, '')

// Other connections in the workspace that use each port: a port can only
// have one, so the list says who has it (the deploy is refused otherwise).
const usedBy = computed(() => {
  const map = new Map<string, string>()
  for (const c of flowStore.configs) {
    if (c.id === props.configId) continue
    const cfg = (c.config ?? {}) as Record<string, unknown>
    let p = ''
    if (c.type === 'serial-port') p = String(cfg.port ?? '')
    else if (c.type === 'modbus-server' && cfg.transport === 'rtu') p = String(cfg.serialPort ?? '')
    if (p) map.set(bare(p), c.name || (c.type === 'serial-port' ? 'another Serial Port' : 'Modbus Server'))
  }
  return map
})

const portOptions = computed(() => [
  ...devicePorts.value.map((p) => ({
    value: p,
    label: usedBy.value.has(p) ? `${p} — used by ${usedBy.value.get(p)}` : p,
  })),
  { value: CUSTOM, label: 'Other…' },
])

const selectedPort = computed<string>({
  get: () => (manualEntry.value ? CUSTOM : port.value),
  set: (v) => {
    if (v === CUSTOM) {
      manualEntry.value = true
      return
    }
    manualEntry.value = false
    port.value = v
  },
})

const conflict = computed(() => (port.value ? usedBy.value.get(bare(port.value)) : undefined))

const summary = computed(() => {
  const p = bare(port.value) || 'port'
  return `${p} · ${baudRate.value} ${dataBits.value}${parity.value[0].toUpperCase()}${stopBits.value}`
})

onMounted(async () => {
  if (props.configId) {
    const existing = flowStore.configs.find((c) => c.id === props.configId)
    if (existing) {
      isEditing.value = true
      name.value = existing.name ?? ''
      const cfg = (existing.config ?? {}) as Record<string, unknown>
      port.value = (cfg.port as string) ?? ''
      baudRate.value = (cfg.baudRate as number) ?? 9600
      dataBits.value = (cfg.dataBits as number) ?? 8
      parity.value = (cfg.parity as string) ?? 'none'
      stopBits.value = (cfg.stopBits as number) ?? 1
    }
  }
  try {
    devicePorts.value = await api.listUartPorts()
  } catch {
    devicePorts.value = []
  }
  if (devicePorts.value.length > 0 && (port.value === '' || devicePorts.value.includes(bare(port.value)))) {
    manualEntry.value = false
    port.value = bare(port.value)
  }
})

function save() {
  const config = {
    port: bare(port.value),
    baudRate: baudRate.value,
    dataBits: dataBits.value,
    parity: parity.value,
    stopBits: stopBits.value,
  }
  // Without a name, the connection is named after its port and settings,
  // which is what the nodes then show.
  const label = name.value.trim() || summary.value
  if (isEditing.value && props.configId) {
    flowStore.updateConfig(props.configId, { name: label, config })
  } else {
    flowStore.addConfig({ id: crypto.randomUUID(), type: 'serial-port', name: label, config })
  }
  cancel()
}

function cancel() {
  ui.clearConfigEditor()
}

const canSave = computed(() => bare(port.value) !== '')
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-3">
      <div class="flex items-center gap-2 mb-1">
        <span class="text-accent text-sm font-bold uppercase tracking-wider">
          {{ isEditing ? 'Edit' : 'New' }} Serial Port
        </span>
      </div>

      <FormField
        label="Port"
        :error="!bare(port) ? 'Port required' : conflict ? `Also used by ${conflict}: a port can only have one connection` : ''"
      >
        <template v-if="devicePorts.length > 0" #action>
          <button
            type="button"
            class="text-[9px] text-terminal-text-dim hover:text-accent underline underline-offset-2"
            @click="manualEntry = !manualEntry"
          >
            {{ manualEntry ? 'Choose from the device ports' : 'Type manually' }}
          </button>
        </template>
        <FormSelect
          v-if="!manualEntry"
          v-model="selectedPort"
          :options="portOptions"
          placeholder="Select a port…"
          width="100%"
        />
        <FormInput v-else v-model="port" placeholder="ttyS3, ttyUSB0…" mono :invalid="!bare(port)" />
      </FormField>

      <FormField label="Baud rate">
        <FormSelect v-model="baudRate" :options="BAUD_RATES" width="100%" />
      </FormField>
      <FormField label="Data bits">
        <ToggleGroup v-model="dataBits" :options="DATA_BITS" />
      </FormField>
      <FormField label="Parity">
        <ToggleGroup v-model="parity" :options="PARITY" />
      </FormField>
      <FormField label="Stop bits">
        <ToggleGroup v-model="stopBits" :options="STOP_BITS" />
      </FormField>

      <FormField label="Name" :hint="`Leave empty to use “${summary}”`">
        <FormInput v-model="name" :placeholder="summary" />
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

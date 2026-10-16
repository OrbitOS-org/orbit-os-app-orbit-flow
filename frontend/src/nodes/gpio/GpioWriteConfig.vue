<script setup lang="ts">
import { computed } from 'vue'
import { useConfigSelector } from '@/composables/useConfigSelector'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'

const { options: pinOptions, openNewConfig, openEditConfig } = useConfigSelector('gpio-pin')

const pin = useNodeProperty<string>('pin', '')
const action = useNodeProperty<string>('action', 'payload')
const onStop = useNodeProperty<string>('onStop', 'off')

const ON_STOP = [
  { value: 'off', label: 'Off' },
  { value: 'on', label: 'On' },
  { value: 'keep', label: 'Leave as is' },
]

const ACTIONS = [
  { value: 'payload', label: 'From payload' },
  { value: 'on', label: 'On' },
  { value: 'off', label: 'Off' },
  { value: 'toggle', label: 'Toggle' },
]

const isPayloadMode = computed(() => action.value === 'payload')
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Pin">
      <template #action>
        <button
          v-if="pin"
          class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors"
          @click="openEditConfig(pin)"
        >Edit</button>
        <button
          class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors"
          @click="openNewConfig()"
        >+ New</button>
      </template>
      <FormSelect
        v-model="pin"
        :options="pinOptions"
        placeholder="Select a GPIO Pin..."
      />
    </FormField>

    <FormField
      label="Action"
      :hint="isPayloadMode ? '' : 'Any incoming message triggers this — msg.payload is ignored'"
    >
      <ToggleGroup v-model="action" :options="ACTIONS" />
    </FormField>

    <div v-if="isPayloadMode" class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      <span class="font-mono text-accent">msg.payload</span> sets the level —
      accepts booleans, 0/1, and "on"/"off"/"high"/"low" strings.
    </div>
    <div v-else-if="action === 'toggle'" class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      Reads the pin's current level and writes the opposite. Adds a read
      before the write, so it's a little slower than On/Off.
    </div>

    <FormField
      label="When flows stop"
      hint="Also when this node is deleted or its flow is disabled. A deploy that keeps the node leaves the pin as it is."
    >
      <ToggleGroup v-model="onStop" :options="ON_STOP" />
    </FormField>

    <div class="text-[10px] text-terminal-text-dim leading-relaxed">
      The pin must be configured as <span class="text-accent">Output</span>.
    </div>
  </div>
</template>

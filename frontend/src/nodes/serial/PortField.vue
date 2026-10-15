<script setup lang="ts">
// The "Serial Port" field the three Serial nodes share: pick a Serial Port
// connection, edit it, or create a new one.
import { useConfigSelector } from '@/composables/useConfigSelector'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import FormSelect from '@/components/ui/FormSelect.vue'

const { options, openNewConfig, openEditConfig } = useConfigSelector('serial-port')
const port = useNodeProperty<string>('port', '')
</script>

<template>
  <FormField label="Serial Port" :error="!port ? 'Choose or add a serial port' : ''">
    <template #action>
      <button
        v-if="port"
        class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors"
        @click="openEditConfig(port)"
      >Edit</button>
      <button
        class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors"
        @click="openNewConfig()"
      >+ New</button>
    </template>
    <FormSelect v-model="port" :options="options" placeholder="Select a serial port..." />
  </FormField>
</template>

<script setup lang="ts">
import { useConfigSelector } from '@/composables/useConfigSelector'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import NumberInput from '@/components/ui/NumberInput.vue'

const { options: deviceOptions, openNewConfig, openEditConfig } = useConfigSelector('spi-device')

const device = useNodeProperty<string>('device', '')
const writeData = useNodeProperty<string>('writeData', '')
const readLength = useNodeProperty<number>('readLength', 0)
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Device">
      <template #action>
        <button
          v-if="device"
          class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors"
          @click="openEditConfig(device)"
        >Edit</button>
        <button
          class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors"
          @click="openNewConfig()"
        >+ New</button>
      </template>
      <FormSelect
        v-model="device"
        :options="deviceOptions"
        placeholder="Select an SPI Device..."
      />
    </FormField>

    <FormField label="Write Data (optional)" hint="Hex bytes, e.g. '9F 00 00' or '0x9F,0x00,0x00'. Empty uses msg.payload">
      <FormInput v-model="writeData" placeholder="9F 00 00" mono />
    </FormField>

    <FormField label="Read Length" hint="MISO bytes to return (full-duplex). 0 = write-only. Overridable via msg.readLength">
      <NumberInput v-model="readLength" :min="0" />
    </FormField>

    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      When Read Length &gt; 0, <span class="font-mono text-accent">msg.payload</span> becomes
      the bytes read back (as a number array). Otherwise the message passes
      through unchanged.
    </div>
  </div>
</template>

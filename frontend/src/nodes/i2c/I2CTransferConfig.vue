<script setup lang="ts">
import { useConfigSelector } from '@/composables/useConfigSelector'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import NumberInput from '@/components/ui/NumberInput.vue'

const { options: busOptions, openNewConfig, openEditConfig } = useConfigSelector('i2c-bus')

const bus = useNodeProperty<string>('bus', '')
const address = useNodeProperty<number>('address', 0)
const writeData = useNodeProperty<string>('writeData', '')
const readLength = useNodeProperty<number>('readLength', 0)
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Bus">
      <template #action>
        <button
          v-if="bus"
          class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors"
          @click="openEditConfig(bus)"
        >Edit</button>
        <button
          class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors"
          @click="openNewConfig()"
        >+ New</button>
      </template>
      <FormSelect
        v-model="bus"
        :options="busOptions"
        placeholder="Select an I2C Bus..."
      />
    </FormField>

    <FormField label="Address (7-bit)" hint="0x00-0x7F. Overridable via msg.address">
      <FormInput
        :model-value="'0x' + address.toString(16).toUpperCase().padStart(2, '0')"
        mono
        @update:model-value="(v: string) => address = parseInt(String(v).replace(/^0x/i, ''), 16) || 0"
      />
    </FormField>

    <FormField label="Write Data (optional)" hint="Hex bytes, e.g. '01 02' or '0x01,0x02'. Empty uses msg.payload">
      <FormInput v-model="writeData" placeholder="01 02" mono />
    </FormField>

    <FormField label="Read Length" hint="Bytes to read back. 0 = write-only. Overridable via msg.readLength">
      <NumberInput v-model="readLength" :min="0" />
    </FormField>

    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      When Read Length &gt; 0, <span class="font-mono text-accent">msg.payload</span> becomes
      the bytes read back (as a number array). Otherwise the message passes
      through unchanged.
    </div>
  </div>
</template>

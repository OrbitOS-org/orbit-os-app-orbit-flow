<script setup lang="ts">
import { useConfigSelector } from '@/composables/useConfigSelector'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import FormSelect from '@/components/ui/FormSelect.vue'

const { options: pinOptions, openNewConfig, openEditConfig } = useConfigSelector('gpio-pin')

const pin = useNodeProperty<string>('pin', '')
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

    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      Reads the current level on every incoming message and forwards it as
      <span class="font-mono text-accent">msg.payload</span> (true/false). The
      pin must be configured as <span class="text-accent">Input</span> — wire
      an Inject node into this one to poll on an interval.
    </div>
  </div>
</template>

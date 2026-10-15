<script setup lang="ts">
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import PortField from './PortField.vue'
import EndingField from './EndingField.vue'
import SplitFields from './SplitFields.vue'
import OutputField from './OutputField.vue'

const timeout = useNodeProperty<number>('timeout', 1000)
</script>

<template>
  <div class="flex flex-col gap-3">
    <PortField />
    <EndingField label="Add at the end of the request" />
    <SplitFields label="Reply ends" />
    <FormField label="Timeout" hint="Overridable via msg.timeout (ms)">
      <NumberInput v-model="timeout" :min="1" unit="ms" />
    </FormField>
    <OutputField />
    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      Sends <span class="font-mono text-accent">msg.payload</span> and waits for the reply, which
      replaces <span class="font-mono text-accent">msg.payload</span>; the rest of the message is
      kept. Requests wait their turn, one at a time per port. With no reply in time the node
      raises an error that a Catch node can handle.
    </div>
  </div>
</template>

<script setup lang="ts">
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'

const OUTPUTS = [
  { value: 'text', label: 'Text' },
  { value: 'bytes', label: 'Bytes' },
]
const output = useNodeProperty<string>('output', 'text')
</script>

<template>
  <FormField label="Output" hint="Bytes gives msg.payload as a list of numbers (0-255)">
    <ToggleGroup v-model="output" :options="OUTPUTS" />
  </FormField>
</template>

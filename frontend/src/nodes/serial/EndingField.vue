<script setup lang="ts">
// What to add after each payload sent: nothing, \n or \r\n, or any other
// characters typed in.
import { computed } from 'vue'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'

defineProps<{ label: string }>()

const PRESETS = ['', '\\n', '\\r\\n']
const OPTIONS = [
  { value: '', label: 'Nothing' },
  { value: '\\n', label: '\\n' },
  { value: '\\r\\n', label: '\\r\\n' },
  { value: 'other', label: 'Other' },
]

const addChar = useNodeProperty<string>('addChar', '')
const preset = computed<string>({
  get: () => (PRESETS.includes(addChar.value) ? addChar.value : 'other'),
  set: (v) => { addChar.value = v === 'other' ? ';' : v },
})
</script>

<template>
  <FormField :label="label">
    <ToggleGroup v-model="preset" :options="OPTIONS" />
  </FormField>
  <FormField v-if="preset === 'other'" label="Characters" hint="e.g. ; or \x03 or 0x0D 0x0A">
    <FormInput v-model="addChar" mono />
  </FormField>
</template>

<script setup lang="ts">
// Where one piece of incoming data ends, as Node-RED's "Split input":
// on a character, after a silence, or at a fixed length. Used by Serial In
// (for each message) and Serial Request (for the reply).
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormCheckbox from '@/components/ui/FormCheckbox.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'

defineProps<{ label: string }>()

const MODES = [
  { value: 'char', label: 'On character' },
  { value: 'gap', label: 'After silence' },
  { value: 'len', label: 'Fixed length' },
]

const split = useNodeProperty<string>('split', 'char')
const splitChar = useNodeProperty<string>('splitChar', '\\n')
const keepChar = useNodeProperty<boolean>('keepChar', false)
const splitGap = useNodeProperty<number>('splitGap', 50)
const splitLength = useNodeProperty<number>('splitLength', 16)
</script>

<template>
  <FormField :label="label">
    <ToggleGroup v-model="split" :options="MODES" />
  </FormField>
  <template v-if="split === 'char'">
    <FormField
      label="Character"
      hint="\n line feed, \r\n carriage return + line feed, \x03 any byte, or 0x0D"
      :error="!splitChar ? 'Type the character that ends a piece' : ''"
    >
      <FormInput v-model="splitChar" placeholder="\n" mono :invalid="!splitChar" />
    </FormField>
    <FormCheckbox v-model="keepChar" label="Keep the character at the end" />
  </template>
  <FormField v-else-if="split === 'gap'" label="Silence" hint="A piece ends when no byte arrives for this long">
    <NumberInput v-model="splitGap" :min="1" unit="ms" />
  </FormField>
  <FormField v-else label="Length">
    <NumberInput v-model="splitLength" :min="1" :max="65536" unit="bytes" />
  </FormField>
</template>

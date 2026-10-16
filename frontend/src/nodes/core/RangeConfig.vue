<script setup lang="ts">
// Range: map a number from one range to another — Node-RED's "range".
import { computed } from 'vue'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import FormCheckbox from '@/components/ui/FormCheckbox.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'

const action = useNodeProperty<string>('action', 'scale')
const minin = useNodeProperty<number>('minin', 0)
const maxin = useNodeProperty<number>('maxin', 4095)
const minout = useNodeProperty<number>('minout', 0)
const maxout = useNodeProperty<number>('maxout', 100)
const round = useNodeProperty<boolean>('round', false)
const property = useNodeProperty<string>('property', 'payload')

const actions = [
  { value: 'scale', label: 'Scale the value' },
  { value: 'clamp', label: 'Scale and limit to the target range' },
  { value: 'roll', label: 'Scale and wrap within the target range' },
  { value: 'drop', label: 'Scale, but drop values outside the input range' },
]
const emptyRange = computed(() => Number(minin.value) === Number(maxin.value))
const example = computed(() => {
  const a = Number(minin.value), b = Number(maxin.value), c = Number(minout.value), d = Number(maxout.value)
  if (a === b) return ''
  const mid = (a + b) / 2
  let out = (mid - a) / (b - a) * (d - c) + c
  if (round.value) out = Math.round(out)
  return `${mid} → ${Number(out.toFixed(4))}`
})
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Property">
      <FormInput v-model="property" placeholder="payload" mono :invalid="!property">
        <template #prefix>msg.</template>
      </FormInput>
    </FormField>

    <FormField label="Action">
      <FormSelect :model-value="action" :options="actions" @update:model-value="action = String($event)" />
    </FormField>

    <FormField label="Map the input range" :error="emptyRange ? 'The two values must be different.' : ''">
      <div class="flex gap-1 items-center">
        <NumberInput :model-value="minin" :invalid="emptyRange" @update:model-value="minin = Number($event)" />
        <span class="text-[10px] text-terminal-text-dim px-1">to</span>
        <NumberInput :model-value="maxin" :invalid="emptyRange" @update:model-value="maxin = Number($event)" />
      </div>
    </FormField>

    <FormField label="To the target range">
      <div class="flex gap-1 items-center">
        <NumberInput :model-value="minout" @update:model-value="minout = Number($event)" />
        <span class="text-[10px] text-terminal-text-dim px-1">to</span>
        <NumberInput :model-value="maxout" @update:model-value="maxout = Number($event)" />
      </div>
    </FormField>

    <FormCheckbox v-model="round" label="Round the result to a whole number" />

    <p v-if="example" class="text-[10px] text-terminal-text-dim">
      Example: <code class="text-terminal-text">{{ example }}</code>
    </p>
  </div>
</template>

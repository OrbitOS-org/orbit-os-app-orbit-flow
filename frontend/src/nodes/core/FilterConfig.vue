<script setup lang="ts">
// Filter (report by exception): pass only changes — Node-RED's "filter".
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import FormCheckbox from '@/components/ui/FormCheckbox.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'
import { computed } from 'vue'

const func = useNodeProperty<string>('func', 'rbe')
const gap = useNodeProperty<string>('gap', '')
const inout = useNodeProperty<string>('inout', 'out')
const property = useNodeProperty<string>('property', 'payload')
const septopics = useNodeProperty<boolean>('septopics', true)
const topi = useNodeProperty<string>('topi', 'topic')

const modes = [
  { value: 'rbe', label: 'block unless the value changes' },
  { value: 'rbei', label: 'block unless the value changes (ignore the first value)' },
  { value: 'deadband', label: 'block unless the value changes by more than' },
  { value: 'deadbandEq', label: 'block unless the value changes by or more than' },
  { value: 'narrowband', label: 'block if the value changes by more than' },
  { value: 'narrowbandEq', label: 'block if the value changes by or more than' },
]
const isBand = computed(() => func.value !== 'rbe' && func.value !== 'rbei')
const gapInvalid = computed(() => isBand.value && !/^\s*\d+(\.\d+)?\s*%?\s*$/.test(gap.value ?? ''))
const compareOptions = [
  { value: 'out', label: 'last value sent' },
  { value: 'in', label: 'last value received' },
]
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Mode">
      <FormSelect :model-value="func" :options="modes" @update:model-value="func = String($event)" />
    </FormField>

    <template v-if="isBand">
      <FormField label="Change" hint="A number (e.g. 5) or a percentage of the last value (e.g. 10%).">
        <FormInput v-model="gap" placeholder="5 or 10%" mono :invalid="gapInvalid" />
      </FormField>
      <FormField label="Compare with">
        <ToggleGroup v-model="inout" :options="compareOptions" />
      </FormField>
    </template>

    <FormField label="Property">
      <FormInput v-model="property" placeholder="payload" mono :invalid="!property">
        <template #prefix>msg.</template>
      </FormInput>
    </FormField>

    <FormCheckbox v-model="septopics" label="Keep a separate value for each topic" />
    <FormField v-if="septopics" label="Topic property">
      <FormInput v-model="topi" mono><template #prefix>msg.</template></FormInput>
    </FormField>

    <p class="text-[10px] text-terminal-text-dim leading-relaxed">
      A message with <code class="text-terminal-text">msg.reset</code> clears the stored value
      (of its topic, or all of them).
    </p>
  </div>
</template>

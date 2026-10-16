<script setup lang="ts">
// Batch: group messages into sequences — Node-RED's "batch".
import FormField from '@/components/ui/FormField.vue'
import FormCheckbox from '@/components/ui/FormCheckbox.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'
import { computed } from 'vue'

const mode = useNodeProperty<string>('mode', 'count')
const count = useNodeProperty<number>('count', 10)
const overlap = useNodeProperty<number>('overlap', 0)
const interval = useNodeProperty<number>('interval', 10)
const allowEmpty = useNodeProperty<boolean>('allowEmptySequence', false)

const modes = [
  { value: 'count', label: 'by number of messages' },
  { value: 'interval', label: 'by time interval' },
]
const overlapInvalid = computed(() => Number(overlap.value) >= Number(count.value))
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Group">
      <ToggleGroup v-model="mode" :options="modes" />
    </FormField>

    <template v-if="mode === 'count'">
      <FormField label="Messages per sequence">
        <NumberInput :model-value="count" :min="1" :step="1" unit="msgs" @update:model-value="count = Number($event)" />
      </FormField>
      <FormField label="Overlap" hint="Messages repeated at the start of the next sequence." :error="overlapInvalid ? 'Must be less than the number of messages.' : ''">
        <NumberInput :model-value="overlap" :min="0" :step="1" unit="msgs" :invalid="overlapInvalid" @update:model-value="overlap = Number($event)" />
      </FormField>
    </template>

    <template v-else>
      <FormField label="Interval">
        <NumberInput :model-value="interval" :min="0.01" :step="1" unit="seconds" @update:model-value="interval = Number($event)" />
      </FormField>
      <FormCheckbox v-model="allowEmpty" label="Send an empty message when nothing arrived" />
    </template>

    <p class="text-[10px] text-terminal-text-dim leading-relaxed">
      The messages come out unchanged, each with <code class="text-terminal-text">msg.parts</code>.
      Add a Join (automatic) after it to get one list per sequence.
    </p>
  </div>
</template>

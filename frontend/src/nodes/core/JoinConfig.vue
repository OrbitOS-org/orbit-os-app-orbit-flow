<script setup lang="ts">
// Join: many messages into one — Node-RED's "join".
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'

const mode = useNodeProperty<string>('mode', 'auto')
const build = useNodeProperty<string>('build', 'array')
const property = useNodeProperty<string>('property', 'payload')
const propertyType = useNodeProperty<string>('propertyType', 'msg')
const key = useNodeProperty<string>('key', 'topic')
const joiner = useNodeProperty<string>('joiner', '\\n')
const count = useNodeProperty<number>('count', 0)
const timeout = useNodeProperty<number>('timeout', 0)

const modes = [
  { value: 'auto', label: 'automatic' },
  { value: 'custom', label: 'manual' },
]
const builds = [
  { value: 'array', label: 'a list' },
  { value: 'string', label: 'a text' },
  { value: 'buffer', label: 'bytes' },
  { value: 'object', label: 'an object { msg.topic: value }' },
  { value: 'merged', label: 'a merged object' },
]
const sources = [
  { value: 'msg', label: 'a property' },
  { value: 'full', label: 'the whole message' },
]
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Mode">
      <ToggleGroup v-model="mode" :options="modes" />
    </FormField>

    <p v-if="mode === 'auto'" class="text-[10px] text-terminal-text-dim leading-relaxed">
      Puts back together a sequence made by a Split or a Batch, using
      <code class="text-terminal-text">msg.parts</code>: a list, text, bytes or object, as it was.
    </p>

    <template v-else>
      <FormField label="Combine each">
        <ToggleGroup v-model="propertyType" :options="sources" />
        <FormInput v-if="propertyType === 'msg'" v-model="property" placeholder="payload" mono>
          <template #prefix>msg.</template>
        </FormInput>
      </FormField>

      <FormField label="To create">
        <FormSelect :model-value="build" :options="builds" @update:model-value="build = String($event)" />
      </FormField>

      <FormField v-if="build === 'object'" label="Key">
        <FormInput v-model="key" mono><template #prefix>msg.</template></FormInput>
      </FormField>
      <FormField v-if="build === 'string' || build === 'buffer'" label="Joined with">
        <FormInput v-model="joiner" mono placeholder="\n" />
      </FormField>

      <FormField label="Send the message" hint="Leave both at 0 to send only on msg.complete.">
        <div class="flex gap-1 items-center">
          <span class="text-[10px] text-terminal-text-dim pr-1">after</span>
          <NumberInput :model-value="count" :min="0" :step="1" unit="parts" @update:model-value="count = Number($event)" />
        </div>
        <div class="flex gap-1 items-center">
          <span class="text-[10px] text-terminal-text-dim pr-1">or after</span>
          <NumberInput :model-value="timeout" :min="0" :step="1" unit="s from the first" @update:model-value="timeout = Number($event)" />
        </div>
      </FormField>
    </template>

    <p class="text-[10px] text-terminal-text-dim leading-relaxed">
      <code class="text-terminal-text">msg.complete</code> sends what has been collected;
      <code class="text-terminal-text">msg.reset</code> discards it.
    </p>
  </div>
</template>

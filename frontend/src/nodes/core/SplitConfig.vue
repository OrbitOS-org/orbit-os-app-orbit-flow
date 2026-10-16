<script setup lang="ts">
// Split: one message per item — Node-RED's "split".
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'

const property = useNodeProperty<string>('property', 'payload')
const splt = useNodeProperty<string>('splt', '\\n')
const spltType = useNodeProperty<string>('spltType', 'str')
const arraySplt = useNodeProperty<number>('arraySplt', 1)
const addname = useNodeProperty<string>('addname', '')

const textOptions = [
  { value: 'str', label: 'at a separator' },
  { value: 'len', label: 'every N characters' },
]
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Split">
      <FormInput v-model="property" placeholder="payload" mono :invalid="!property">
        <template #prefix>msg.</template>
      </FormInput>
    </FormField>

    <FormField label="Text and bytes">
      <ToggleGroup v-model="spltType" :options="textOptions" />
      <FormInput
        v-model="splt"
        mono
        :placeholder="spltType === 'len' ? '10' : '\\n'"
      />
    </FormField>

    <FormField label="Lists" hint="Items per message.">
      <NumberInput :model-value="arraySplt" :min="1" :step="1" @update:model-value="arraySplt = Number($event)" />
    </FormField>

    <FormField label="Objects" hint="Copy each key to this message property (leave empty to skip).">
      <FormInput v-model="addname" placeholder="topic" mono>
        <template #prefix>msg.</template>
      </FormInput>
    </FormField>

    <p class="text-[10px] text-terminal-text-dim leading-relaxed">
      Each message carries <code class="text-terminal-text">msg.parts</code>; a Join in automatic
      mode puts the sequence back together. Use <code class="text-terminal-text">\n</code>,
      <code class="text-terminal-text">\t</code> or <code class="text-terminal-text">\r</code> for special characters.
    </p>
  </div>
</template>

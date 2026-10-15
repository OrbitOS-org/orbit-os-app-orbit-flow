<script setup lang="ts">
// Write File, as Node-RED's write file node: File, Action (append /
// overwrite / delete), "Add newline". Folders are created when needed;
// text is written as UTF-8 and a byte array as bytes.
import { computed } from 'vue'
import FormField from '@/components/ui/FormField.vue'
import FormCheckbox from '@/components/ui/FormCheckbox.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FileField from './FileField.vue'

const mode = useNodeProperty<string>('mode', 'append')
const appendNewline = useNodeProperty<boolean>('appendNewline', true)

const ACTIONS = [
  { value: 'append', label: 'Append' },
  { value: 'overwrite', label: 'Overwrite' },
  { value: 'delete', label: 'Delete' },
]

// Flows saved before keep "create only"; it shows as Overwrite here.
const action = computed<string>({
  get: () => (mode.value === 'create' ? 'overwrite' : mode.value),
  set: (v) => { mode.value = v },
})

const HELP: Record<string, string> = {
  append: 'Adds each msg.payload to the end of the file.',
  overwrite: 'Replaces the file with msg.payload.',
  delete: 'Deletes the file when a message arrives. The payload is not used.',
}
</script>

<template>
  <div class="flex flex-col gap-3 flex-1 min-h-0">
    <FileField />

    <FormField label="Action">
      <ToggleGroup v-model="action" :options="ACTIONS" />
    </FormField>

    <FormCheckbox v-if="action !== 'delete'" v-model="appendNewline" label="Add newline (\n) to each payload" />

    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      {{ HELP[action] }}
      <template v-if="action !== 'delete'">
        Folders are created when needed. Text is written as UTF-8, a list of numbers (0-255) as bytes, other values as JSON.
      </template>
    </div>
  </div>
</template>

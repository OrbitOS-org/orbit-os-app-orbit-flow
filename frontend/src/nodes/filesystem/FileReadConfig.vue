<script setup lang="ts">
// Read File, as Node-RED's read file node: File and Output (a single
// string / a msg per line / bytes), plus "new lines since the last read"
// for following a log. A folder in File gives the list of its files.
import { computed } from 'vue'
import FormField from '@/components/ui/FormField.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FileField from './FileField.vue'

const outputProp = useNodeProperty<string>('output', '')
const encoding = useNodeProperty<string>('encoding', 'utf-8')
const incremental = useNodeProperty<boolean>('incremental', false)

// Flows saved before "output" existed: incremental → new, binary → bytes.
const output = computed<string>({
  get: () => outputProp.value || (incremental.value ? 'new' : encoding.value === 'binary' ? 'bytes' : 'string'),
  set: (v) => { outputProp.value = v },
})

const OUTPUTS = [
  { value: 'string', label: 'a single string' },
  { value: 'lines', label: 'a msg per line' },
  { value: 'new', label: 'new lines since the last read' },
  { value: 'bytes', label: 'bytes' },
]

const ENCODINGS = [
  { value: 'utf-8', label: 'UTF-8' },
  { value: 'utf-16', label: 'UTF-16 (detect from the file)' },
  { value: 'utf-16le', label: 'UTF-16 LE (Excel "Unicode text")' },
  { value: 'utf-16be', label: 'UTF-16 BE' },
  { value: 'latin1', label: 'Latin-1 (ISO-8859-1)' },
  { value: 'windows-1252', label: 'Windows-1252' },
]
const textEncoding = computed<string>({
  get: () => (encoding.value === 'auto' || encoding.value === 'binary' ? 'utf-8' : encoding.value),
  set: (v) => { encoding.value = v },
})

const HELP: Record<string, string> = {
  string: 'Sends the whole file as one text in msg.payload.',
  lines: 'Sends one message per line of the file.',
  new: 'Each read sends only the lines added since the previous read, one message per line, for following a log. The first read starts at the end of the file.',
  bytes: 'Sends the file as bytes (a list of numbers 0-255), for binary files.',
}
</script>

<template>
  <div class="flex flex-col gap-3 flex-1 min-h-0">
    <FileField pick="any" hint="A file, or a folder for the list of its files." />

    <FormField label="Output">
      <FormSelect v-model="output" :options="OUTPUTS" width="100%" />
    </FormField>

    <FormField v-if="output !== 'bytes'" label="Encoding">
      <FormSelect v-model="textEncoding" :options="ENCODINGS" width="100%" />
    </FormField>

    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      {{ HELP[output] }} msg.filename tells which file was read. With a folder in File,
      msg.payload is the list of its files (name, path, size, modified).
    </div>
  </div>
</template>

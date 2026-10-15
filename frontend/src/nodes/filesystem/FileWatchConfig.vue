<script setup lang="ts">
// Watch, as Node-RED's watch node: a file or folder, and whether to watch
// its sub-folders. A message for each file created, changed or deleted.
import { onMounted } from 'vue'
import FormCheckbox from '@/components/ui/FormCheckbox.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FileField from './FileField.vue'

const recursive = useNodeProperty<boolean>('recursive', false)
const mode = useNodeProperty<string>('mode', 'watch')
const incremental = useNodeProperty<boolean>('incremental', true)

// The node only watches now. A node saved with the old "Read" modes is
// switched to watching when opened; listing a folder is Read File's job.
onMounted(() => {
  if (mode.value !== 'watch') mode.value = 'watch'
  if (!incremental.value) incremental.value = true
})
</script>

<template>
  <div class="flex flex-col gap-3 flex-1 min-h-0">
    <FileField label="File or folder" pick="any" hint="A file, or a folder to watch all its files." />

    <FormCheckbox v-model="recursive" label="Watch sub-folders recursively" />

    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      Sends a message when a file is created, changed or deleted: msg.payload and msg.filename are the
      file's name (e.g. incoming/order-1029.csv), msg.event is create, write or remove. Wire it to Read File,
      with File left empty, to get the content.
    </div>
  </div>
</template>

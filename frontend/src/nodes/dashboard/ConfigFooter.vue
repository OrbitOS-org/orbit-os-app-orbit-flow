<script setup lang="ts">
// Create/Update and Cancel under a config editor.
defineProps<{ editing: boolean; canSave: boolean }>()
const emit = defineEmits<{ save: []; cancel: [] }>()
</script>

<template>
  <div class="shrink-0 flex items-center gap-2 px-4 py-2.5 border-t border-terminal-border bg-terminal-surface">
    <button
      class="px-3 py-1 text-[10px] uppercase tracking-wider font-bold bg-accent/10 text-accent border border-accent/30 hover:bg-accent/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      :disabled="!canSave"
      @click="emit('save')"
    >{{ editing ? 'Update' : 'Create' }}</button>
    <button
      class="px-3 py-1 text-[10px] uppercase tracking-wider text-terminal-text-dim border border-terminal-border hover:text-terminal-text hover:border-terminal-text-dim transition-colors"
      @click="emit('cancel')"
    >Cancel</button>
  </div>
</template>

<script setup lang="ts">
import PortField from './PortField.vue'
import EndingField from './EndingField.vue'
</script>

<template>
  <div class="flex flex-col gap-3">
    <PortField />
    <EndingField label="Add at the end" />
    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      Writes <span class="font-mono text-accent">msg.payload</span> to the port: text as it is,
      a list of numbers (0-255) as bytes, other values as text or JSON.
    </div>
  </div>
</template>

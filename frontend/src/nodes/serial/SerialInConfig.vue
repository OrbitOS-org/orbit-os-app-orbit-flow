<script setup lang="ts">
import PortField from './PortField.vue'
import SplitFields from './SplitFields.vue'
import OutputField from './OutputField.vue'
</script>

<template>
  <div class="flex flex-col gap-3">
    <PortField />
    <SplitFields label="Split input" />
    <OutputField />
    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      Sends one message for each piece of data received, with it in
      <span class="font-mono text-accent">msg.payload</span> and the port name in
      <span class="font-mono text-accent">msg.port</span>. While a Serial Request on the same
      port waits for its reply, the reply goes to it and not here.
    </div>
  </div>
</template>

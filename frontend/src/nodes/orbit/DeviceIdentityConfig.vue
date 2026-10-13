<script setup lang="ts">
// No configuration — the identity fields are fetched once and cached.
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      No configuration. Reads once (at deploy, or on the first message if the
      device wasn't reachable yet) and caches the result — this data never
      changes while the device is running, so it isn't re-fetched on every
      message. Forwards as <span class="font-mono text-accent">msg.payload</span>:
      <span class="font-mono text-accent">deviceName</span>,
      <span class="font-mono text-accent">serialNumber</span>,
      <span class="font-mono text-accent">hardwareModel</span>,
      <span class="font-mono text-accent">boardModel</span>,
      <span class="font-mono text-accent">socModel</span>,
      <span class="font-mono text-accent">architecture</span>,
      <span class="font-mono text-accent">cpuModel</span>,
      <span class="font-mono text-accent">cpuCores</span>,
      <span class="font-mono text-accent">totalRamBytes</span>,
      <span class="font-mono text-accent">osVersion</span>,
      <span class="font-mono text-accent">kernelVersion</span>.
    </div>
  </div>
</template>

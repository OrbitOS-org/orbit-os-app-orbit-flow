<script setup lang="ts">
// No configuration — every message triggers a fresh GetMetrics() call.
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="text-[10px] text-terminal-text-dim leading-relaxed border border-terminal-border bg-terminal-bg p-2 rounded">
      No configuration. Each incoming message fetches live CPU, memory,
      temperature and disk usage for this device and forwards it as
      <span class="font-mono text-accent">msg.payload</span>:
      <span class="font-mono text-accent">cpuUsagePercent</span>,
      <span class="font-mono text-accent">memoryUsagePercent</span>,
      <span class="font-mono text-accent">socTemperatureC</span>,
      <span class="font-mono text-accent">uptimeSeconds</span>,
      <span class="font-mono text-accent">memory</span>,
      <span class="font-mono text-accent">disk</span>.
    </div>
  </div>
</template>

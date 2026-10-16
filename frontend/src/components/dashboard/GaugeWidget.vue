<script setup lang="ts">
// ui-gauge: a half dial from min to max.
import { computed } from 'vue'

const props = defineProps<{ value: unknown; min: number; max: number; units: string }>()

const num = computed(() => (typeof props.value === 'number' ? props.value : null))
const frac = computed(() => {
  if (num.value === null || props.max === props.min) return 0
  return Math.max(0, Math.min(1, (num.value - props.min) / (props.max - props.min)))
})
const arc = computed(() => {
  const a = Math.PI * (1 - frac.value)
  return `M20 105 A80 80 0 0 1 ${(100 + 80 * Math.cos(a)).toFixed(2)} ${(105 - 80 * Math.sin(a)).toFixed(2)}`
})
const shown = computed(() => {
  if (num.value === null) return '–'
  const v = num.value
  return Math.abs(v) >= 100 ? v.toFixed(0) : String(Number(v.toFixed(2)))
})
</script>

<template>
  <svg class="gauge" viewBox="0 0 200 122" role="img" :aria-label="`${shown} ${units}`">
    <path class="bg" d="M20 105 A80 80 0 0 1 180 105" fill="none" stroke-width="16" stroke-linecap="round" />
    <path v-if="frac > 0" class="fg" :d="arc" fill="none" stroke-width="16" stroke-linecap="round" />
    <text x="100" y="96" text-anchor="middle" class="v">{{ shown }}</text>
    <text x="100" y="114" text-anchor="middle" class="u">{{ units }}</text>
    <text x="20" y="121" text-anchor="middle" class="m">{{ min }}</text>
    <text x="180" y="121" text-anchor="middle" class="m">{{ max }}</text>
  </svg>
</template>

<style scoped>
.gauge { width: 100%; max-width: 220px; justify-self: center; display: block; }
.bg { stroke: rgb(var(--c-surface-alt)); }
.fg { stroke: rgb(var(--c-accent)); }
.v { fill: rgb(var(--c-text-bright)); font-size: 26px; font-weight: 700; font-variant-numeric: tabular-nums; }
.u { fill: rgb(var(--c-text-dim)); font-size: 11px; }
.m { fill: rgb(var(--c-text-dim)); font-size: 9px; font-variant-numeric: tabular-nums; }
</style>

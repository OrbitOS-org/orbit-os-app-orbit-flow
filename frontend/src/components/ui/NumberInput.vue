<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: number
  min?: number
  max?: number
  step?: number
  unit?: string
  placeholder?: string
  invalid?: boolean
}>(), {
  min: -Infinity,
  max: Infinity,
  step: 1,
  unit: '',
  placeholder: '',
  invalid: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

const canDecrement = computed(() => props.modelValue > props.min)
const canIncrement = computed(() => props.modelValue < props.max)

function clamp(v: number) {
  return Math.min(props.max, Math.max(props.min, v))
}

function onInput(e: Event) {
  const raw = (e.target as HTMLInputElement).value
  const parsed = raw === '' ? 0 : Number(raw)
  if (Number.isFinite(parsed)) emit('update:modelValue', clamp(parsed))
}

function bump(delta: number) {
  emit('update:modelValue', clamp(props.modelValue + delta))
}
</script>

<template>
  <div
    class="flex items-stretch w-full bg-terminal-surface-alt/70 rounded-lg border focus-within:border-accent/60 focus-within:ring-2 focus-within:ring-accent/25 transition-colors"
    :class="invalid ? 'border-status-error' : 'border-transparent'"
  >
    <button
      type="button"
      class="px-2 text-xs text-terminal-text-dim hover:text-terminal-text hover:bg-terminal-surface-alt/40 disabled:opacity-30 disabled:cursor-not-allowed transition-colors border-r border-terminal-border"
      :disabled="!canDecrement"
      @click="bump(-step)"
      tabindex="-1"
      aria-label="Decrement"
    >−</button>

    <input
      type="number"
      :value="modelValue"
      :min="min !== -Infinity ? min : undefined"
      :max="max !== Infinity ? max : undefined"
      :step="step"
      :placeholder="placeholder"
      class="flex-1 min-w-0 bg-transparent px-2 py-1.5 text-xs text-terminal-text outline-none placeholder:text-terminal-text-dim text-right [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
      @input="onInput"
    />

    <span
      v-if="unit"
      class="flex items-center px-2 text-xs text-terminal-text-dim border-l border-terminal-border/60 shrink-0 select-none"
    >
      {{ unit }}
    </span>

    <button
      type="button"
      class="px-2 text-xs text-terminal-text-dim hover:text-terminal-text hover:bg-terminal-surface-alt/40 disabled:opacity-30 disabled:cursor-not-allowed transition-colors border-l border-terminal-border"
      :disabled="!canIncrement"
      @click="bump(step)"
      tabindex="-1"
      aria-label="Increment"
    >+</button>
  </div>
</template>

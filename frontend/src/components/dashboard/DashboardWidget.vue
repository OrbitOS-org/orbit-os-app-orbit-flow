<script setup lang="ts">
// One widget of the dashboard, drawn by its type.
import { computed, ref, watch } from 'vue'
import GaugeWidget from './GaugeWidget.vue'
import ChartWidget from './ChartWidget.vue'
import type { DashboardWidget } from '@/composables/useApi'

const props = defineProps<{
  widget: DashboardWidget
  value: unknown
  series: Record<string, { t: number; v: number }[]>
  now: number
}>()
const emit = defineEmits<{ action: [value: unknown] }>()

const p = computed(() => props.widget.props ?? {})
const num = (k: string, d: number) => (typeof p.value[k] === 'number' ? (p.value[k] as number) : d)
const str = (k: string) => (typeof p.value[k] === 'string' ? (p.value[k] as string) : '')

// Switch and slider show what the user set at once, then follow the device.
const switchOn = computed(() => props.value === true)
function toggle() { emit('action', !switchOn.value) }

const sliderValue = ref(0)
watch(() => props.value, (v) => { if (typeof v === 'number') sliderValue.value = v }, { immediate: true })
function slid() { emit('action', Number(sliderValue.value)) }

const textValue = computed(() => (props.value === undefined || props.value === null || props.value === '' ? '–' : String(props.value)))
const inputId = computed(() => `w-${props.widget.id}`)
</script>

<template>
  <div class="w">
    <template v-if="widget.type === 'ui-gauge'">
      <span class="t">{{ widget.label }}</span>
      <GaugeWidget :value="value" :min="num('min', 0)" :max="num('max', 100)" :units="str('units')" />
    </template>

    <template v-else-if="widget.type === 'ui-chart'">
      <span class="t">{{ widget.label }}</span>
      <ChartWidget :series="series" :chart-type="str('chartType') || 'line'" :units="str('units')" :keep-seconds="num('keepSeconds', 3600)" :now="now" />
    </template>

    <template v-else-if="widget.type === 'ui-text'">
      <span class="t">{{ widget.label }}</span>
      <span class="big">{{ textValue }}<small v-if="str('units') && textValue !== '–'"> {{ str('units') }}</small></span>
    </template>

    <template v-else-if="widget.type === 'ui-switch'">
      <div class="row">
        <span class="lbl">{{ widget.label }}<b>{{ switchOn ? 'On' : 'Off' }}</b></span>
        <button
          type="button"
          role="switch"
          class="switch"
          :class="{ on: switchOn }"
          :aria-checked="switchOn"
          :aria-label="widget.label"
          @click="toggle"
        />
      </div>
    </template>

    <template v-else-if="widget.type === 'ui-slider'">
      <label class="t" :for="inputId">{{ widget.label }} · <b>{{ sliderValue }}</b></label>
      <input
        :id="inputId"
        v-model.number="sliderValue"
        type="range"
        :min="num('min', 0)"
        :max="num('max', 100)"
        :step="num('step', 1)"
        @change="slid"
      >
    </template>

    <template v-else-if="widget.type === 'ui-button'">
      <button type="button" class="btn" @click="emit('action', null)">{{ widget.label || 'Button' }}</button>
    </template>
  </div>
</template>

<style scoped>
.w { background: rgb(var(--c-surface)); border-radius: 12px; padding: 12px 14px; display: grid; gap: 6px; align-content: start; min-width: 0; box-shadow: 0 1px 2px rgb(0 0 0 / 0.06); }
.t { font-size: 12.5px; color: rgb(var(--c-text-dim)); overflow-wrap: anywhere; }
.t b { color: rgb(var(--c-text-bright)); font-variant-numeric: tabular-nums; }
.big { font-size: 26px; font-weight: 700; color: rgb(var(--c-text-bright)); font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
.big small { font-size: 14px; font-weight: 500; color: rgb(var(--c-text-dim)); }
.row { display: flex; align-items: center; justify-content: space-between; gap: 10px; min-height: 40px; }
.lbl { display: grid; font-size: 12.5px; color: rgb(var(--c-text-dim)); }
.lbl b { font-size: 15px; color: rgb(var(--c-text-bright)); }
.switch { width: 51px; height: 31px; border-radius: 16px; background: rgb(var(--c-surface-alt)); position: relative; flex: none; transition: background 0.15s; border: 0; cursor: pointer; }
.switch::after { content: ""; position: absolute; left: 2px; top: 2px; width: 27px; height: 27px; border-radius: 50%; background: #fff; box-shadow: 0 1px 3px rgb(0 0 0 / 0.25); transition: left 0.15s; }
.switch.on { background: rgb(var(--c-success)); }
.switch.on::after { left: 22px; }
.switch:focus-visible, .btn:focus-visible, input:focus-visible { outline: 2px solid rgb(var(--c-accent)); outline-offset: 2px; }
input[type=range] { width: 100%; accent-color: rgb(var(--c-accent)); }
.btn { border: 0; border-radius: 10px; padding: 11px; font-size: 15px; font-weight: 600; background: rgb(var(--c-accent)); color: #fff; cursor: pointer; }
.btn:active { opacity: 0.8; }
@media (prefers-reduced-motion: reduce) { .switch, .switch::after { transition: none; } }
</style>

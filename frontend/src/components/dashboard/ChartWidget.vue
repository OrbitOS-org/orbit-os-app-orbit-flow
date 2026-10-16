<script setup lang="ts">
// ui-chart: one line (or bar) per series, over the time the node keeps.
import { computed, onMounted, onUnmounted, ref } from 'vue'

type Pt = { t: number; v: number }
const props = defineProps<{
  series: Record<string, Pt[]>
  chartType: string
  units: string
  keepSeconds: number
  now: number
}>()

const H = 150, L = 38, R = 8, T = 8, B = 20
// The drawing is as wide as the card, so text is never stretched.
const box = ref<HTMLElement | null>(null)
const W = ref(400)
let ro: ResizeObserver | undefined
onMounted(() => {
  ro = new ResizeObserver(([e]) => { W.value = Math.max(160, Math.round(e.contentRect.width)) })
  if (box.value) ro.observe(box.value)
})
onUnmounted(() => ro?.disconnect())
const COLORS = ['rgb(var(--c-accent))', '#34c759', '#ff9500', '#ff3b30', '#5ac8fa', '#af52de']

const names = computed(() => Object.keys(props.series).sort())
const all = computed(() => names.value.flatMap((n) => props.series[n] ?? []))
const tMax = computed(() => props.now)
const tMin = computed(() => {
  const earliest = all.value.length ? Math.min(...all.value.map((p) => p.t)) : props.now - 60000
  return Math.max(earliest, props.now - props.keepSeconds * 1000)
})

// A nice y range with round ticks.
const yr = computed(() => {
  const vs = all.value.map((p) => p.v)
  let lo = vs.length ? Math.min(...vs) : 0
  let hi = vs.length ? Math.max(...vs) : 1
  if (props.chartType === 'bar') lo = Math.min(0, lo)
  if (lo === hi) { lo -= 1; hi += 1 }
  const span = hi - lo
  const step = Math.pow(10, Math.floor(Math.log10(span / 3)))
  const nice = [1, 2, 5, 10].map((m) => m * step).find((s) => span / s <= 4) ?? step * 10
  lo = Math.floor(lo / nice) * nice
  hi = Math.ceil(hi / nice) * nice
  const ticks: number[] = []
  for (let v = lo; v <= hi + nice / 2; v += nice) ticks.push(Number(v.toFixed(10)))
  return { lo, hi, ticks }
})

const x = (t: number) => L + ((t - tMin.value) / Math.max(1, tMax.value - tMin.value)) * (W.value - L - R)
const y = (v: number) => T + (1 - (v - yr.value.lo) / (yr.value.hi - yr.value.lo)) * (H - T - B)

const lines = computed(() => names.value.map((n, i) => {
  const pts = (props.series[n] ?? []).filter((p) => p.t >= tMin.value)
  const d = pts.map((p, j) => `${j ? 'L' : 'M'}${x(p.t).toFixed(1)} ${y(p.v).toFixed(1)}`).join(' ')
  const last = pts[pts.length - 1]
  return { name: n, color: COLORS[i % COLORS.length], d, last }
}))

const bars = computed(() => {
  const n = names.value.length
  const slot = (W.value - L - R) / Math.max(1, n)
  return names.value.map((name, i) => {
    const pts = props.series[name] ?? []
    const v = pts.length ? pts[pts.length - 1].v : 0
    const top = y(Math.max(v, 0)), bottom = y(Math.min(v, 0))
    return { name, v, color: COLORS[i % COLORS.length], x: L + i * slot + slot * 0.2, w: slot * 0.6, y: top, h: Math.max(1, bottom - top) }
  })
})

const fmt = (v: number) => (Math.abs(v) >= 1000 ? v.toFixed(0) : String(Number(v.toFixed(2))))
const timeLabel = (t: number) => new Date(t).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: props.keepSeconds <= 600 ? '2-digit' : undefined })
const latest = computed(() => {
  const l = lines.value.find((s) => s.last)
  return l?.last ? `${fmt(l.last.v)}${props.units ? ' ' + props.units : ''}` : ''
})
</script>

<template>
  <div ref="box" class="chart-wrap">
    <div v-if="latest && names.length === 1" class="latest">{{ latest }}</div>
    <svg class="chart" :viewBox="`0 0 ${W} ${H}`" role="img">
      <g v-for="v in yr.ticks" :key="v">
        <line class="grid" :x1="L" :x2="W - R" :y1="y(v)" :y2="y(v)" />
        <text class="tick" :x="L - 4" :y="y(v) + 3" text-anchor="end">{{ fmt(v) }}</text>
      </g>
      <template v-if="chartType === 'bar'">
        <rect v-for="b in bars" :key="b.name" :x="b.x" :y="b.y" :width="b.w" :height="b.h" :fill="b.color" rx="3" />
        <text v-for="b in bars" :key="b.name + 't'" class="tick" :x="b.x + b.w / 2" :y="H - 6" text-anchor="middle">{{ b.name }}</text>
      </template>
      <template v-else>
        <text class="tick" :x="L" :y="H - 6">{{ timeLabel(tMin) }}</text>
        <text class="tick" :x="W - R" :y="H - 6" text-anchor="end">{{ timeLabel(tMax) }}</text>
        <g v-for="s in lines" :key="s.name">
          <path :d="s.d" fill="none" :stroke="s.color" stroke-width="2" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
          <circle v-if="s.last" :cx="x(s.last.t)" :cy="y(s.last.v)" r="3" :fill="s.color" />
        </g>
      </template>
    </svg>
    <div v-if="names.length > 1 && chartType !== 'bar'" class="legend">
      <span v-for="s in lines" :key="s.name"><i :style="{ background: s.color }" />{{ s.name }}<b v-if="s.last">{{ fmt(s.last.v) }}</b></span>
    </div>
    <div v-if="!names.length" class="empty">Waiting for values…</div>
  </div>
</template>

<style scoped>
.chart-wrap { display: grid; gap: 4px; position: relative; }
.latest { font-size: 24px; font-weight: 700; color: rgb(var(--c-text-bright)); font-variant-numeric: tabular-nums; }
.chart { width: 100%; height: 150px; display: block; }
.grid { stroke: rgb(var(--c-border)); stroke-width: 1; vector-effect: non-scaling-stroke; }
.tick { fill: rgb(var(--c-text-dim)); font-size: 9px; font-variant-numeric: tabular-nums; }
.legend { display: flex; flex-wrap: wrap; gap: 4px 12px; font-size: 12px; color: rgb(var(--c-text-dim)); }
.legend span { display: inline-flex; align-items: center; gap: 5px; }
.legend i { width: 9px; height: 9px; border-radius: 2px; }
.legend b { color: rgb(var(--c-text)); font-weight: 600; font-variant-numeric: tabular-nums; }
.empty { position: absolute; inset: 0; display: grid; place-items: center; font-size: 12.5px; color: rgb(var(--c-text-dim)); }
</style>

<script setup lang="ts">
// The six dashboard widgets: where they go, and their own options.
import { computed } from 'vue'
import { useFlowStore } from '@/stores/flowStore'
import { useConfigSelector } from '@/composables/useConfigSelector'
import { useNodeProperty } from '@/composables/useNodeProperty'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import FormCheckbox from '@/components/ui/FormCheckbox.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'

const flowStore = useFlowStore()
const type = computed(() => flowStore.selectedNode?.type ?? '')
const { options: groupOptions, openNewConfig, openEditConfig } = useConfigSelector('ui-group')

const group = useNodeProperty<string>('group', '')
const width = useNodeProperty<number>('width', 0)
const order = useNodeProperty<number>('order', 0)
const min = useNodeProperty<number>('min', 0)
const max = useNodeProperty<number>('max', 100)
const step = useNodeProperty<number>('step', 1)
const units = useNodeProperty<string>('units', '')
const chartType = useNodeProperty<string>('chartType', 'line')
const removeOlder = useNodeProperty<number>('removeOlder', 1)
const removeOlderUnit = useNodeProperty<string>('removeOlderUnit', '3600')
const removeOlderPoints = useNodeProperty<number>('removeOlderPoints', 1000)
const topic = useNodeProperty<string>('topic', '')
const passthru = useNodeProperty<boolean>('passthru', false)
const onvalue = useNodeProperty<string>('onvalue', 'true')
const onvalueType = useNodeProperty<string>('onvalueType', 'bool')
const offvalue = useNodeProperty<string>('offvalue', 'false')
const offvalueType = useNodeProperty<string>('offvalueType', 'bool')
const payload = useNodeProperty<string>('payload', '')
const payloadType = useNodeProperty<string>('payloadType', 'str')

const DEFAULT_WIDTH: Record<string, number> = { 'ui-chart': 8 }
const VALUE_TYPES = [
  { value: 'str', label: 'text' },
  { value: 'num', label: 'number' },
  { value: 'bool', label: 'true / false' },
  { value: 'json', label: 'JSON' },
  { value: 'date', label: 'timestamp' },
]
const CHART_TYPES = [{ value: 'line', label: 'Line' }, { value: 'bar', label: 'Bar' }]
const KEEP_UNITS = [
  { value: '60', label: 'minutes' },
  { value: '3600', label: 'hours' },
  { value: '86400', label: 'days' },
]
const hasValue = (t: string) => t !== 'date'
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Group" :error="!group ? 'Choose a group, or the widget goes to an “Other” page' : ''">
      <template #action>
        <button v-if="group" class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors" @click="openEditConfig(group)">Edit</button>
        <button class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors" @click="openNewConfig()">+ New</button>
      </template>
      <FormSelect v-model="group" :options="groupOptions" placeholder="Select a group…" width="100%" />
    </FormField>

    <div class="flex gap-2">
      <FormField label="Width" class="flex-1 min-w-0" :hint="`0 = default (${DEFAULT_WIDTH[type] ?? 4} of 12)`">
        <NumberInput :model-value="width" :min="0" :max="12" :step="1" @update:model-value="width = Number($event)" />
      </FormField>
      <FormField label="Order" class="flex-1 min-w-0">
        <NumberInput :model-value="order" :min="0" :step="1" @update:model-value="order = Number($event)" />
      </FormField>
    </div>
    <p class="text-[10px] text-terminal-text-dim leading-relaxed -mt-1">The label on the dashboard is the node's name.</p>

    <template v-if="type === 'ui-gauge' || type === 'ui-slider'">
      <div class="flex gap-2">
        <FormField label="Min" class="flex-1 min-w-0"><NumberInput :model-value="min" @update:model-value="min = Number($event)" /></FormField>
        <FormField label="Max" class="flex-1 min-w-0"><NumberInput :model-value="max" @update:model-value="max = Number($event)" /></FormField>
      </div>
      <FormField v-if="type === 'ui-slider'" label="Step"><NumberInput :model-value="step" :min="0" @update:model-value="step = Number($event)" /></FormField>
    </template>

    <FormField v-if="type === 'ui-gauge' || type === 'ui-text' || type === 'ui-chart'" label="Units">
      <FormInput v-model="units" placeholder="e.g. °C, %, W" />
    </FormField>

    <template v-if="type === 'ui-chart'">
      <FormField label="Type"><ToggleGroup v-model="chartType" :options="CHART_TYPES" /></FormField>
      <FormField label="Keep" hint="Points older than this are dropped. Each msg.topic is a line; msg.payload = [] empties the chart.">
        <div class="flex gap-1 items-center">
          <NumberInput :model-value="removeOlder" :min="0.01" :step="1" @update:model-value="removeOlder = Number($event)" />
          <FormSelect :model-value="removeOlderUnit" :options="KEEP_UNITS" @update:model-value="removeOlderUnit = String($event)" />
        </div>
        <div class="flex gap-1 items-center">
          <span class="text-[10px] text-terminal-text-dim pr-1">and at most</span>
          <NumberInput :model-value="removeOlderPoints" :min="10" :step="100" unit="points per line" @update:model-value="removeOlderPoints = Number($event)" />
        </div>
      </FormField>
    </template>

    <template v-if="type === 'ui-switch'">
      <FormField label="On value">
        <div class="flex gap-1">
          <FormSelect :model-value="onvalueType" :options="VALUE_TYPES.slice(0, 4)" width="120px" @update:model-value="onvalueType = String($event)" />
          <FormInput v-model="onvalue" mono />
        </div>
      </FormField>
      <FormField label="Off value">
        <div class="flex gap-1">
          <FormSelect :model-value="offvalueType" :options="VALUE_TYPES.slice(0, 4)" width="120px" @update:model-value="offvalueType = String($event)" />
          <FormInput v-model="offvalue" mono />
        </div>
      </FormField>
    </template>

    <template v-if="type === 'ui-button'">
      <FormField label="Send">
        <div class="flex gap-1">
          <FormSelect :model-value="payloadType" :options="VALUE_TYPES" width="120px" @update:model-value="payloadType = String($event)" />
          <FormInput v-if="hasValue(payloadType)" v-model="payload" mono />
        </div>
      </FormField>
    </template>

    <template v-if="type === 'ui-switch' || type === 'ui-slider' || type === 'ui-button'">
      <FormField label="Topic" hint="Set on the message the widget sends (optional).">
        <FormInput v-model="topic" mono />
      </FormField>
      <FormCheckbox
        v-if="type !== 'ui-button'"
        v-model="passthru"
        label="Pass messages received on the input through to the output"
      />
    </template>

    <p class="text-[10px] text-terminal-text-dim leading-relaxed">
      <template v-if="type === 'ui-switch'">Using the switch sends the on or off value. A message received sets the switch (the on value, or true, turns it on).</template>
      <template v-else-if="type === 'ui-slider'">Moving the slider sends its value when released. A number received moves it.</template>
      <template v-else-if="type === 'ui-button'">Pressing the button sends the message above. A message received presses it.</template>
      <template v-else>Shows <code class="text-terminal-text">msg.payload</code>.</template>
      The dashboard opens from the button at the top of the editor.
    </p>
  </div>
</template>

<script setup lang="ts">
// Trigger: send a message, wait, send a second one — Node-RED's options.
import { computed, nextTick } from 'vue'
import { useVueFlow } from '@vue-flow/core'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import FormCheckbox from '@/components/ui/FormCheckbox.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'
import { useFlowStore } from '@/stores/flowStore'

const flowStore = useFlowStore()
const { updateNodeInternals } = useVueFlow()

const op1 = useNodeProperty<string>('op1', '1')
const op1type = useNodeProperty<string>('op1type', 'str')
const op2 = useNodeProperty<string>('op2', '0')
const op2type = useNodeProperty<string>('op2type', 'str')
const then = useNodeProperty<string>('then', 'wait')
const duration = useNodeProperty<number>('duration', 250)
const units = useNodeProperty<string>('units', 'ms')
const extend = useNodeProperty<boolean>('extend', false)
const overrideDelay = useNodeProperty<boolean>('overrideDelay', false)
const reset = useNodeProperty<string>('reset', '')
const bytopic = useNodeProperty<string>('bytopic', 'all')
const topic = useNodeProperty<string>('topic', 'topic')

// The second output is a port on the node: keep data.outputs in step.
const second = computed<boolean>({
  get: () => Boolean((flowStore.selectedNode?.data?.config as Record<string, unknown> | undefined)?.second),
  set: (v) => {
    const node = flowStore.selectedNode
    if (!node) return
    const cfg = (node.data?.config ?? {}) as Record<string, unknown>
    flowStore.updateNodeData(node.id, { config: { ...cfg, second: v }, outputs: v ? 2 : 1 })
    nextTick(() => updateNodeInternals([node.id]))
  },
})

const types = [
  { value: 'str', label: 'text' },
  { value: 'num', label: 'number' },
  { value: 'bool', label: 'true / false' },
  { value: 'json', label: 'JSON' },
  { value: 'date', label: 'timestamp' },
  { value: 'pay', label: 'the original msg' },
  { value: 'nul', label: 'nothing' },
]
const types2 = [...types.slice(0, 6), { value: 'payl', label: 'the latest msg' }, types[6]]
const hasValue = (t: string) => ['str', 'num', 'bool', 'json'].includes(t)

const thenOptions = [
  { value: 'wait', label: 'wait for' },
  { value: 'reset', label: 'wait to be reset' },
  { value: 'resend', label: 'resend every' },
]
const unitOptions = [
  { value: 'ms', label: 'milliseconds' },
  { value: 's', label: 'seconds' },
  { value: 'min', label: 'minutes' },
  { value: 'hr', label: 'hours' },
]
const topicOptions = [
  { value: 'all', label: 'all messages' },
  { value: 'topic', label: 'each msg.topic' },
]
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Send">
      <div class="flex gap-1">
        <FormSelect :model-value="op1type" :options="types" width="140px" @update:model-value="op1type = String($event)" />
        <FormInput v-if="hasValue(op1type)" v-model="op1" mono />
      </div>
    </FormField>

    <FormField label="Then">
      <ToggleGroup v-model="then" :options="thenOptions" />
    </FormField>

    <FormField v-if="then !== 'reset'" :label="then === 'resend' ? 'Every' : 'Wait'">
      <div class="flex gap-1">
        <NumberInput :model-value="duration" :min="0" :step="1" @update:model-value="duration = Number($event)" />
        <FormSelect :model-value="units" :options="unitOptions" @update:model-value="units = String($event)" />
      </div>
    </FormField>

    <template v-if="then === 'wait'">
      <FormCheckbox v-model="extend" label="Extend the wait when a new message arrives" />
      <FormCheckbox v-model="overrideDelay" label="Override the wait with msg.delay (ms)" />
      <FormField label="Then send">
        <div class="flex gap-1">
          <FormSelect :model-value="op2type" :options="types2" width="140px" @update:model-value="op2type = String($event)" />
          <FormInput v-if="hasValue(op2type)" v-model="op2" mono />
        </div>
      </FormField>
      <FormCheckbox v-model="second" label="Send the second message on a second output" />
    </template>

    <FormField label="Reset when" hint="msg.reset is set, or msg.payload equals this value (leave empty for msg.reset only).">
      <FormInput v-model="reset" placeholder="msg.payload value" mono />
    </FormField>

    <FormField label="Handle">
      <ToggleGroup v-model="bytopic" :options="topicOptions" />
    </FormField>
    <FormField v-if="bytopic === 'topic'" label="Topic property">
      <FormInput v-model="topic" mono><template #prefix>msg.</template></FormInput>
    </FormField>
  </div>
</template>

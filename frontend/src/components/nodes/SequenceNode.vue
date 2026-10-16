<script setup lang="ts">
// Trigger, Filter, Range, Split, Join and Batch on the canvas. The line
// under the name is the node's summary (e.g. "on · 60 s · off").
import { computed } from 'vue'
import BaseNode from '@/components/nodes/BaseNode.vue'
import { getNodeSummary } from '@/components/help'

defineOptions({ inheritAttrs: false })

interface Props {
  id: string
  type: string
  data: {
    label?: string
    config?: Record<string, unknown>
    status?: { fill?: string; shape?: string; text?: string } | null
    inputs?: number
    outputs?: number
    disabled?: boolean
  }
  selected?: boolean
}

const props = withDefaults(defineProps<Props>(), { selected: false })

const summary = computed(() => getNodeSummary(props.type, props.data.config) ?? '')
// The trigger can send its second message on a second output.
const outputs = computed(() =>
  props.data.outputs ?? (props.type === 'trigger' && props.data.config?.second ? 2 : 1))
</script>

<template>
  <BaseNode
    :id="props.id"
    :label="props.data.label"
    :node-type="props.type"
    :selected="props.selected"
    :inputs="props.data.inputs ?? 1"
    :outputs="outputs"
    :status="(props.data.status as any)"
    :disabled="props.data.disabled"
  >
    <template #body>
      <span class="truncate block">{{ summary }}</span>
    </template>
  </BaseNode>
</template>

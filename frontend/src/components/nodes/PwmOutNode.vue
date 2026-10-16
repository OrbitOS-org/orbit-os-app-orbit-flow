<script setup lang="ts">
// PWM Out on the canvas: channel and frequency under the name.
import { computed } from 'vue'
import BaseNode from '@/components/nodes/BaseNode.vue'
import { getNodeSummary } from '@/components/help'

defineOptions({ inheritAttrs: false })

interface Props {
  id: string
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
const summary = computed(() => getNodeSummary('pwm-out', props.data.config) ?? '')
</script>

<template>
  <BaseNode
    :id="props.id"
    :label="props.data.label"
    node-type="pwm-out"
    :selected="props.selected"
    :inputs="props.data.inputs ?? 1"
    :outputs="props.data.outputs ?? 1"
    :status="(props.data.status as any)"
    :disabled="props.data.disabled"
  >
    <template #body>
      <span class="truncate block">{{ summary }}</span>
    </template>
  </BaseNode>
</template>

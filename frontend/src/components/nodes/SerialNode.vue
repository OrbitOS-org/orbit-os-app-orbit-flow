<script setup lang="ts">
// Serial In / Serial Out / Serial Request on the canvas. The line under the
// name is the Serial Port the node uses.
import BaseNode from '@/components/nodes/BaseNode.vue'
import { useFlowStore } from '@/stores/flowStore'

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
const flowStore = useFlowStore()

const PORTS: Record<string, [number, number]> = {
  'serial-in': [0, 1],
  'serial-out': [1, 0],
  'serial-request': [1, 1],
}

function portText(): string {
  const id = props.data.config?.port as string | undefined
  if (!id) return 'no port selected'
  const cfg = flowStore.configs.find((c) => c.id === id)
  return cfg?.name || String((cfg?.config as Record<string, unknown> | undefined)?.port ?? 'Serial Port')
}
</script>

<template>
  <BaseNode
    :id="props.id"
    :label="props.data.label"
    :node-type="props.type"
    :selected="props.selected"
    :inputs="props.data.inputs ?? PORTS[props.type]?.[0] ?? 1"
    :outputs="props.data.outputs ?? PORTS[props.type]?.[1] ?? 1"
    :status="(props.data.status as any)"
    :disabled="props.data.disabled"
  >
    <template #body>
      <span class="truncate block">{{ portText() }}</span>
    </template>
  </BaseNode>
</template>

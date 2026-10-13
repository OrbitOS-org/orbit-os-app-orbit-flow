<script setup lang="ts">
import BaseNode from '@/components/nodes/BaseNode.vue'
import { useFlowStore } from '@/stores/flowStore'

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

const flowStore = useFlowStore()

function bodyText(): string {
  const busId = props.data.config?.bus as string | undefined
  const busLabel = busId ? flowStore.configs.find((c) => c.id === busId)?.name || 'I2C Bus' : 'no bus selected'
  const addr = props.data.config?.address as number | undefined
  if (addr === undefined) return busLabel
  return `${busLabel} · 0x${addr.toString(16).toUpperCase().padStart(2, '0')}`
}
</script>

<template>
  <BaseNode
    :id="props.id"
    :label="props.data.label"
    node-type="i2c-transfer"
    :selected="props.selected"
    :inputs="props.data.inputs ?? 1"
    :outputs="props.data.outputs ?? 1"
    :status="(props.data.status as any)"
    :disabled="props.data.disabled"
  >
    <template #body>
      <span class="truncate block">{{ bodyText() }}</span>
    </template>
  </BaseNode>
</template>

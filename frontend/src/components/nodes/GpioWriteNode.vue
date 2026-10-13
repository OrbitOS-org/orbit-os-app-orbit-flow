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
  const pinId = props.data.config?.pin as string | undefined
  const pinLabel = pinId
    ? flowStore.configs.find((c) => c.id === pinId)?.name || 'GPIO Pin'
    : 'no pin selected'
  const action = (props.data.config?.action as string) || 'payload'
  if (action === 'payload') return pinLabel
  return `${pinLabel} · ${action}`
}
</script>

<template>
  <BaseNode
    :id="props.id"
    :label="props.data.label"
    node-type="gpio-write"
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

<script setup lang="ts">
// Dashboard widgets on the canvas: the group they go in under the name.
import { computed } from 'vue'
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

const OUTPUTS: Record<string, number> = { 'ui-switch': 1, 'ui-slider': 1, 'ui-button': 1 }
const groupText = computed(() => {
  const id = props.data.config?.group as string | undefined
  if (!id) return 'no group'
  const g = flowStore.configs.find((c) => c.id === id)
  const page = g ? flowStore.configs.find((c) => c.id === (g.config as Record<string, unknown>)?.page) : undefined
  return [page?.name, g?.name].filter(Boolean).join(' › ') || 'group'
})
</script>

<template>
  <BaseNode
    :id="props.id"
    :label="props.data.label"
    :node-type="props.type"
    :selected="props.selected"
    :inputs="props.data.inputs ?? 1"
    :outputs="props.data.outputs ?? OUTPUTS[props.type] ?? 0"
    :status="(props.data.status as any)"
    :disabled="props.data.disabled"
  >
    <template #body>
      <span class="truncate block">{{ groupText }}</span>
    </template>
  </BaseNode>
</template>

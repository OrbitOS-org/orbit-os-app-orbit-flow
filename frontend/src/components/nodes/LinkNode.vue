<script setup lang="ts">
import { computed } from "vue";
import { Handle, Position } from "@vue-flow/core";
import type { NodeProps } from "@vue-flow/core";
import NodeIcon from "@/components/nodes/NodeIcon.vue";
import { nodeFill, NODE_INK } from "@/components/nodes/tokens";
import { useFlowStore } from "@/stores/flowStore";

defineOptions({ inheritAttrs: false });

const props = defineProps<NodeProps>();
const flowStore = useFlowStore();

const nodeType = computed(() => props.type ?? "link-in");
const fill = computed(() => nodeFill(nodeType.value));
const isDirty = computed(() => flowStore.isNodeDirty(props.id));
const label = computed(() => props.data?.label || "");

const hasInput = computed(() => (props.data?.inputs ?? 0) > 0);
const hasOutput = computed(() => (props.data?.outputs ?? 0) > 0);
</script>

<template>
  <!-- Node-RED link nodes: a small square with the icon only. -->
  <div
    class="flow-link-node relative select-none flex items-center justify-center"
    :class="{ 'opacity-45': props.data?.disabled, selected: props.selected, dashed: props.data?.disabled }"
    :style="{ background: fill, color: NODE_INK }"
  >
    <Handle v-if="hasInput" id="input-0" type="target" :position="Position.Left" />
    <Handle v-if="hasOutput" id="output-0" type="source" :position="Position.Right" />

    <NodeIcon :type="nodeType" />

    <span v-if="isDirty" class="link-dirty" title="Undeployed changes" />

    <div v-if="label" class="link-label">{{ label }}</div>
  </div>
</template>

<style scoped>
.flow-link-node {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.16), 0 1px 2px rgb(0 0 0 / 0.08);
  transition: box-shadow 0.15s;
}
.flow-link-node.selected { box-shadow: 0 0 0 2.5px rgb(var(--c-accent)), 0 6px 16px rgb(0 0 0 / 0.18); }
.flow-link-node.dashed { outline: 1px dashed rgb(0 0 0 / 0.35); outline-offset: -1px; }
.flow-link-node :deep(svg) { width: 15px; height: 15px; opacity: 0.85; }
.link-dirty {
  position: absolute; top: -4px; right: -4px; width: 9px; height: 9px; border-radius: 50%;
  background: rgb(var(--c-accent)); box-shadow: 0 0 0 2px rgb(var(--c-bg));
}
.link-label {
  position: absolute; top: calc(100% + 4px); left: 50%; transform: translateX(-50%);
  font-size: 11px; white-space: nowrap; color: rgb(var(--c-text-dim));
}
</style>

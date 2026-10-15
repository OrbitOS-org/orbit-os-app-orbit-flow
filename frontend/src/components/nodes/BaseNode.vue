<script setup lang="ts">
import { computed, ref, useSlots } from "vue";
import { Handle, Position } from "@vue-flow/core";
import NodeIcon from "@/components/nodes/NodeIcon.vue";
import AppTooltip from "@/components/ui/AppTooltip.vue";
import { nodeFill, NODE_INK, STATUS_COLORS } from "@/components/nodes/tokens";
import { useFlowStore } from "@/stores/flowStore";
import { useUiStore } from "@/stores/uiStore";

export interface ActionButton {
    /** Label displayed vertically on the button. */
    label: string;
    /** Tooltip text. */
    title?: string;
}

export interface BaseNodeProps {
    id: string;
    label?: string;
    nodeType?: string;
    inputs?: number;
    outputs?: number;
    /** Optional per-output tooltips. Index matches the port index. */
    outputLabels?: string[];
    selected?: boolean;
    disabled?: boolean;
    actionButton?: ActionButton | null;
    toggleButton?: boolean;
    toggleState?: boolean;
    status?: {
        fill?: "red" | "green" | "yellow" | "blue" | "grey";
        shape?: "ring" | "dot";
        text?: string;
    } | null;
}

const props = withDefaults(defineProps<BaseNodeProps>(), {
    label: "",
    nodeType: "unknown",
    inputs: 0,
    outputs: 0,
    selected: false,
    disabled: false,
    actionButton: null,
    toggleButton: false,
    toggleState: true,
    status: null,
});

const emit = defineEmits<{
    action: [];
    toggle: [value: boolean];
}>();

const actionActive = ref(false);
function handleAction() {
    actionActive.value = true;
    emit('action');
    setTimeout(() => { actionActive.value = false }, 80);
}

function handleToggle() {
    emit('toggle', !props.toggleState);
}

// Stop Flows: the Inject and Debug buttons are off, as in Node-RED.
const ui = useUiStore();
const flowStore = useFlowStore();
const slots = useSlots();

const fill = computed(() => nodeFill(props.nodeType));
const typeLabel = computed(() => {
    const labels: Record<string, string> = {
        inject: "Inject",
        debug: "Debug",
        function: "Function",
        "context-watch": "Context Watch",
        change: "Change",
        switch: "Switch",
        template: "Template",
        json: "JSON",
        delay: "Delay",
        filter: "Filter",
        comment: "Comment",
        "link-in": "Link In",
        "link-out": "Link Out",
        "link-call": "Link Call",
        "mqtt-in": "MQTT Subscribe",
        "mqtt-out": "MQTT Publish",
        "mqtt-request": "MQTT Request",
        "modbus-read": "Modbus Read",
        "modbus-write": "Modbus Write",
        "modbus-parser": "Modbus Parser",
        "s7-read": "S7 Read",
        "s7-write": "S7 Write",
        "s7-parser": "S7 Parser",
        "opcua-read": "OPC UA Read",
        "opcua-subscribe": "OPC UA Subscribe",
        "opcua-write": "OPC UA Write",
        statemachine: "State Machine",
        "file-read": "Read File",
        "file-out": "Write File",
        "file-watch": "Watch",
        "mobile-notify": "Mobile Notification",
        "gpio-read": "GPIO Read",
        "gpio-write": "GPIO Write",
        "system-metrics": "System Metrics",
        "device-identity": "Device Info",
        "i2c-transfer": "I2C Transfer",
        "spi-transfer": "SPI Transfer",
        "serial-in": "Serial In",
        "serial-out": "Serial Out",
        "serial-request": "Serial Request",
    };
    return labels[props.nodeType] ?? props.nodeType;
});
const hasCustomLabel = computed(() => {
    if (!props.label) return false;
    const l = props.label.toLowerCase();
    return (
        l !== props.nodeType.toLowerCase() &&
        l !== typeLabel.value.toLowerCase()
    );
});
const isDirty = computed(() => flowStore.isNodeDirty(props.id));

const isHighlighted = computed(
    () => flowStore.hoveredHighlightNodeId === props.id && !props.selected,
);

const statusColor = computed(() =>
    STATUS_COLORS[props.status?.fill ?? ''] ?? STATUS_COLORS.grey
);

// Node-RED geometry: one row is 34 px high; each extra port adds 22 px.
const ROW = 34;
const PORT_STEP = 22;

const maxPorts = computed(() => Math.max(props.inputs, props.outputs, 1));
const hasSubline = computed(() => !hasCustomLabel.value && !!slots.body);
const nodeHeight = computed(() =>
    Math.max(hasSubline.value ? 44 : ROW, (maxPorts.value - 1) * PORT_STEP + ROW),
);

function portTop(i: number, count: number): string {
    // Spread the ports evenly around the middle of the node.
    const span = (count - 1) * PORT_STEP;
    return `${nodeHeight.value / 2 - span / 2 + i * PORT_STEP}px`;
}

const inputHandles = computed(() =>
    Array.from({ length: props.inputs }, (_, i) => ({
        id: `input-${i}`,
        style: { top: portTop(i, props.inputs) },
    })),
);

const outputHandles = computed(() =>
    Array.from({ length: props.outputs }, (_, i) => ({
        id: `output-${i}`,
        title: props.outputLabels?.[i] ?? "",
        style: { top: portTop(i, props.outputs) },
    })),
);
</script>

<template>
  <div class="relative" :class="{ 'opacity-45': props.disabled }">
    <!-- Action button (inject): a tab on the left, as in Node-RED -->
    <button
      v-if="props.actionButton"
      class="node-tab left nopan nodrag"
      :class="{ pressed: actionActive }"
      :style="{ background: fill }"
      :disabled="props.disabled || ui.flowsStopped"
      :title="props.actionButton.title ?? props.actionButton.label"
      :aria-label="props.actionButton.title ?? props.actionButton.label"
      @click.stop="handleAction"
      @dblclick.stop
      @mousedown.stop
      @pointerdown.stop
    ><span /></button>

    <!-- Main node body -->
    <div
      class="flow-node"
      :class="{ selected: props.selected, highlighted: isHighlighted, dashed: props.disabled }"
      :style="{ background: fill, color: NODE_INK, height: nodeHeight + 'px' }"
    >
      <Handle
        v-for="h in inputHandles"
        :key="h.id"
        :id="h.id"
        type="target"
        :position="Position.Left"
        :style="h.style"
        class="node-port"
      />
      <AppTooltip
        v-for="h in outputHandles"
        :key="h.id"
        :text="h.title"
        side="right"
      >
        <Handle
          :id="h.id"
          type="source"
          :position="Position.Right"
          :style="h.style"
          class="node-port"
        />
      </AppTooltip>

      <div class="node-icon">
        <slot name="icon"><NodeIcon :type="props.nodeType" /></slot>
      </div>

      <div class="node-text">
        <div class="node-label">{{ hasCustomLabel ? props.label : typeLabel }}</div>
        <div v-if="hasSubline" class="node-sub"><slot name="body" /></div>
        <div v-if="$slots.actions" class="pt-0.5"><slot name="actions" /></div>
      </div>
      <slot name="badge" />

      <span v-if="isDirty" class="node-dirty" title="Undeployed changes" />
    </div>

    <!-- Toggle button (debug): a tab on the right -->
    <button
      v-if="props.toggleButton"
      class="node-tab right nopan nodrag"
      :class="{ off: !props.toggleState }"
      :style="{ background: props.toggleState ? fill : undefined }"
      :disabled="props.disabled || ui.flowsStopped"
      :title="props.toggleState ? 'On — click to turn off' : 'Off — click to turn on'"
      :aria-label="props.toggleState ? 'Turn off' : 'Turn on'"
      @click.stop="handleToggle"
      @dblclick.stop
      @mousedown.stop
      @pointerdown.stop
    ><span /></button>

    <!-- Status, under the node as in Node-RED. Hidden for disabled nodes. -->
    <div
      v-if="props.status && props.status.fill && !props.disabled"
      class="node-status"
    >
      <span
        class="mark"
        :class="props.status.shape === 'ring' ? 'ring' : ''"
        :style="props.status.shape === 'ring' ? { borderColor: statusColor } : { background: statusColor }"
      />
      <span>{{ props.status.text ?? "" }}</span>
    </div>
  </div>
</template>

<style scoped>
.flow-node {
    position: relative;
    display: flex;
    align-items: stretch;
    min-width: 140px;
    border-radius: 8px;
    box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.16), 0 1px 2px rgb(0 0 0 / 0.08);
    transition: box-shadow 0.15s;
    font-size: 13px;
    font-weight: 500;
    user-select: none;
}
.flow-node.dashed { box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.4); outline: 1px dashed rgb(0 0 0 / 0.35); outline-offset: -1px; }
.flow-node.selected { box-shadow: 0 0 0 2.5px rgb(var(--c-accent)), 0 6px 16px rgb(0 0 0 / 0.18); }
.flow-node.highlighted { outline: 2px dashed rgb(var(--c-accent)); outline-offset: 2px; }
.node-icon {
    width: 30px;
    flex: none;
    display: grid;
    place-items: center;
    background: rgb(0 0 0 / 0.07);
    border-right: 1px solid rgb(0 0 0 / 0.1);
    border-radius: 8px 0 0 8px;
    opacity: 0.85;
}
.node-icon :deep(svg) { width: 15px; height: 15px; }
.node-text { flex: 1; min-width: 0; display: flex; flex-direction: column; justify-content: center; padding: 4px 12px 4px 10px; }
.node-label { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; line-height: 1.25; }
.node-sub { font-size: 11px; font-weight: 400; opacity: 0.66; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; line-height: 1.3; }
.node-dirty {
    position: absolute; top: -4px; right: -4px; width: 9px; height: 9px; border-radius: 50%;
    background: rgb(var(--c-accent)); box-shadow: 0 0 0 2px rgb(var(--c-bg));
}
.node-tab {
    position: absolute; top: 50%; transform: translateY(-50%);
    width: 18px; height: 24px; padding: 0; border: 0; cursor: pointer;
    box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.2);
    display: grid; place-items: center;
}
.node-tab.left { left: -22px; border-radius: 6px 3px 3px 6px; }
.node-tab.right { right: -22px; border-radius: 3px 6px 6px 3px; }
.node-tab span { width: 8px; height: 8px; border-radius: 2px; background: rgb(0 0 0 / 0.25); }
.node-tab.pressed { filter: brightness(0.85); }
.node-tab.off { background: rgb(var(--c-surface)); }
.node-tab.off span { background: rgb(var(--c-text-dim) / 0.5); }
.node-tab:disabled { pointer-events: none; }
.node-status {
    position: absolute; left: 4px; top: calc(100% + 5px);
    display: flex; align-items: center; gap: 5px;
    font-size: 11px; color: rgb(var(--c-text-dim)); white-space: nowrap;
}
.node-status .mark { width: 8px; height: 8px; border-radius: 2px; flex: none; }
.node-status .mark.ring { background: transparent; border: 2px solid; border-radius: 50%; }
</style>

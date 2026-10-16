<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@/composables/useApi'
import { useAuthStore } from '@/stores/authStore'
import type { NodeCatalogEntry } from '@/types/flow'
import NodeIcon from '@/components/nodes/NodeIcon.vue'
import { nodeFill, NODE_INK } from '@/components/nodes/tokens'

const api = useApi()
const auth = useAuthStore()

const nodes = ref<NodeCatalogEntry[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const filterText = ref('')
const expandedCategories = ref<Set<string>>(new Set(['common']))

onMounted(async () => {
  try {
    nodes.value = await api.getNodes()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load nodes'
  } finally {
    loading.value = false
  }
})

const filteredNodes = computed(() => {
  const q = filterText.value.trim().toLowerCase()
  if (!q) return nodes.value
  return nodes.value.filter(n =>
    n.label.toLowerCase().includes(q) || n.type.toLowerCase().includes(q) ||
    (n.keywords ?? []).some(k => k.includes(q))
  )
})

const CATEGORY_ORDER = ['common', 'function', 'sequence', 'network', 'industrial', 'hardware', 'device', 'cloud', 'parser', 'storage', 'other']

// Names shown in the palette; the key is the node's category.
const CATEGORY_LABEL: Record<string, string> = { cloud: 'Cloud Services' }

const categories = computed(() => {
  const map = new Map<string, NodeCatalogEntry[]>()
  for (const node of filteredNodes.value) {
    const cat = node.category ?? 'other'
    if (!map.has(cat)) map.set(cat, [])
    map.get(cat)!.push(node)
  }
  for (const [, catNodes] of map) {
    catNodes.sort((a, b) => a.label.localeCompare(b.label))
  }
  const ordered: { name: string; nodes: NodeCatalogEntry[] }[] = []
  for (const cat of CATEGORY_ORDER) {
    if (map.has(cat)) {
      ordered.push({ name: cat, nodes: map.get(cat)! })
      map.delete(cat)
    }
  }
  for (const [name, catNodes] of map) {
    ordered.push({ name, nodes: catNodes })
  }
  return ordered
})

function toggleCategory(name: string): void {
  if (expandedCategories.value.has(name)) {
    expandedCategories.value.delete(name)
  } else {
    expandedCategories.value.add(name)
  }
}

function onDragStart(event: DragEvent, node: NodeCatalogEntry): void {
  if (!auth.can('deploy')) {
    event.preventDefault()
    return
  }
  if (!event.dataTransfer) return
  event.dataTransfer.setData(
    'application/flow-node',
    JSON.stringify({
      type: node.type,
      inputs: node.inputs,
      outputs: node.outputs,
      defaults: node.defaults,
    })
  )
  event.dataTransfer.effectAllowed = 'move'
}
</script>

<template>
  <aside class="palette h-full border-r border-terminal-border flex flex-col overflow-hidden select-none">
    <!-- Search, iOS style -->
    <div class="px-2.5 pt-2.5 pb-1.5 shrink-0">
      <label class="ios-search">
        <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.2" stroke-linecap="round">
          <circle cx="11" cy="11" r="6.5" /><path d="m20 20-4.2-4.2" />
        </svg>
        <input v-model="filterText" type="text" placeholder="Search nodes" aria-label="Search nodes" />
      </label>
    </div>

    <!-- Body -->
    <div class="flex-1 overflow-y-auto">
      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center h-16 text-terminal-text-dim text-xs">
        loading...
      </div>

      <!-- Error -->
      <div v-else-if="error" class="px-3 py-3 text-xs text-status-error">
        {{ error }}
      </div>

      <!-- Empty -->
      <div v-else-if="categories.length === 0" class="px-3 py-3 text-xs text-terminal-text-dim">
        No nodes found.
      </div>

      <!-- Categories -->
      <template v-else>
        <div
          v-for="cat in categories"
          :key="cat.name"
        >
          <!-- Category Header -->
          <button
            class="palette-cat"
            :aria-expanded="expandedCategories.has(cat.name) || !!filterText"
            @click="toggleCategory(cat.name)"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="w-2.5 h-2.5 transition-transform duration-150 shrink-0"
              :class="expandedCategories.has(cat.name) || filterText ? 'rotate-90' : ''"
              fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
            </svg>
            <span>{{ CATEGORY_LABEL[cat.name] ?? cat.name }}</span>
            <span class="count">{{ cat.nodes.length }}</span>
          </button>

          <!-- Nodes -->
          <div v-show="expandedCategories.has(cat.name) || filterText" class="px-2.5 pb-2 flex flex-col gap-[5px]">
            <div
              v-for="node in cat.nodes"
              :key="node.type"
              :draggable="auth.can('deploy')"
              :class="auth.can('deploy')
                ? 'cursor-grab active:cursor-grabbing group'
                : 'cursor-default group opacity-70'"
              :title="auth.can('deploy') ? node.description : 'Read-only — sign in as editor or admin to add nodes'"
              @dragstart="onDragStart($event, node)"
            >
              <!-- The node as it looks in the workspace. Nodes with inputs
                   only carry the icon on the right, as in Node-RED. -->
              <div
                class="palette-node"
                :class="{ right: node.inputs > 0 && node.outputs === 0 }"
                :style="{ background: nodeFill(node.type), color: NODE_INK }"
              >
                <div class="ic"><NodeIcon :type="node.type" /></div>
                <div class="lb">{{ node.label }}</div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>

  </aside>
</template>

<style scoped>
.palette { background: rgb(var(--c-surface)); }
.ios-search {
  display: flex; align-items: center; gap: 6px;
  background: rgb(var(--c-surface-alt) / 0.8); border-radius: 10px; padding: 6px 9px;
  color: rgb(var(--c-text-dim));
}
.ios-search input {
  flex: 1; min-width: 0; border: 0; outline: none; background: transparent;
  font-size: 13px; color: rgb(var(--c-text));
}
.ios-search input::placeholder { color: rgb(var(--c-text-dim) / 0.8); }
.ios-search:focus-within { box-shadow: 0 0 0 2px rgb(var(--c-accent) / 0.5); }
.palette-cat {
  width: 100%; display: flex; align-items: center; gap: 6px;
  padding: 10px 12px 6px; font-size: 11px; font-weight: 600; letter-spacing: 0.04em;
  text-transform: uppercase; color: rgb(var(--c-text-dim)); cursor: pointer;
}
.palette-cat:hover { color: rgb(var(--c-text)); }
.palette-cat .count { margin-left: auto; font-weight: 500; opacity: 0.6; font-variant-numeric: tabular-nums; }
.palette-node {
  display: flex; align-items: stretch; min-height: 30px; border-radius: 7px;
  box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.14); font-size: 12.5px; font-weight: 500;
  transition: transform 0.1s, box-shadow 0.1s;
}
.group:hover .palette-node { transform: translateX(2px); box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.2), 0 2px 6px rgb(0 0 0 / 0.1); }
.palette-node .ic {
  width: 28px; flex: none; display: grid; place-items: center;
  background: rgb(0 0 0 / 0.07); border-right: 1px solid rgb(0 0 0 / 0.1); border-radius: 7px 0 0 7px; opacity: 0.85;
}
.palette-node .ic :deep(svg) { width: 14px; height: 14px; }
.palette-node.right .ic { order: 2; border-right: 0; border-left: 1px solid rgb(0 0 0 / 0.1); border-radius: 0 7px 7px 0; }
.palette-node .lb { flex: 1; min-width: 0; padding: 5px 10px; align-self: center; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
</style>

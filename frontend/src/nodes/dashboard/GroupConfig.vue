<script setup lang="ts">
// Dashboard Group (ui-group): a titled section of a page.
import { computed, onMounted, ref } from 'vue'
import { newId } from '@/lib/id'
import { useFlowStore } from '@/stores/flowStore'
import { useUiStore } from '@/stores/uiStore'
import { useConfigSelector } from '@/composables/useConfigSelector'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ConfigFooter from './ConfigFooter.vue'

const props = defineProps<{ configId?: string }>()
const flowStore = useFlowStore()
const ui = useUiStore()
const { options: pageOptions, openNewConfig } = useConfigSelector('ui-page')

const name = ref('')
const page = ref('')
const width = ref(12)
const order = ref(0)
const editing = ref(false)

onMounted(() => {
  const existing = props.configId ? flowStore.configs.find((c) => c.id === props.configId) : undefined
  if (existing) {
    editing.value = true
    name.value = existing.name ?? ''
    const cfg = (existing.config ?? {}) as Record<string, unknown>
    page.value = String(cfg.page ?? '')
    width.value = Number(cfg.width ?? 12)
    order.value = Number(cfg.order ?? 0)
  } else {
    page.value = String(pageOptions.value[0]?.value ?? '')
    order.value = flowStore.getConfigsByType('ui-group').length + 1
  }
})

function save() {
  const config = { page: page.value, width: width.value, order: order.value }
  if (editing.value && props.configId) flowStore.updateConfig(props.configId, { name: name.value, config })
  else flowStore.addConfig({ id: newId(), type: 'ui-group', name: name.value, config })
  ui.clearConfigEditor()
}
const canSave = computed(() => !!name.value.trim())
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-3">
      <span class="text-accent text-sm font-bold uppercase tracking-wider">{{ editing ? 'Edit' : 'New' }} Dashboard Group</span>
      <FormField label="Name" :error="!canSave ? 'Name required' : ''">
        <FormInput v-model="name" placeholder="e.g. Status" :invalid="!canSave" />
      </FormField>
      <FormField label="Page">
        <template #action>
          <button class="text-[9px] uppercase tracking-wider text-terminal-text-dim hover:text-accent transition-colors" @click="openNewConfig()">+ New</button>
        </template>
        <FormSelect v-model="page" :options="pageOptions" placeholder="Select a page…" width="100%" />
      </FormField>
      <FormField label="Width" hint="Columns of the page, out of 12 (on a phone every group is full width).">
        <NumberInput v-model="width" :min="1" :max="12" :step="1" unit="of 12" />
      </FormField>
      <FormField label="Order" hint="Groups are placed from the lowest number.">
        <NumberInput v-model="order" :min="0" :step="1" />
      </FormField>
    </div>
    <ConfigFooter :editing="editing" :can-save="canSave" @save="save" @cancel="ui.clearConfigEditor()" />
  </div>
</template>

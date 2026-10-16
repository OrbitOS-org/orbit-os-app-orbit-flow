<script setup lang="ts">
// Dashboard Page (ui-page): a tab of the dashboard.
import { computed, onMounted, ref } from 'vue'
import { newId } from '@/lib/id'
import { useFlowStore } from '@/stores/flowStore'
import { useUiStore } from '@/stores/uiStore'
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ConfigFooter from './ConfigFooter.vue'

const props = defineProps<{ configId?: string }>()
const flowStore = useFlowStore()
const ui = useUiStore()

const name = ref('')
const order = ref(0)
const editing = ref(false)

onMounted(() => {
  const existing = props.configId ? flowStore.configs.find((c) => c.id === props.configId) : undefined
  if (existing) {
    editing.value = true
    name.value = existing.name ?? ''
    order.value = Number((existing.config as Record<string, unknown>)?.order ?? 0)
  } else {
    order.value = flowStore.getConfigsByType('ui-page').length + 1
  }
})

function save() {
  const config = { order: order.value }
  if (editing.value && props.configId) flowStore.updateConfig(props.configId, { name: name.value, config })
  else flowStore.addConfig({ id: newId(), type: 'ui-page', name: name.value, config })
  ui.clearConfigEditor()
}
const canSave = computed(() => !!name.value.trim())
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-3">
      <span class="text-accent text-sm font-bold uppercase tracking-wider">{{ editing ? 'Edit' : 'New' }} Dashboard Page</span>
      <FormField label="Name" :error="!canSave ? 'Name required' : ''">
        <FormInput v-model="name" placeholder="e.g. Garage" :invalid="!canSave" />
      </FormField>
      <FormField label="Order" hint="Pages are shown as tabs, from the lowest number.">
        <NumberInput v-model="order" :min="0" :step="1" />
      </FormField>
    </div>
    <ConfigFooter :editing="editing" :can-save="canSave" @save="save" @cancel="ui.clearConfigEditor()" />
  </div>
</template>

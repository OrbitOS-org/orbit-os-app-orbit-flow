<script setup lang="ts">
import { SwitchRoot, SwitchThumb } from 'radix-vue'

withDefaults(defineProps<{
  modelValue: boolean
  label?: string
}>(), {
  label: '',
})

defineEmits<{
  'update:modelValue': [value: boolean]
}>()
</script>

<template>
  <label class="flex items-center gap-2 cursor-pointer group">
    <SwitchRoot
      :checked="modelValue"
      class="w-[38px] h-[23px] shrink-0 rounded-full relative transition-colors duration-200
             bg-terminal-surface-alt
             data-[state=checked]:bg-status-success"
      @update:checked="$emit('update:modelValue', $event)"
    >
      <SwitchThumb
        class="block w-[19px] h-[19px] rounded-full bg-white shadow-md transition-transform duration-200
               translate-x-[2px] data-[state=checked]:translate-x-[17px]"
      />
    </SwitchRoot>
    <span v-if="label" class="text-xs text-terminal-text group-hover:text-terminal-text-bright transition-colors">
      {{ label }}
    </span>
  </label>
</template>

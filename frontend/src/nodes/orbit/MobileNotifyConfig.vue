<script setup lang="ts">
import FormField from '@/components/ui/FormField.vue'
import FormInput from '@/components/ui/FormInput.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'

const body = useNodeProperty<string>('body', '')
const imageUrl = useNodeProperty<string>('imageUrl', '')
const collapseKey = useNodeProperty<string>('collapseKey', '')
const ttlSeconds = useNodeProperty<number>('ttlSeconds', 0)

// Workaround for Vue's Mustache-style interpolation: rendering a literal
// "{{...}}" string inline (rather than through a named constant like these)
// closes the outer {{ }} prematurely — Vue's tokenizer scans for the next
// raw "}}" without parsing the JS expression, so a string literal containing
// "}}" ends the interpolation early.
const MUSTACHE_TOKEN = '{{mustache}}'
const MUSTACHE_PAYLOAD = '{{payload}}'
const MUSTACHE_TOPIC = '{{topic}}'
</script>

<template>
  <div class="flex flex-col gap-3 flex-1 min-h-0">
    <div class="text-[10px] text-text-muted leading-tight">
      Sends a push notification to the OrbitOS device owner's phone. The
      title is set by the runtime ("&lt;device&gt; · Orbit Flow") — this node
      only controls the body below.
    </div>

    <FormField label="Body (optional)">
      <FormInput
        v-model="body"
        placeholder="{{payload}} went offline"
        mono
      />
      <div v-if="!body" class="text-[10px] text-text-muted leading-tight">
        Leave empty to send msg.payload as the body.
      </div>
      <div v-else class="text-[10px] text-text-muted leading-tight">
        {{ MUSTACHE_TOKEN }} supported, rendered against the message (e.g.
        {{ MUSTACHE_PAYLOAD }}, {{ MUSTACHE_TOPIC }}).
      </div>
    </FormField>

    <FormField label="Image URL (optional)">
      <FormInput v-model="imageUrl" placeholder="https://example.com/alert.png" mono />
    </FormField>

    <FormField
      label="Collapse key (optional)"
      hint="Notifications sharing a key replace each other instead of stacking"
    >
      <FormInput v-model="collapseKey" placeholder="e.g. modbus-offline" mono />
    </FormField>

    <FormField
      label="TTL"
      hint="How long FCM holds the notification if the phone is offline (0 = provider default)"
    >
      <NumberInput v-model="ttlSeconds" :min="0" unit="sec" />
    </FormField>
  </div>
</template>

<script setup lang="ts">
// PWM Out: a PWM channel of the device, the duty cycle from msg.payload.
import { computed, onMounted, ref } from 'vue'
import FormField from '@/components/ui/FormField.vue'
import FormSelect from '@/components/ui/FormSelect.vue'
import NumberInput from '@/components/ui/NumberInput.vue'
import ToggleGroup from '@/components/ui/ToggleGroup.vue'
import { useNodeProperty } from '@/composables/useNodeProperty'
import { useApi, type PwmChannelInfo } from '@/composables/useApi'

const api = useApi()
const channel = useNodeProperty<number>('channel', -1)
const channelName = useNodeProperty<string>('channelName', '')
const frequency = useNodeProperty<number>('frequency', 1000)
const scale = useNodeProperty<string>('scale', 'percent')
const onStop = useNodeProperty<string>('onStop', 'off')
const stopValue = useNodeProperty<number>('stopValue', 0)

const detected = ref<PwmChannelInfo[]>([])
const loaded = ref(false)
const manual = ref(false)

onMounted(async () => {
  try {
    detected.value = await api.listPwmChannels()
  } catch {
    detected.value = []
  }
  loaded.value = true
  manual.value = detected.value.length === 0
})

const channelOptions = computed(() =>
  detected.value.map((c) => ({ value: String(c.channel), label: c.name ? `${c.name} (channel ${c.channel})` : `channel ${c.channel}` })))

const selected = computed({
  get: () => (Number(channel.value) >= 0 ? String(channel.value) : ''),
  set: (v: string) => {
    const c = detected.value.find((x) => String(x.channel) === v)
    channel.value = Number(v)
    channelName.value = c?.name ?? ''
  },
})

const SCALES = [
  { value: 'percent', label: '0–100 %' },
  { value: 'unit', label: '0–1' },
  { value: 'byte', label: '0–255' },
]
const ON_STOP = [
  { value: 'off', label: 'Off' },
  { value: 'value', label: 'Value' },
  { value: 'keep', label: 'Leave as is' },
]
const scaleMax = computed(() => ({ percent: 100, unit: 1, byte: 255 }[scale.value] ?? 100))
</script>

<template>
  <div class="flex flex-col gap-3">
    <FormField label="Channel" :error="Number(channel) < 0 ? 'Choose a channel' : ''">
      <template v-if="detected.length > 0" #action>
        <button
          type="button"
          class="text-[9px] text-terminal-text-dim hover:text-accent underline underline-offset-2"
          @click="manual = !manual"
        >{{ manual ? 'Choose from the device' : 'Type manually' }}</button>
      </template>
      <FormSelect v-if="!manual" v-model="selected" :options="channelOptions" placeholder="Select a channel…" width="100%" />
      <NumberInput v-else :model-value="Math.max(0, Number(channel))" :min="0" :step="1" @update:model-value="channel = Number($event); channelName = ''" />
      <p v-if="loaded && detected.length === 0" class="text-[10px] text-terminal-text-dim leading-relaxed">
        This device reports no PWM channels. A Raspberry Pi has them (pwm0, pwm1); other boards may not.
        You can still type a channel number for another device.
      </p>
    </FormField>

    <FormField label="Frequency" hint="Fans usually 25 kHz, LEDs 1 kHz, servos 50 Hz. msg.frequency (Hz) changes it.">
      <NumberInput :model-value="frequency" :min="1" :step="100" unit="Hz" @update:model-value="frequency = Number($event)" />
    </FormField>

    <FormField label="msg.payload range">
      <ToggleGroup v-model="scale" :options="SCALES" />
    </FormField>

    <FormField
      label="When flows stop"
      hint="Also when this node is deleted or its flow is disabled. A deploy that keeps the node leaves the output as it is."
    >
      <ToggleGroup v-model="onStop" :options="ON_STOP" />
      <NumberInput
        v-if="onStop === 'value'"
        :model-value="stopValue"
        :min="0"
        :max="scaleMax"
        :step="scale === 'unit' ? 0.05 : 1"
        @update:model-value="stopValue = Number($event)"
      />
    </FormField>

    <p class="text-[10px] text-terminal-text-dim leading-relaxed">
      <code class="text-terminal-text">msg.payload</code> sets the duty cycle: a number in the range above,
      <code class="text-terminal-text">"40%"</code>, or true/false for full on/off. Values outside the range are limited.
    </p>
  </div>
</template>

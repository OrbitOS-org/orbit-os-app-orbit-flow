import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type InfoTab = 'help' | 'config' | 'context' | 'state-machines' | 'debug'
export type ConnectionStatus = 'connected' | 'disconnected' | 'connecting'
export type DeployStatus = 'idle' | 'deploying' | 'deployed' | 'failed'
export type LogsLimit = 100 | 200 | 500 | 1000

const ALLOWED_LOGS_LIMITS = [100, 200, 500, 1000] as const

function loadLogsLimit(): LogsLimit {
  const raw = parseInt(localStorage.getItem('orbit-flow-logs-limit') ?? '200', 10)
  return (ALLOWED_LOGS_LIMITS as readonly number[]).includes(raw) ? (raw as LogsLimit) : 200
}
export type PropertiesContext =
  | { type: 'node' }
  | { type: 'flow-create' }
  | { type: 'flow-edit'; flowId: string }
  // assignTo: the node property that gets a connection created here ("+ New"
  // on a node), so the node uses it at once, as in Node-RED.
  | { type: 'config-edit'; configType: string; configId?: string; assignTo?: { nodeId: string; key: string } }
  | null

export const useUiStore = defineStore('ui', () => {
  // ── State ──────────────────────────────────────────────────────────
  const leftPanelOpen = ref<boolean>(true)
  const propertiesPanelOpen = ref<boolean>(false)
  const propertiesPanelWidth = ref<number>(450)
  const infoPanelOpen = ref<boolean>(true)
  const infoPanelWidth = ref<number>(320)
  const activeInfoTab = ref<InfoTab>('debug')
  const contextAutoRefresh = ref<boolean>(
    localStorage.getItem('orbit-flow-context-auto-refresh') === 'true',
  )
  const stateMachineAutoRefresh = ref<boolean>(
    localStorage.getItem('orbit-flow-state-machine-auto-refresh') === 'true',
  )
  const selectedStateMachineFlowID = ref<string | null>(null)
  const selectedStateMachineNodeID = ref<string | null>(null)
  const connectionStatus = ref<ConnectionStatus>('disconnected')
  const deployStatus = ref<DeployStatus>('idle')
  // Why the last deploy failed, when the server said (e.g. a serial port
  // used by two connections); shown in the banner.
  const deployError = ref<string | null>(null)
  // Stop Flows (Deploy menu), as in Node-RED: no flow runs on the device
  // until Start Flows. Read from the device; kept there across restarts.
  const flowsStopped = ref(false)
  // A short message in the banner under the header ("Flows stopped…").
  const notice = ref<{ text: string; tone: 'ok' | 'warn' } | null>(null)
  let noticeTimer: ReturnType<typeof setTimeout> | null = null
  function showNotice(text: string, tone: 'ok' | 'warn' = 'ok') {
    if (noticeTimer) clearTimeout(noticeTimer)
    notice.value = { text, tone }
    noticeTimer = setTimeout(() => { notice.value = null; noticeTimer = null }, 3500)
  }
  function setFlowsStopped(v: boolean) {
    flowsStopped.value = v
  }
  const propertiesContext = ref<PropertiesContext>(null)
  const logsPanelOpen = ref<boolean>(false)
  const logsLimit = ref<LogsLimit>(loadLogsLimit())
  let deployResetTimer: ReturnType<typeof setTimeout> | null = null

  // ── Getters ────────────────────────────────────────────────────────
  const isConnected = computed(() => connectionStatus.value === 'connected')
  const isConnecting = computed(() => connectionStatus.value === 'connecting')

  const connectionStatusColor = computed(() => {
    switch (connectionStatus.value) {
      case 'connected':    return '#4ade80'
      case 'connecting':   return '#FFBF00'
      case 'disconnected': return '#ef4444'
    }
  })

  // ── Actions ────────────────────────────────────────────────────────
  function toggleLeftPanel() {
    leftPanelOpen.value = !leftPanelOpen.value
  }

  function togglePropertiesPanel() {
    propertiesPanelOpen.value = !propertiesPanelOpen.value
  }

  function openPropertiesPanel() {
    propertiesPanelOpen.value = true
  }

  function closePropertiesPanel() {
    propertiesPanelOpen.value = false
    propertiesContext.value = null
  }

  function openFlowCreateProperties() {
    propertiesContext.value = { type: 'flow-create' }
    propertiesPanelOpen.value = true
  }

  function openFlowEditProperties(flowId: string) {
    propertiesContext.value = { type: 'flow-edit', flowId }
    propertiesPanelOpen.value = true
  }

  function clearFlowProperties() {
    propertiesContext.value = null
  }

  function openConfigEditor(configType: string, configId?: string, assignTo?: { nodeId: string; key: string }) {
    propertiesContext.value = { type: 'config-edit', configType, configId, assignTo }
    propertiesPanelOpen.value = true
  }

  function clearConfigEditor() {
    propertiesContext.value = null
  }

  function toggleInfoPanel() {
    infoPanelOpen.value = !infoPanelOpen.value
  }

  function openInfoPanel() {
    infoPanelOpen.value = true
  }

  function closeInfoPanel() {
    infoPanelOpen.value = false
  }

  function setInfoTab(tab: InfoTab) {
    activeInfoTab.value = tab
  }

  function setContextAutoRefresh(v: boolean) {
    contextAutoRefresh.value = v
    localStorage.setItem('orbit-flow-context-auto-refresh', v ? 'true' : 'false')
  }

  function setStateMachineAutoRefresh(v: boolean) {
    stateMachineAutoRefresh.value = v
    localStorage.setItem('orbit-flow-state-machine-auto-refresh', v ? 'true' : 'false')
  }

  function setSelectedStateMachineNodeID(id: string | null) {
    selectedStateMachineNodeID.value = id
  }

  function setSelectedStateMachine(flowId: string | null, nodeId: string | null) {
    selectedStateMachineFlowID.value = flowId
    selectedStateMachineNodeID.value = nodeId
  }

  function setConnectionStatus(status: ConnectionStatus) {
    connectionStatus.value = status
  }

  function toggleLogsPanel() {
    logsPanelOpen.value = !logsPanelOpen.value
  }

  function openLogsPanel() {
    logsPanelOpen.value = true
  }

  function closeLogsPanel() {
    logsPanelOpen.value = false
  }

  function setLogsLimit(n: LogsLimit) {
    logsLimit.value = n
    localStorage.setItem('orbit-flow-logs-limit', String(n))
  }

  function setDeployStatus(status: DeployStatus, error: string | null = null) {
    deployError.value = status === 'failed' ? error : null
    if (deployResetTimer !== null) {
      clearTimeout(deployResetTimer)
      deployResetTimer = null
    }
    deployStatus.value = status
    if (status === 'deployed') {
      deployResetTimer = setTimeout(() => {
        deployStatus.value = 'idle'
        deployResetTimer = null
      }, 3000)
    }
  }

  return {
    // state
    leftPanelOpen,
    propertiesPanelOpen,
    propertiesPanelWidth,
    propertiesContext,
    infoPanelOpen,
    infoPanelWidth,
    activeInfoTab,
    contextAutoRefresh,
    stateMachineAutoRefresh,
    selectedStateMachineFlowID,
    selectedStateMachineNodeID,
    connectionStatus,
    deployStatus,
    deployError,
    flowsStopped,
    setFlowsStopped,
    notice,
    showNotice,
    logsPanelOpen,
    logsLimit,

    // getters
    isConnected,
    isConnecting,
    connectionStatusColor,

    // actions
    toggleLeftPanel,
    togglePropertiesPanel,
    openPropertiesPanel,
    closePropertiesPanel,
    openFlowCreateProperties,
    openFlowEditProperties,
    clearFlowProperties,
    openConfigEditor,
    clearConfigEditor,
    toggleInfoPanel,
    openInfoPanel,
    closeInfoPanel,
    setInfoTab,
    setContextAutoRefresh,
    setStateMachineAutoRefresh,
    setSelectedStateMachineNodeID,
    setSelectedStateMachine,
    setConnectionStatus,
    setDeployStatus,
    toggleLogsPanel,
    openLogsPanel,
    closeLogsPanel,
    setLogsLimit,
  }
})

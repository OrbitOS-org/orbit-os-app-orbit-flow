// Node design tokens — one palette per category.
// The values below are the dark theme's: every color is intentionally
// dark/muted to avoid eye strain. The light theme's are derived from them
// (see lightTokens), so a category is still defined in one place.

export interface NodeTokens {
  accent:     string
  accentDim:  string
  accentBdr:  string
  accentGlow: string
  bg:         string
  bgHdr:      string
  bgIcon:     string
  border:     string
  textSub:    string
}

export const TOKENS: Record<string, NodeTokens> = {
  link: {
    accent:     '#9ca3af',
    accentDim:  '#9ca3af12',
    accentBdr:  '#9ca3af2a',
    accentGlow: '#9ca3af12',
    bg:         '#0a0a0b',
    bgHdr:      '#161618',
    bgIcon:     '#1c1c1f',
    border:     '#2e2e33',
    textSub:    '#71717a',
  },
  input: {
    accent:     '#4dff8f',
    accentDim:  '#4dff8f15',
    accentBdr:  '#4dff8f33',
    accentGlow: '#4dff8f14',
    bg:         '#080f09',
    bgHdr:      '#0b1f13',
    bgIcon:     '#0f2e1c',
    border:     '#1e4a26',
    textSub:    '#5aad62',
  },
  process: {
    accent:     '#38b6ff',
    accentDim:  '#38b6ff12',
    accentBdr:  '#38b6ff2a',
    accentGlow: '#38b6ff12',
    bg:         '#060c14',
    bgHdr:      '#0c1a2e',
    bgIcon:     '#101828',
    border:     '#1a3050',
    textSub:    '#5a8ab8',
  },
  context: {
    accent:     '#c084fc',
    accentDim:  '#c084fc12',
    accentBdr:  '#c084fc2a',
    accentGlow: '#c084fc12',
    bg:         '#0a0612',
    bgHdr:      '#160e28',
    bgIcon:     '#1a1030',
    border:     '#2e1a50',
    textSub:    '#9a6abf',
  },
  output: {
    accent:     '#ff6b2b',
    accentDim:  '#ff6b2b12',
    accentBdr:  '#ff6b2b2a',
    accentGlow: '#ff6b2b12',
    bg:         '#090503',
    bgHdr:      '#1a0e08',
    bgIcon:     '#150b06',
    border:     '#4a2010',
    textSub:    '#b8704a',
  },
  error: {
    accent:     '#ef4444',
    accentDim:  '#ef444412',
    accentBdr:  '#ef44442a',
    accentGlow: '#ef444414',
    bg:         '#0e0606',
    bgHdr:      '#1f0c0c',
    bgIcon:     '#2a0f0f',
    border:     '#4a1a1a',
    textSub:    '#b86060',
  },
  // OPC UA palette: cyan/teal — industrial automation cue that reads well
  // on dark surfaces and is clearly distinct from MQTT's magenta family.
  opcua: {
    accent:     '#22d3ee',
    accentDim:  '#22d3ee12',
    accentBdr:  '#22d3ee2a',
    accentGlow: '#22d3ee14',
    bg:         '#04101a',
    bgHdr:      '#0a1f2a',
    bgIcon:     '#0e2630',
    border:     '#155a6b',
    textSub:    '#5a9eb0',
  },
  // HTTP palette: electric indigo — web/internet cue, distinct from
  // the cooler blue family (process/change) and the cyan family (OPC UA).
  http: {
    accent:     '#7c83fd',
    accentDim:  '#7c83fd12',
    accentBdr:  '#7c83fd2a',
    accentGlow: '#7c83fd14',
    bg:         '#070713',
    bgHdr:      '#0e1024',
    bgIcon:     '#13152e',
    border:     '#2a2e5c',
    textSub:    '#7682cc',
  },
  // TCP palette: deep teal — transport-layer cue, deliberately darker
  // and greener than OPC-UA's bright cyan and distinct from HTTP indigo.
  tcp: {
    accent:     '#3da99a',
    accentDim:  '#3da99a12',
    accentBdr:  '#3da99a2a',
    accentGlow: '#3da99a14',
    bg:         '#04120f',
    bgHdr:      '#0a2520',
    bgIcon:     '#0e2d28',
    border:     '#155449',
    textSub:    '#5a9e90',
  },
  // UDP palette: warm amber — counterpart to TCP teal. The warm/cool
  // pairing makes the two transport families immediately distinguishable
  // on the canvas without competing with template's softer peach.
  udp: {
    accent:     '#d6a04b',
    accentDim:  '#d6a04b12',
    accentBdr:  '#d6a04b2a',
    accentGlow: '#d6a04b14',
    bg:         '#100b04',
    bgHdr:      '#241a0a',
    bgIcon:     '#2c200c',
    border:     '#54401a',
    textSub:    '#a8895a',
  },
  // Tech-specific palette: MQTT brand magenta-purple from the mqtt.org logo,
  // brightened so it reads on a dark surface.
  mqtt: {
    accent:     '#c026d3',
    accentDim:  '#c026d312',
    accentBdr:  '#c026d32a',
    accentGlow: '#c026d312',
    bg:         '#0c060e',
    bgHdr:      '#1a0a1c',
    bgIcon:     '#22122a',
    border:     '#3a1a40',
    textSub:    '#a85aaa',
  },
  // OrbitOS platform palette: deep violet — distinct from context's lighter
  // lavender — for nodes that call the device's own OrbitOS SDK managers
  // directly (e.g. mobile-notify) rather than doing generic network/
  // filesystem I/O.
  orbit: {
    accent:     '#8b5cf6',
    accentDim:  '#8b5cf612',
    accentBdr:  '#8b5cf62a',
    accentGlow: '#8b5cf614',
    bg:         '#0e0616',
    bgHdr:      '#1c0a2a',
    bgIcon:     '#241035',
    border:     '#3a1a54',
    textSub:    '#a878c8',
  },
  statemachine: {
    accent:     '#c4af3e',
    accentDim:  '#c4af3e12',
    accentBdr:  '#c4af3e2a',
    accentGlow: '#c4af3e14',
    bg:         '#0c0a05',
    bgHdr:      '#1a1608',
    bgIcon:     '#211c0c',
    border:     '#4a3f12',
    textSub:    '#a8945a',
  },
  function: {
    accent:     '#5fc4a8',
    accentDim:  '#5fc4a812',
    accentBdr:  '#5fc4a82a',
    accentGlow: '#5fc4a814',
    bg:         '#061210',
    bgHdr:      '#0c1f1a',
    bgIcon:     '#102822',
    border:     '#1e4a40',
    textSub:    '#5a9e8a',
  },
  debug: {
    accent:     '#4a8062',
    accentDim:  '#4a806212',
    accentBdr:  '#4a80622a',
    accentGlow: '#4a806214',
    bg:         '#050a07',
    bgHdr:      '#0a1410',
    bgIcon:     '#0e1a14',
    border:     '#1e3a2c',
    textSub:    '#5a8a72',
  },
  inject: {
    accent:     '#9ec559',
    accentDim:  '#9ec55912',
    accentBdr:  '#9ec5592a',
    accentGlow: '#9ec55914',
    bg:         '#0a0e05',
    bgHdr:      '#131a08',
    bgIcon:     '#18200c',
    border:     '#3a4a18',
    textSub:    '#8aa05a',
  },
  change: {
    accent:     '#5b8de0',
    accentDim:  '#5b8de012',
    accentBdr:  '#5b8de02a',
    accentGlow: '#5b8de014',
    bg:         '#050912',
    bgHdr:      '#0a1224',
    bgIcon:     '#10182e',
    border:     '#2a3a5e',
    textSub:    '#6a85b8',
  },
  switch: {
    accent:     '#7a93b3',
    accentDim:  '#7a93b312',
    accentBdr:  '#7a93b32a',
    accentGlow: '#7a93b314',
    bg:         '#080a0d',
    bgHdr:      '#121620',
    bgIcon:     '#181c26',
    border:     '#353e4e',
    textSub:    '#7a8090',
  },
  template: {
    accent:     '#e8a577',
    accentDim:  '#e8a57712',
    accentBdr:  '#e8a5772a',
    accentGlow: '#e8a57714',
    bg:         '#0e0805',
    bgHdr:      '#1c1208',
    bgIcon:     '#22150a',
    border:     '#4a2e1a',
    textSub:    '#b88a6a',
  },
  // Filesystem palette: slate-stone — evokes disk / storage / archival.
  // Cooler and greyer than process/switch; distinct from MQTT magenta and
  // OPC UA cyan. Reads as "data at rest" rather than "data in motion".
  filesystem: {
    accent:     '#94a3b8',
    accentDim:  '#94a3b812',
    accentBdr:  '#94a3b82a',
    accentGlow: '#94a3b814',
    bg:         '#080a0d',
    bgHdr:      '#10141a',
    bgIcon:     '#161a22',
    border:     '#2e3742',
    textSub:    '#6a7585',
  },
  // Iron-oxide rust — warm brown-orange, evokes industrial / fieldbus equipment.
  rust: {
    accent:     '#c97644',
    accentDim:  '#c9764412',
    accentBdr:  '#c976442a',
    accentGlow: '#c9764414',
    bg:         '#0c0805',
    bgHdr:      '#1c0f08',
    bgIcon:     '#22140a',
    border:     '#4a2818',
    textSub:    '#b88a5a',
  },
}

// Node type → category mapping is sourced from the per-group manifests in
// frontend/src/nodes/<group>/index.ts. Each group declares either a default
// `category` for all its node types or per-type `categories`. To change the
// palette colour of a node type, edit the group's manifest — not this file.

import { getCategory } from '@/nodes'
import { useTheme } from '@/composables/useTheme'

export const STATUS_COLORS: Record<string, string> = {
  red: '#e24b4a',
  green: '#4ade80',
  yellow: '#ef9f27',
  blue: '#60a5fa',
  grey: '#6b7280',
}

// ── Light theme ────────────────────────────────────────────────────

type RGB = [number, number, number]

function parseHex(hex: string): RGB {
  const n = parseInt(hex.slice(1, 7), 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

function toHex([r, g, b]: RGB): string {
  return '#' + [r, g, b].map((v) => Math.round(v).toString(16).padStart(2, '0')).join('')
}

function mix(a: RGB, b: RGB, amountOfB: number): RGB {
  return [0, 1, 2].map((i) => a[i] + (b[i] - a[i]) * amountOfB) as RGB
}

/** Perceived brightness, 0 (black) to 1 (white). */
function brightness([r, g, b]: RGB): number {
  return (0.299 * r + 0.587 * g + 0.114 * b) / 255
}

const WHITE: RGB = [255, 255, 255]
const BLACK: RGB = [0, 0, 0]
const INK_DIM: RGB = [96, 100, 125]

/** The same category on a light surface: darker accent, pale tints. */
function lightTokens(dark: NodeTokens): NodeTokens {
  let accent = parseHex(dark.accent)
  // Bright accents (lime, cyan, amber) do not read on white: darken them
  // until they do, keeping the hue.
  const b = brightness(accent)
  if (b > 0.42) accent = mix(accent, BLACK, 1 - 0.42 / b)
  const hex = toHex(accent)
  return {
    accent:     hex,
    accentDim:  hex + '14',
    accentBdr:  hex + '40',
    accentGlow: hex + '26',
    bg:         '#ffffff',
    bgHdr:      toHex(mix(accent, WHITE, 0.93)),
    bgIcon:     toHex(mix(accent, WHITE, 0.86)),
    border:     toHex(mix(accent, WHITE, 0.62)),
    textSub:    toHex(mix(accent, INK_DIM, 0.65)),
  }
}

const LIGHT_TOKENS: Record<string, NodeTokens> = {}

const { theme } = useTheme()

/** Colours of a node type in the current theme. Reactive to the theme. */
export function getTokens(nodeType: string): NodeTokens {
  const category = TOKENS[getCategory(nodeType)] ? getCategory(nodeType) : 'process'
  if (theme.value !== 'light') return TOKENS[category]
  return (LIGHT_TOKENS[category] ??= lightTokens(TOKENS[category]))
}

// ── Node fill colours (Node-RED style) ─────────────────────────────
// Nodes are drawn as a pastel block with dark text, the same in both
// themes, so a node keeps its colour wherever it appears (palette,
// workspace, help). The colours follow Node-RED's where Node-RED has the
// node, so people who know it recognise them.
const NODE_FILL: Record<string, string> = {
  inject: '#a6bbcf',
  debug: '#87a980',
  catch: '#e49191',
  status: '#94c1d0',
  'link-in': '#dddddd', 'link-out': '#dddddd', 'link-call': '#dddddd',
  function: '#fdd0a2', 'function-expr': '#fdd0a2', 'function-go': '#fdd0a2',
  change: '#e2d96e', switch: '#e2d96e',
  template: '#f3b567',
  delay: '#e6e0f8', statemachine: '#e6e0f8', 'context-watch': '#e6e0f8',
  json: '#debd5c', xml: '#debd5c', csv: '#debd5c', 'csv-out': '#debd5c',
  'mqtt-in': '#d8bfd8', 'mqtt-out': '#d8bfd8', 'mqtt-request': '#d8bfd8',
  'http-in': '#e7e7ae', 'http-request': '#e7e7ae', 'http-response': '#e7e7ae',
  'tcp-in': '#c0c0c0', 'tcp-out': '#c0c0c0', 'tcp-request': '#c0c0c0',
  'udp-in': '#c0c0c0', 'udp-out': '#c0c0c0',
  'modbus-read': '#e9967a', 'modbus-write': '#e9967a', 'modbus-parser': '#e9967a',
  'opcua-read': '#9fd4d9', 'opcua-subscribe': '#9fd4d9', 'opcua-write': '#9fd4d9',
  's7-read': '#b5cde8', 's7-write': '#b5cde8', 's7-parser': '#b5cde8',
  'gpio-read': '#c6dbef', 'gpio-write': '#c6dbef',
  'i2c-transfer': '#c6dbef', 'spi-transfer': '#c6dbef',
  'system-metrics': '#cfd4e6', 'device-identity': '#cfd4e6',
  'mobile-notify': '#f0c3d4',
  'file-read': '#deb887', 'file-out': '#deb887', 'file-watch': '#deb887',
}

/** Fill colour of a node type. */
export function nodeFill(nodeType: string): string {
  return NODE_FILL[nodeType] ?? '#d9d9de'
}

/** Text colour on a node fill: always dark, the fills are light. */
export const NODE_INK = '#1c1c1e'

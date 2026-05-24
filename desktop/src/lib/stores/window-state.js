const STORAGE_KEY = 'sarde-ui-state'

const PERSISTED_KEYS = [
  'leftPanel',
  'rightPanel',
  'previewMode',
  'settingsSection',
  'propertiesMode',
]

export function persistUiState(ui) {
  try {
    const state = {}
    for (const key of PERSISTED_KEYS) {
      if (ui[key] !== undefined) state[key] = ui[key]
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {}
}

export function restoreUiState(ui) {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return
    const state = JSON.parse(raw)
    for (const key of PERSISTED_KEYS) {
      if (state[key] !== undefined) {
        ui[key] = state[key]
      }
    }
  } catch {}
}

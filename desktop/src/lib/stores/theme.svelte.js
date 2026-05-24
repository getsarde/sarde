const STORAGE_KEY = 'sarde_theme'

export const THEMES = [
  { id: 'system', name: 'System', category: 'auto' },
  { id: 'dark', name: 'Dark', category: 'dark' },
  { id: 'light', name: 'Light', category: 'light' },
  { id: 'nord', name: 'Nord', category: 'dark' },
  { id: 'dracula', name: 'Dracula', category: 'dark' },
  { id: 'tokyo-night', name: 'Tokyo Night', category: 'dark' },
  { id: 'rose-pine', name: 'Rose Pine', category: 'dark' },
  { id: 'github-dark', name: 'GitHub Dark', category: 'dark' },
  { id: 'one-dark-pro', name: 'One Dark Pro', category: 'dark' },
]

let _currentTheme = $state('system')

export function getCurrentTheme() {
  return _currentTheme
}

export function setTheme(id) {
  _currentTheme = id
  applyTheme(id)
  localStorage.setItem(STORAGE_KEY, id)
}

export function initTheme() {
  const saved = localStorage.getItem(STORAGE_KEY)
  if (saved) {
    _currentTheme = saved
  }
  applyTheme(_currentTheme)

  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (_currentTheme === 'system') {
      applyTheme('system')
    }
  })
}

function applyTheme(id) {
  let resolved = id
  if (id === 'system') {
    resolved = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  document.documentElement.setAttribute('data-theme', resolved)
}

<script>
  import { onMount } from 'svelte'
  import { open as openDialog } from '@tauri-apps/plugin-dialog'
  import { preview, project, ui, setupPreviewListeners, setupWatcherListeners, doc, tabs } from './lib/stores/app.svelte.js'
  import { initTheme } from './lib/stores/theme.svelte.js'
  import { projectOpen, projectClose } from './lib/api.js'
  import { restoreUiState, persistUiState } from './lib/stores/window-state.js'
  import './app.css'

  import WelcomeScreen from './lib/components/WelcomeScreen.svelte'
  import CreateProjectWizard from './lib/components/CreateProjectWizard.svelte'
  import EditorLayout from './lib/components/EditorLayout.svelte'
  import OnboardingTour from './lib/components/OnboardingTour.svelte'
  import TitleBar from './lib/components/TitleBar.svelte'

  let showOnboarding = $state(false)

  onMount(() => {
    initTheme()
    const cleanupPreview = setupPreviewListeners()
    const cleanupWatcher = setupWatcherListeners()
    restoreUiState(ui)

    if (!localStorage.getItem('coderoo-onboarding-done')) {
      showOnboarding = true
    }

    const openProjectHandler = (event) => {
      if (event.detail) openProject(event.detail)
    }
    const beforeUnloadHandler = () => persistUiState(ui)
    const showOnboardingHandler = () => { showOnboarding = true }
    window.addEventListener('coderoo:open-project', openProjectHandler)
    window.addEventListener('beforeunload', beforeUnloadHandler)
    window.addEventListener('coderoo:show-onboarding', showOnboardingHandler)
    return () => {
      window.removeEventListener('coderoo:open-project', openProjectHandler)
      window.removeEventListener('beforeunload', beforeUnloadHandler)
      window.removeEventListener('coderoo:show-onboarding', showOnboardingHandler)
      cleanupPreview.then(fn => fn())
      cleanupWatcher.then(fn => fn())
    }
  })

  let screen = $state('launcher') // 'launcher' | 'create' | 'starting' | 'ready' | 'error'
  let projectPath = $state('')
  let projectName = $state('')
  let errorMsg = $state('')

  /** Start the project at a given path */
  async function startProject(path) {
    projectPath = path
    projectName = path.replace(/\\/g, '/').split('/').pop()
    screen = 'starting'

    addRecent(path, projectName)

    try {
      const result = await projectOpen(path)
      if (result?.title) {
        projectName = result.title
      }

      // Set the configured content path for file tree / search panel.
      project.contentPath = (result?.contentDir || `${path.replace(/\\/g, '/')}/content`).replace(/\\/g, '/')
      project.root = result?.dir || path
      project.name = projectName

      screen = 'ready'
    } catch (e) {
      const raw = String(e)
      if (raw.includes('site.yaml') || raw.includes('No such file'))
        errorMsg = 'This folder does not appear to be a Coderoo project.\nCreate one with "New Project" or pick a different folder.'
      else if (raw.includes('Permission') || raw.includes('Access'))
        errorMsg = 'Permission denied. Check that you have access to this folder.'
      else
        errorMsg = raw.replace(/^(invoke|Error):?\s*/i, '')
      screen = 'error'
    }
  }

  /** Open project via folder dialog, or directly from a path */
  async function openProject(path) {
    if (path) {
      await startProject(path)
      return
    }
    const selected = await openDialog({
      directory: true,
      title: 'Select Coderoo Project Folder',
    })
    if (selected) await startProject(selected)
  }

  /** After the wizard creates a new project, open it */
  async function onProjectCreated(path) {
    await startProject(path)
  }

  async function backToLauncher() {
    await projectClose().catch(() => {})
    screen = 'launcher'
    preview.port = 0
    preview.running = false
    project.contentPath = ''
    project.root = ''
    project.name = ''
    tabs.items = []
    tabs.activeId = null
    doc.content = ''
    doc.filePath = ''
    doc.contentPath = ''
    doc.dirty = false
    projectPath = ''
    projectName = ''
    errorMsg = ''
  }

  // --- Recent projects ---

  function addRecent(path, name) {
    try {
      let recents = JSON.parse(localStorage.getItem('coderoo-recent-projects') || '[]')
      recents = recents.filter(r => r.path !== path)
      recents.unshift({ name, path, lastOpened: new Date().toISOString() })
      if (recents.length > 10) recents.length = 10
      localStorage.setItem('coderoo-recent-projects', JSON.stringify(recents))
    } catch {}
  }
</script>

<div class="app">
  <TitleBar showTabs={screen === 'ready'} />

  {#if screen === 'launcher'}
    <WelcomeScreen onOpen={openProject} onCreate={() => (screen = 'create')} />

  {:else if screen === 'create'}
    <CreateProjectWizard onCreated={onProjectCreated} onBack={() => (screen = 'launcher')} />

  {:else if screen === 'starting'}
    <div class="center-screen">
      <div class="spinner"></div>
      <h2>Starting Coderoo...</h2>
      <p class="sub">{projectName}</p>
    </div>

  {:else if screen === 'error'}
    <div class="center-screen">
      <div class="error-icon">!</div>
      <h2>Failed to Start</h2>
      <p class="error-text">{errorMsg}</p>
      <button class="btn" onclick={backToLauncher}>Back</button>
    </div>

  {:else if screen === 'ready'}
    <EditorLayout {projectName} onClose={backToLauncher} />
  {/if}

  {#if showOnboarding}
    <OnboardingTour onDismiss={() => { showOnboarding = false; localStorage.setItem('coderoo-onboarding-done', '1') }} />
  {/if}
</div>

<style>
  .app {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: var(--cr-bg-base);
    color: var(--cr-text);
    font-family: var(--cr-font-ui);
  }

  .center-screen {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
  }

  h2 { margin: 0; font-size: 22px; }

  .sub {
    margin: 0;
    color: var(--cr-text-muted);
    font-size: 14px;
  }

  .spinner {
    width: 36px;
    height: 36px;
    border: 3px solid var(--cr-border);
    border-top-color: var(--cr-info);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .error-icon {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: var(--cr-danger);
    color: var(--cr-bg-base);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;
    font-weight: bold;
  }

  .error-text { color: var(--cr-danger); margin: 0; }
</style>

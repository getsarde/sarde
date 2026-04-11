<script>
  import { onMount } from 'svelte'
  import { open as openDialog } from '@tauri-apps/plugin-dialog'
  import { preview, project, setupPreviewListeners } from './lib/stores/app.svelte.js'
  import { projectOpen, projectClose } from './lib/api.js'
  import './app.css'

  import WelcomeScreen from './lib/components/WelcomeScreen.svelte'
  import CreateProjectWizard from './lib/components/CreateProjectWizard.svelte'
  import EditorLayout from './lib/components/EditorLayout.svelte'

  onMount(() => { setupPreviewListeners() })

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

      // Set the content path for file tree / search panel.
      project.contentPath = path.replace(/\\/g, '/') + '/content'
      project.root = path
      project.name = projectName

      screen = 'ready'
    } catch (e) {
      errorMsg = String(e)
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

  function backToLauncher() {
    projectClose().catch(() => {})
    screen = 'launcher'
    preview.port = 0
    preview.running = false
    project.contentPath = ''
    project.root = ''
    project.name = ''
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
      <button class="back-btn" onclick={backToLauncher}>Back</button>
    </div>

  {:else if screen === 'ready'}
    <EditorLayout {projectName} onClose={backToLauncher} />
  {/if}
</div>

<style>
  .app {
    height: 100vh;
    display: flex;
    background: #1e1e2e;
    color: #cdd6f4;
    font-family: system-ui, -apple-system, sans-serif;
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
    color: #6c7086;
    font-size: 14px;
  }

  .spinner {
    width: 36px;
    height: 36px;
    border: 3px solid #313244;
    border-top-color: #89b4fa;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .error-icon {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: #f38ba8;
    color: #1e1e2e;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;
    font-weight: bold;
  }

  .error-text { color: #f38ba8; margin: 0; }

  .back-btn {
    margin-top: 8px;
    padding: 8px 20px;
    border: 1px solid #313244;
    border-radius: 6px;
    background: transparent;
    color: #cdd6f4;
    font-size: 13px;
    cursor: pointer;
  }
  .back-btn:hover { background: #313244; }
</style>

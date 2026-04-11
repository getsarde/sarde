use std::path::PathBuf;
use std::sync::Mutex;
use tauri_plugin_shell::process::CommandChild;

/// Application state shared across all Tauri IPC commands.
pub struct AppState {
    /// Root directory of the currently open project.
    pub project_dir: Mutex<Option<PathBuf>>,
    /// Parsed site.yaml as a serde_yaml Value (preserves unknown fields).
    pub config: Mutex<Option<serde_yaml::Value>>,
    /// List of recently opened project directories.
    pub recent_projects: Mutex<Vec<RecentProject>>,
    /// Child process handle for `coderoo serve`.
    pub preview_child: Mutex<Option<CommandChild>>,
    /// Port the preview server is listening on.
    pub preview_port: Mutex<u16>,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct RecentProject {
    pub dir: String,
    pub title: String,
    pub opened_at: String,
}

impl AppState {
    pub fn new() -> Self {
        Self {
            project_dir: Mutex::new(None),
            config: Mutex::new(None),
            recent_projects: Mutex::new(Vec::new()),
            preview_child: Mutex::new(None),
            preview_port: Mutex::new(0),
        }
    }

    /// Returns the content directory path for the open project.
    /// Uses `content.dir` from config if set, otherwise defaults to `content/`.
    pub fn content_dir(&self) -> Option<PathBuf> {
        let project_dir = self.project_dir.lock().unwrap();
        let project_dir = project_dir.as_ref()?;

        let config = self.config.lock().unwrap();
        let content_subdir = config
            .as_ref()
            .and_then(|c| c.get("content"))
            .and_then(|c| c.get("dir"))
            .and_then(|d| d.as_str())
            .unwrap_or("content");

        Some(project_dir.join(content_subdir))
    }
}

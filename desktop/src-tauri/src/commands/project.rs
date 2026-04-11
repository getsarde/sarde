use crate::state::{AppState, RecentProject};
use std::fs;
use std::path::PathBuf;

#[derive(serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ProjectInfo {
    pub dir: String,
    pub state: String,
    pub title: String,
    pub collections: Vec<CollectionSummary>,
}

#[derive(serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CollectionSummary {
    pub name: String,
    pub title: String,
    pub page_count: usize,
}

/// Open a project: validate content/ exists, read site.yaml, cache config.
#[tauri::command]
pub fn open_project(dir: String, state: tauri::State<AppState>) -> Result<ProjectInfo, String> {
    let abs_dir = fs::canonicalize(&dir).map_err(|e| format!("Invalid path: {}", e))?;

    // Validate content/ directory exists.
    let content_dir = abs_dir.join("content");
    if !content_dir.is_dir() {
        return Err(format!(
            "Not a valid project: content/ directory not found in {}",
            abs_dir.display()
        ));
    }

    // Read and parse site.yaml.
    let config = read_site_config(&abs_dir)?;
    let title = config
        .get("site")
        .and_then(|s| s.get("title"))
        .and_then(|t| t.as_str())
        .unwrap_or("Untitled")
        .to_string();

    // Cache config.
    *state.config.lock().unwrap() = Some(config);
    *state.project_dir.lock().unwrap() = Some(abs_dir.clone());

    // Add to recent projects.
    {
        let mut recents = state.recent_projects.lock().unwrap();
        let dir_str = abs_dir.to_string_lossy().to_string();
        recents.retain(|r| r.dir != dir_str);
        recents.insert(
            0,
            RecentProject {
                dir: dir_str,
                title: title.clone(),
                opened_at: chrono::Utc::now().to_rfc3339(),
            },
        );
        if recents.len() > 10 {
            recents.truncate(10);
        }
    }

    // Scan collections.
    let collections = scan_collections(&content_dir);

    Ok(ProjectInfo {
        dir: abs_dir.to_string_lossy().to_string(),
        state: "open".into(),
        title,
        collections,
    })
}

/// Create a new project: scaffold directories, write site.yaml, _index.md, .gitignore.
#[tauri::command]
pub fn create_project(
    dir: String,
    title: String,
    state: tauri::State<AppState>,
) -> Result<ProjectInfo, String> {
    let abs_dir = PathBuf::from(&dir);

    // Check if site already exists.
    if abs_dir.join("site.yaml").exists() {
        return Err(format!("site.yaml already exists in {}", abs_dir.display()));
    }

    let title = if title.is_empty() {
        "My Site".to_string()
    } else {
        title
    };

    // Scaffold directories.
    for d in &["content", "static"] {
        fs::create_dir_all(abs_dir.join(d))
            .map_err(|e| format!("Creating directory: {}", e))?;
    }

    // Write site.yaml.
    let site_yaml = format!(
        "site:\n  title: \"{}\"\n  url: \"http://localhost:3000\"\n",
        title.replace('"', "\\\"")
    );
    fs::write(abs_dir.join("site.yaml"), &site_yaml)
        .map_err(|e| format!("Writing site.yaml: {}", e))?;

    // Write content/_index.md.
    let index_md = "---\ntitle: Welcome\n---\n\n# Welcome to your new site\n\nEdit this page at `content/_index.md`, then run `coderoo serve` to see your changes.\n";
    fs::write(abs_dir.join("content/_index.md"), index_md)
        .map_err(|e| format!("Writing _index.md: {}", e))?;

    // Write .gitignore.
    fs::write(abs_dir.join(".gitignore"), "dist/\n.cache/\n")
        .map_err(|e| format!("Writing .gitignore: {}", e))?;

    // Open the newly created project.
    open_project(abs_dir.to_string_lossy().to_string(), state)
}

/// Close the current project: stop preview if running, clear state.
#[tauri::command]
pub fn close_project(state: tauri::State<AppState>) -> Result<(), String> {
    // Kill preview if running.
    {
        let mut child = state.preview_child.lock().unwrap();
        if let Some(c) = child.take() {
            let _ = c.kill();
        }
        *state.preview_port.lock().unwrap() = 0;
    }

    *state.project_dir.lock().unwrap() = None;
    *state.config.lock().unwrap() = None;
    Ok(())
}

/// Get info about the currently open project.
#[tauri::command]
pub fn get_project_info(state: tauri::State<AppState>) -> Result<ProjectInfo, String> {
    let project_dir = state.project_dir.lock().unwrap();
    let project_dir = project_dir
        .as_ref()
        .ok_or("No project open")?;

    let config = state.config.lock().unwrap();
    let title = config
        .as_ref()
        .and_then(|c| c.get("site"))
        .and_then(|s| s.get("title"))
        .and_then(|t| t.as_str())
        .unwrap_or("Untitled")
        .to_string();

    let content_dir = state.content_dir().ok_or("No project open")?;
    let collections = scan_collections(&content_dir);

    Ok(ProjectInfo {
        dir: project_dir.to_string_lossy().to_string(),
        state: "open".into(),
        title,
        collections,
    })
}

/// List recently opened projects.
#[tauri::command]
pub fn list_recent_projects(state: tauri::State<AppState>) -> Vec<RecentProject> {
    state.recent_projects.lock().unwrap().clone()
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

fn read_site_config(project_dir: &PathBuf) -> Result<serde_yaml::Value, String> {
    let config_path = project_dir.join("site.yaml");
    if !config_path.exists() {
        // Return empty config if no site.yaml.
        return Ok(serde_yaml::Value::Mapping(Default::default()));
    }
    let data = fs::read_to_string(&config_path)
        .map_err(|e| format!("Reading site.yaml: {}", e))?;
    serde_yaml::from_str(&data).map_err(|e| format!("Parsing site.yaml: {}", e))
}

/// Public wrapper for use from other modules (e.g. config.rs).
pub fn scan_collections_pub(content_dir: &PathBuf) -> Vec<CollectionSummary> {
    scan_collections(content_dir)
}

/// Scan content/ subdirectories and count .md files per collection.
fn scan_collections(content_dir: &PathBuf) -> Vec<CollectionSummary> {
    let mut collections = Vec::new();

    let entries = match fs::read_dir(content_dir) {
        Ok(entries) => entries,
        Err(_) => return collections,
    };

    for entry in entries.flatten() {
        let path = entry.path();
        if !path.is_dir() {
            continue;
        }
        let name = match path.file_name().and_then(|n| n.to_str()) {
            Some(n) => n.to_string(),
            None => continue,
        };

        // Skip hidden directories.
        if name.starts_with('.') {
            continue;
        }

        // Count .md files in this collection.
        let page_count = walkdir::WalkDir::new(&path)
            .into_iter()
            .filter_map(|e| e.ok())
            .filter(|e| {
                e.file_type().is_file()
                    && e.path()
                        .extension()
                        .map_or(false, |ext| ext == "md" || ext == "markdown")
            })
            .count();

        let title = capitalize(&name);
        collections.push(CollectionSummary {
            name,
            title,
            page_count,
        });
    }

    collections
}

fn capitalize(s: &str) -> String {
    let mut chars = s.chars();
    match chars.next() {
        None => String::new(),
        Some(c) => c.to_uppercase().collect::<String>() + chars.as_str(),
    }
}

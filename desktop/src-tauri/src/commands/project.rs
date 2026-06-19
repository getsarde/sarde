use crate::state::{AppState, RecentProject};
use crate::watcher;
use std::fs;
use std::path::PathBuf;
use tauri_plugin_fs::FsExt;

#[derive(serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ProjectInfo {
    pub dir: String,
    pub content_dir: String,
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

/// Open a project: read sarde.yaml, validate configured content dir, cache config.
#[tauri::command]
pub fn open_project(dir: String, app_handle: tauri::AppHandle, state: tauri::State<AppState>) -> Result<ProjectInfo, String> {
    let abs_dir = fs::canonicalize(&dir).map_err(|e| format!("Invalid path: {}", e))?;

    // Read and parse sarde.yaml before validating content/, since content.dir may override it.
    let config = read_site_config(&abs_dir)?;
    let content_dir = resolve_content_dir(&abs_dir, &config);
    if !content_dir.is_dir() {
        return Err(format!(
            "Not a valid project: content directory not found at {}",
            content_dir.display()
        ));
    }

    // Grant the FS plugin access to the entire project tree so that frontend
    // readDir/readTextFile/rename/mkdir calls work on any drive.
    let scope = app_handle.fs_scope();
    let _ = scope.allow_directory(&abs_dir, true);

    let title = config
        .get("site")
        .and_then(|s| s.get("title"))
        .and_then(|t| t.as_str())
        .unwrap_or("Untitled")
        .to_string();

    // Cache config.
    *state.config.lock().unwrap() = Some(config);
    *state.project_dir.lock().unwrap() = Some(abs_dir.clone());

    // Start file system watcher for content and config changes.
    watcher::stop_watcher(&state.watcher);
    let config_path = abs_dir.join("sarde.yaml");
    match watcher::start_watcher(app_handle, content_dir.clone(), config_path) {
        Ok(w) => {
            *state.watcher.lock().unwrap() = Some(w);
        }
        Err(e) => {
            eprintln!("Warning: could not start file watcher: {}", e);
        }
    }

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
        dir: clean_path(&abs_dir),
        content_dir: clean_path(&content_dir),
        state: "open".into(),
        title,
        collections,
    })
}

/// Create a new project: scaffold directories, write sarde.yaml, _index.md, .gitignore.
/// Template determines the initial collection structure.
#[tauri::command]
pub fn create_project(
    dir: String,
    title: String,
    template: Option<String>,
    description: Option<String>,
    author: Option<String>,
    app_handle: tauri::AppHandle,
    state: tauri::State<AppState>,
) -> Result<ProjectInfo, String> {
    let abs_dir = PathBuf::from(&dir);

    // Check if site already exists.
    if abs_dir.join("sarde.yaml").exists() {
        return Err(format!("sarde.yaml already exists in {}", abs_dir.display()));
    }

    let title = if title.is_empty() {
        "My Site".to_string()
    } else {
        title
    };
    let template = template.unwrap_or_else(|| "empty".to_string());

    // Scaffold directories.
    for d in &["content", "static"] {
        fs::create_dir_all(abs_dir.join(d))
            .map_err(|e| format!("Creating directory: {}", e))?;
    }

    // Build sarde.yaml content.
    let mut yaml_parts = vec![format!(
        "site:\n  title: \"{}\"\n  url: \"http://localhost:4727\"",
        title.replace('"', "\\\"")
    )];

    if let Some(ref desc) = description {
        if !desc.is_empty() {
            yaml_parts.push(format!("  description: \"{}\"", desc.replace('"', "\\\"")));
        }
    }
    if let Some(ref auth) = author {
        if !auth.is_empty() {
            yaml_parts.push(format!("  author: \"{}\"", auth.replace('"', "\\\"")));
        }
    }

    // Template-specific config sections.
    match template.as_str() {
        "blog" => {
            yaml_parts.push("\nbuild:\n  feed: true".to_string());
        }
        "docs" => {
            yaml_parts.push("\nsidebar:\n  auto_generate: true".to_string());
            yaml_parts.push("\nbuild:\n  search: true".to_string());
        }
        _ => {} // "empty" — no extra config
    }

    yaml_parts.push(String::new()); // trailing newline
    fs::write(abs_dir.join("sarde.yaml"), yaml_parts.join("\n"))
        .map_err(|e| format!("Writing sarde.yaml: {}", e))?;

    // Write content/_index.md.
    let index_md = "---\ntitle: Welcome\n---\n\n# Welcome to your new site\n\nEdit this page at `content/_index.md`, then run `sarde dev` to see your changes.\n";
    fs::write(abs_dir.join("content/_index.md"), index_md)
        .map_err(|e| format!("Writing _index.md: {}", e))?;

    // Template-specific content scaffolding.
    match template.as_str() {
        "blog" => {
            let posts_dir = abs_dir.join("content/posts");
            fs::create_dir_all(&posts_dir)
                .map_err(|e| format!("Creating posts dir: {}", e))?;
            fs::write(
                posts_dir.join("_index.md"),
                "---\ntitle: Posts\n---\n",
            )
            .map_err(|e| format!("Writing posts/_index.md: {}", e))?;
            fs::write(
                posts_dir.join("hello-world.md"),
                &format!(
                    "---\ntitle: Hello World\ndate: {}T00:00:00Z\ndraft: true\ntags:\n  - getting-started\n---\n\n# Hello World\n\nThis is your first blog post. Edit or delete this file to get started.\n",
                    chrono::Utc::now().format("%Y-%m-%d")
                ),
            )
            .map_err(|e| format!("Writing hello-world.md: {}", e))?;
        }
        "docs" => {
            let docs_dir = abs_dir.join("content/docs");
            fs::create_dir_all(&docs_dir)
                .map_err(|e| format!("Creating docs dir: {}", e))?;
            fs::write(
                docs_dir.join("_index.md"),
                "---\ntitle: Documentation\n---\n",
            )
            .map_err(|e| format!("Writing docs/_index.md: {}", e))?;
            let lessons: &[(&str, &str, &str)] = &[
                (
                    "01-getting-started.md",
                    "Getting Started",
                    "This is the first page in your documentation. Edit it to introduce your project.\n\nFiles with a numeric prefix (`01-`, `02-`, …) are ordered automatically and can be drag-reordered in the sidebar.\n",
                ),
                (
                    "02-installation.md",
                    "Installation",
                    "Describe how users install or set up your project.\n\n## Prerequisites\n\n- Requirement one\n- Requirement two\n\n## Steps\n\n1. First step\n2. Second step\n",
                ),
                (
                    "03-configuration.md",
                    "Configuration",
                    "Document configuration options here.\n\n```yaml\n# Example configuration\nkey: value\n```\n",
                ),
                (
                    "04-next-steps.md",
                    "Next Steps",
                    "Point readers to deeper topics once they're up and running.\n\n- Link to related guides\n- Link to API references\n",
                ),
            ];
            for (i, (filename, title, body)) in lessons.iter().enumerate() {
                fs::write(
                    docs_dir.join(filename),
                    &format!(
                        "---\ntitle: {}\nweight: {}\n---\n\n# {}\n\n{}",
                        title,
                        i + 1,
                        title,
                        body
                    ),
                )
                .map_err(|e| format!("Writing {}: {}", filename, e))?;
            }
        }
        _ => {} // "empty" — no extra content
    }

    // Write .gitignore.
    fs::write(abs_dir.join(".gitignore"), "dist/\n.cache/\n")
        .map_err(|e| format!("Writing .gitignore: {}", e))?;

    // Open the newly created project.
    open_project(abs_dir.to_string_lossy().to_string(), app_handle, state)
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

    // Stop file watcher.
    watcher::stop_watcher(&state.watcher);

    *state.project_dir.lock().unwrap() = None;
    *state.config.lock().unwrap() = None;
    Ok(())
}

/// Get info about the currently open project.
#[tauri::command]
pub fn get_project_info(state: tauri::State<AppState>) -> Result<ProjectInfo, String> {
    let dir_string;
    let title;
    {
        let project_dir = state.project_dir.lock().unwrap();
        let project_dir = project_dir.as_ref().ok_or("No project open")?;
        dir_string = project_dir.to_string_lossy().to_string();

        let config = state.config.lock().unwrap();
        title = config
            .as_ref()
            .and_then(|c| c.get("site"))
            .and_then(|s| s.get("title"))
            .and_then(|t| t.as_str())
            .unwrap_or("Untitled")
            .to_string();
    }

    let content_dir = state.content_dir().ok_or("No project open")?;
    let collections = scan_collections(&content_dir);

    Ok(ProjectInfo {
        dir: clean_path(&std::path::PathBuf::from(&dir_string)),
        content_dir: clean_path(&content_dir),
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

/// Strip the Windows `\\?\` verbatim prefix from canonicalized paths.
fn clean_path(path: &std::path::Path) -> String {
    let s = path.to_string_lossy();
    #[cfg(windows)]
    {
        return s.strip_prefix("\\\\?\\").unwrap_or(&s).to_string();
    }
    #[cfg(not(windows))]
    {
        return s.into_owned();
    }
}

fn read_site_config(project_dir: &PathBuf) -> Result<serde_yaml::Value, String> {
    let config_path = project_dir.join("sarde.yaml");
    if !config_path.exists() {
        // Return empty config if no sarde.yaml.
        return Ok(serde_yaml::Value::Mapping(Default::default()));
    }
    let data = fs::read_to_string(&config_path)
        .map_err(|e| format!("Reading sarde.yaml: {}", e))?;
    serde_yaml::from_str(&data).map_err(|e| format!("Parsing sarde.yaml: {}", e))
}

fn resolve_content_dir(project_dir: &PathBuf, config: &serde_yaml::Value) -> PathBuf {
    let content_subdir = config
        .get("content")
        .and_then(|c| c.get("dir"))
        .and_then(|d| d.as_str())
        .unwrap_or("content");
    let path = PathBuf::from(content_subdir);
    if path.is_absolute() {
        path
    } else {
        project_dir.join(path)
    }
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

use crate::state::AppState;
use crate::yaml;
use std::fs;

#[derive(serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ContentSummary {
    pub path: String,
    pub title: String,
    pub draft: bool,
    pub date: String,
    pub weight: i64,
}

#[derive(serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ContentFile {
    pub path: String,
    pub title: String,
    pub collection: String,
    pub frontmatter: serde_json::Value,
    pub body: String,
    pub draft: bool,
    pub date: String,
    pub word_count: usize,
    pub reading_time: usize,
}

/// List content files, optionally filtered by collection.
#[tauri::command]
pub fn list_content(
    collection: Option<String>,
    state: tauri::State<AppState>,
) -> Result<Vec<ContentSummary>, String> {
    let content_dir = state.content_dir().ok_or("No project open")?;

    let mut summaries = Vec::new();
    let walker = walkdir::WalkDir::new(&content_dir)
        .into_iter()
        .filter_map(|e| e.ok())
        .filter(|e| {
            e.file_type().is_file()
                && e.path()
                    .extension()
                    .map_or(false, |ext| ext == "md" || ext == "markdown")
        });

    for entry in walker {
        let abs_path = entry.path();
        let rel_path = abs_path
            .strip_prefix(&content_dir)
            .unwrap_or(abs_path)
            .to_string_lossy()
            .replace('\\', "/");

        // Filter by collection if specified.
        if let Some(ref col) = collection {
            let first_segment = rel_path.split('/').next().unwrap_or("");
            if first_segment != col.as_str() {
                continue;
            }
        }

        // Read and parse frontmatter.
        let raw = match fs::read_to_string(abs_path) {
            Ok(r) => r,
            Err(_) => continue,
        };
        let (fm, body) = yaml::parse_frontmatter(&raw);
        let (title, draft, date, weight, _, _) = yaml::extract_summary(&fm, &body);

        summaries.push(ContentSummary {
            path: rel_path,
            title,
            draft,
            date,
            weight,
        });
    }

    Ok(summaries)
}

/// Read a single content file, returning frontmatter and body.
#[tauri::command]
pub fn read_content(
    path: String,
    state: tauri::State<AppState>,
) -> Result<ContentFile, String> {
    let content_dir = state.content_dir().ok_or("No project open")?;
    yaml::validate_content_path(&path)?;

    let abs_path = content_dir.join(path.replace('/', std::path::MAIN_SEPARATOR_STR));
    let raw = fs::read_to_string(&abs_path)
        .map_err(|e| format!("Reading file: {}", e))?;

    let (fm, body) = yaml::parse_frontmatter(&raw);
    let (title, draft, date, _, word_count, reading_time) = yaml::extract_summary(&fm, &body);

    // Determine collection from path.
    let collection = path
        .split('/')
        .next()
        .filter(|_| path.contains('/'))
        .unwrap_or("")
        .to_string();

    // Convert serde_yaml::Value to serde_json::Value for the frontend.
    let fm_json = yaml_to_json(&fm);

    Ok(ContentFile {
        path,
        title,
        collection,
        frontmatter: fm_json,
        body,
        draft,
        date,
        word_count,
        reading_time,
    })
}

/// Save frontmatter and body to an existing content file.
#[tauri::command]
pub fn save_content(
    path: String,
    frontmatter: serde_json::Value,
    body: String,
    state: tauri::State<AppState>,
) -> Result<(), String> {
    let content_dir = state.content_dir().ok_or("No project open")?;
    yaml::validate_content_path(&path)?;

    let abs_path = content_dir.join(path.replace('/', std::path::MAIN_SEPARATOR_STR));

    // Convert JSON frontmatter to YAML Value.
    let fm_yaml = json_to_yaml(&frontmatter);
    let content = yaml::serialize_frontmatter(&fm_yaml, &body);

    // Atomic write: temp file + rename.
    let tmp_path = abs_path.with_extension("md.tmp");
    fs::write(&tmp_path, &content)
        .map_err(|e| format!("Writing temp file: {}", e))?;
    fs::rename(&tmp_path, &abs_path)
        .map_err(|e| format!("Renaming temp file: {}", e))?;

    Ok(())
}

/// Create a new content file in a collection.
#[tauri::command]
pub fn create_content(
    collection: String,
    title: String,
    state: tauri::State<AppState>,
) -> Result<ContentFile, String> {
    let content_dir = state.content_dir().ok_or("No project open")?;

    let slug = yaml::slugify(&title);
    if slug.is_empty() {
        return Err(format!("Cannot generate slug from title \"{}\"", title));
    }

    let rel_path = if collection.is_empty() {
        format!("{}.md", slug)
    } else {
        format!("{}/{}.md", collection, slug)
    };

    yaml::validate_content_path(&rel_path)?;

    let abs_path = content_dir.join(rel_path.replace('/', std::path::MAIN_SEPARATOR_STR));
    if abs_path.exists() {
        return Err(format!("File already exists: {}", rel_path));
    }

    // Ensure parent directory exists.
    if let Some(parent) = abs_path.parent() {
        fs::create_dir_all(parent)
            .map_err(|e| format!("Creating directory: {}", e))?;
    }

    // Scaffold frontmatter.
    let project_dir = state.project_dir.lock().unwrap();
    let fm = scaffold_frontmatter(project_dir.as_deref(), &collection, &title);
    let body = "\n";
    let content = yaml::serialize_frontmatter(&fm, body);
    fs::write(&abs_path, &content)
        .map_err(|e| format!("Writing file: {}", e))?;

    let fm_json = yaml_to_json(&fm);
    let now = chrono::Utc::now().to_rfc3339();

    Ok(ContentFile {
        path: rel_path,
        title,
        collection,
        frontmatter: fm_json,
        body: body.to_string(),
        draft: true,
        date: now,
        word_count: 0,
        reading_time: 0,
    })
}

/// Delete a content file.
#[tauri::command]
pub fn delete_content(
    path: String,
    state: tauri::State<AppState>,
) -> Result<(), String> {
    let content_dir = state.content_dir().ok_or("No project open")?;
    yaml::validate_content_path(&path)?;

    let abs_path = content_dir.join(path.replace('/', std::path::MAIN_SEPARATOR_STR));
    fs::remove_file(&abs_path)
        .map_err(|e| format!("Deleting file: {}", e))?;

    Ok(())
}

/// Rename/move a content file.
#[tauri::command]
pub fn rename_content(
    old_path: String,
    new_path: String,
    state: tauri::State<AppState>,
) -> Result<(), String> {
    let content_dir = state.content_dir().ok_or("No project open")?;
    yaml::validate_content_path(&old_path)?;
    yaml::validate_content_path(&new_path)?;

    let abs_old = content_dir.join(old_path.replace('/', std::path::MAIN_SEPARATOR_STR));
    let abs_new = content_dir.join(new_path.replace('/', std::path::MAIN_SEPARATOR_STR));

    // Ensure target parent directory exists.
    if let Some(parent) = abs_new.parent() {
        fs::create_dir_all(parent)
            .map_err(|e| format!("Creating directory: {}", e))?;
    }

    fs::rename(&abs_old, &abs_new)
        .map_err(|e| format!("Renaming file: {}", e))?;

    Ok(())
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

/// Scaffold frontmatter for a new content file.
/// Layers: base defaults → schema defaults → archetype → user title + date.
fn scaffold_frontmatter(
    project_dir: Option<&std::path::Path>,
    collection: &str,
    title: &str,
) -> serde_yaml::Value {
    let mut fm = serde_yaml::Mapping::new();
    fm.insert(
        serde_yaml::Value::String("draft".into()),
        serde_yaml::Value::Bool(true),
    );

    // Load archetype if available.
    if let Some(proj_dir) = project_dir {
        if !collection.is_empty() {
            if let Some(arch_fm) = load_archetype(proj_dir, collection) {
                if let serde_yaml::Value::Mapping(arch_map) = arch_fm {
                    for (k, v) in arch_map {
                        fm.insert(k, v);
                    }
                }
            }
        }
    }

    // Always set title and date.
    fm.insert(
        serde_yaml::Value::String("title".into()),
        serde_yaml::Value::String(title.to_string()),
    );

    // Set date if not already present (or if empty).
    let has_valid_date = fm
        .get(&serde_yaml::Value::String("date".into()))
        .and_then(|v| v.as_str())
        .map_or(false, |s| !s.is_empty());

    if !has_valid_date {
        fm.insert(
            serde_yaml::Value::String("date".into()),
            serde_yaml::Value::String(chrono::Utc::now().to_rfc3339()),
        );
    }

    serde_yaml::Value::Mapping(fm)
}

/// Load archetype frontmatter from archetypes/<collection>.md or archetypes/default.md.
fn load_archetype(project_dir: &std::path::Path, collection: &str) -> Option<serde_yaml::Value> {
    let candidates = [
        project_dir.join("archetypes").join(format!("{}.md", collection)),
        project_dir.join("archetypes").join("default.md"),
    ];

    for path in &candidates {
        if let Ok(raw) = fs::read_to_string(path) {
            let (fm, _) = yaml::parse_frontmatter(&raw);
            if let serde_yaml::Value::Mapping(ref m) = fm {
                if !m.is_empty() {
                    return Some(fm);
                }
            }
        }
    }
    None
}

/// Convert serde_yaml::Value to serde_json::Value.
fn yaml_to_json(val: &serde_yaml::Value) -> serde_json::Value {
    // Round-trip through string to convert types correctly.
    match serde_json::to_value(val) {
        Ok(v) => v,
        Err(_) => serde_json::Value::Null,
    }
}

/// Convert serde_json::Value to serde_yaml::Value.
fn json_to_yaml(val: &serde_json::Value) -> serde_yaml::Value {
    match serde_yaml::to_value(val) {
        Ok(v) => v,
        Err(_) => serde_yaml::Value::Mapping(Default::default()),
    }
}

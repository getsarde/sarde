use crate::state::AppState;
use std::fs;

/// Get the current site configuration (parsed site.yaml).
#[tauri::command]
pub fn get_config(state: tauri::State<AppState>) -> Result<serde_json::Value, String> {
    let config = state.config.lock().unwrap();
    let config = config.as_ref().ok_or("No project open")?;

    // Convert YAML value to JSON for the frontend.
    serde_json::to_value(config).map_err(|e| format!("Serializing config: {}", e))
}

/// Update site config: read site.yaml, merge provided top-level sections, write back.
#[tauri::command]
pub fn update_config(
    settings: serde_json::Value,
    state: tauri::State<AppState>,
) -> Result<(), String> {
    let config_path = {
        let project_dir = state.project_dir.lock().unwrap();
        let project_dir = project_dir.as_ref().ok_or("No project open")?;
        project_dir.join("site.yaml")
    };

    // Read current site.yaml.
    let data = fs::read_to_string(&config_path)
        .map_err(|e| format!("Reading site.yaml: {}", e))?;
    let mut raw: serde_yaml::Value =
        serde_yaml::from_str(&data).unwrap_or(serde_yaml::Value::Mapping(Default::default()));

    // Merge each top-level key from settings into the root mapping.
    let root = raw
        .as_mapping_mut()
        .ok_or("Invalid site.yaml format")?;

    if let Some(obj) = settings.as_object() {
        for (key, val) in obj {
            let yaml_key = serde_yaml::Value::String(key.clone());
            let yaml_val =
                serde_yaml::to_value(val).map_err(|e| format!("Converting value: {}", e))?;
            root.insert(yaml_key, yaml_val);
        }
    }

    // Write back.
    let output = serde_yaml::to_string(&raw).map_err(|e| format!("Serializing site.yaml: {}", e))?;
    fs::write(&config_path, &output).map_err(|e| format!("Writing site.yaml: {}", e))?;

    // Update cached config.
    *state.config.lock().unwrap() = Some(raw);

    Ok(())
}

/// Get metadata about all collections (name, title, page count).
#[tauri::command]
pub fn get_collections(
    state: tauri::State<AppState>,
) -> Result<Vec<crate::commands::project::CollectionSummary>, String> {
    let content_dir = state.content_dir().ok_or("No project open")?;
    Ok(crate::commands::project::scan_collections_pub(&content_dir))
}

/// Get the frontmatter schema for a collection (from config.yaml in the collection dir).
#[tauri::command]
pub fn get_schema(
    collection: String,
    state: tauri::State<AppState>,
) -> Result<serde_json::Value, String> {
    let collection = collection.trim().to_string();
    if collection.is_empty() || collection.contains("..") || collection.contains('/') || collection.contains('\\') {
        return Err("Invalid collection name".into());
    }
    let content_dir = state.content_dir().ok_or("No project open")?;
    let col_dir = content_dir.join(&collection);

    // Try config.yaml then config.yml.
    for name in &["config.yaml", "config.yml"] {
        let schema_path = col_dir.join(name);
        if let Ok(data) = fs::read_to_string(&schema_path) {
            let parsed: serde_yaml::Value =
                serde_yaml::from_str(&data).map_err(|e| format!("Parsing {}: {}", name, e))?;

            // Extract frontmatter_schema field.
            if let Some(schema) = parsed.get("frontmatter_schema") {
                return serde_json::to_value(schema)
                    .map_err(|e| format!("Serializing schema: {}", e));
            }
        }
    }

    // No schema found — return null (not an error).
    Ok(serde_json::Value::Null)
}

/// Create a new collection: mkdir + write _index.md with title frontmatter.
#[tauri::command]
pub fn create_collection(
    name: String,
    state: tauri::State<AppState>,
) -> Result<crate::commands::project::CollectionSummary, String> {
    let content_dir = state.content_dir().ok_or("No project open")?;

    // Validate name: non-empty, no path traversal.
    let name = name.trim().to_string();
    if name.is_empty() || name.contains("..") || name.contains('/') || name.contains('\\') {
        return Err("Invalid collection name".into());
    }

    let col_dir = content_dir.join(&name);
    if col_dir.exists() {
        return Err(format!("Collection '{}' already exists", name));
    }

    fs::create_dir_all(&col_dir).map_err(|e| format!("Creating directory: {}", e))?;

    // Write _index.md with title.
    let title = capitalize_name(&name);
    let index_content = format!("---\ntitle: {}\n---\n", title);
    fs::write(col_dir.join("_index.md"), &index_content)
        .map_err(|e| format!("Writing _index.md: {}", e))?;

    Ok(crate::commands::project::CollectionSummary {
        name,
        title,
        page_count: 1,
    })
}

/// Delete a collection directory recursively.
#[tauri::command]
pub fn delete_collection(
    name: String,
    state: tauri::State<AppState>,
) -> Result<(), String> {
    let content_dir = state.content_dir().ok_or("No project open")?;

    let name = name.trim().to_string();
    if name.is_empty() || name.contains("..") || name.contains('/') || name.contains('\\') {
        return Err("Invalid collection name".into());
    }

    let col_dir = content_dir.join(&name);
    if !col_dir.is_dir() {
        return Err(format!("Collection '{}' not found", name));
    }

    // Ensure the path is actually inside content_dir (prevent traversal).
    let canonical = fs::canonicalize(&col_dir).map_err(|e| format!("Resolving path: {}", e))?;
    let canonical_content = fs::canonicalize(&content_dir).map_err(|e| format!("Resolving content dir: {}", e))?;
    if !canonical.starts_with(&canonical_content) {
        return Err("Path outside content directory".into());
    }

    fs::remove_dir_all(&col_dir).map_err(|e| format!("Deleting collection: {}", e))?;
    Ok(())
}

/// Read the navigation configuration.
/// Returns parsed nav.yaml if it exists, otherwise auto-generates from the content directory structure.
#[tauri::command]
pub fn read_nav(state: tauri::State<AppState>) -> Result<serde_json::Value, String> {
    let project_dir = state.project_dir.lock().unwrap();
    let project_dir = project_dir.as_ref().ok_or("No project open")?;

    let nav_path = project_dir.join("nav.yaml");

    if nav_path.exists() {
        let data =
            fs::read_to_string(&nav_path).map_err(|e| format!("Reading nav.yaml: {}", e))?;
        let parsed: serde_yaml::Value =
            serde_yaml::from_str(&data).map_err(|e| format!("Parsing nav.yaml: {}", e))?;
        let json =
            serde_json::to_value(&parsed).map_err(|e| format!("Converting nav.yaml: {}", e))?;
        return Ok(serde_json::json!({ "source": "file", "items": json }));
    }

    // Auto-generate from content directory structure.
    let content_dir = state.content_dir().ok_or("No project open")?;
    let items = auto_generate_nav(&content_dir)?;
    Ok(serde_json::json!({ "source": "auto", "items": items }))
}

/// Save navigation configuration to nav.yaml.
#[tauri::command]
pub fn save_nav(
    items: serde_json::Value,
    state: tauri::State<AppState>,
) -> Result<(), String> {
    let project_dir = state.project_dir.lock().unwrap();
    let project_dir = project_dir.as_ref().ok_or("No project open")?;

    let nav_path = project_dir.join("nav.yaml");
    let yaml_val: serde_yaml::Value =
        serde_yaml::to_value(&items).map_err(|e| format!("Converting to YAML: {}", e))?;
    let output =
        serde_yaml::to_string(&yaml_val).map_err(|e| format!("Serializing nav.yaml: {}", e))?;
    fs::write(&nav_path, &output).map_err(|e| format!("Writing nav.yaml: {}", e))?;
    Ok(())
}

/// Delete nav.yaml to reset to auto-generated navigation.
#[tauri::command]
pub fn delete_nav(state: tauri::State<AppState>) -> Result<(), String> {
    let project_dir = state.project_dir.lock().unwrap();
    let project_dir = project_dir.as_ref().ok_or("No project open")?;

    let nav_path = project_dir.join("nav.yaml");
    if nav_path.exists() {
        fs::remove_file(&nav_path).map_err(|e| format!("Deleting nav.yaml: {}", e))?;
    }
    Ok(())
}

/// Auto-generate a navigation tree from content directory structure.
fn auto_generate_nav(content_dir: &std::path::Path) -> Result<serde_json::Value, String> {
    let mut items = Vec::new();

    let mut entries: Vec<_> = fs::read_dir(content_dir)
        .map_err(|e| format!("Reading content dir: {}", e))?
        .filter_map(|e| e.ok())
        .collect();
    entries.sort_by_key(|e| e.file_name());

    for entry in entries {
        let name = entry.file_name().to_string_lossy().to_string();
        if name.starts_with('.') || name.starts_with('_') {
            continue;
        }

        let path = entry.path();
        if path.is_dir() {
            let label = extract_dir_title(&path).unwrap_or_else(|| capitalize_name(&name));
            let children = auto_generate_nav_children(&path, &name)?;
            items.push(serde_json::json!({
                "label": label,
                "path": format!("/{}/", name),
                "children": children,
                "auto": true,
            }));
        } else if name.ends_with(".md") && name != "_index.md" {
            let stem = name.trim_end_matches(".md");
            let label = capitalize_name(stem);
            items.push(serde_json::json!({
                "label": label,
                "path": format!("/{}/", stem),
                "auto": true,
            }));
        }
    }

    Ok(serde_json::Value::Array(items))
}

fn auto_generate_nav_children(
    dir: &std::path::Path,
    parent_slug: &str,
) -> Result<Vec<serde_json::Value>, String> {
    let mut items = Vec::new();

    let mut entries: Vec<_> = fs::read_dir(dir)
        .map_err(|e| format!("Reading dir: {}", e))?
        .filter_map(|e| e.ok())
        .collect();
    entries.sort_by_key(|e| e.file_name());

    for entry in entries {
        let name = entry.file_name().to_string_lossy().to_string();
        if name.starts_with('.') || name == "_index.md" {
            continue;
        }

        let path = entry.path();
        if path.is_dir() {
            let label = extract_dir_title(&path).unwrap_or_else(|| capitalize_name(&name));
            let slug = format!("{}/{}", parent_slug, name);
            let children = auto_generate_nav_children(&path, &slug)?;
            items.push(serde_json::json!({
                "label": label,
                "path": format!("/{}/", slug),
                "children": children,
                "auto": true,
            }));
        } else if name.ends_with(".md") {
            let stem = name.trim_end_matches(".md");
            let clean_stem = stem
                .trim_start_matches(|c: char| c.is_ascii_digit() || c == '-')
                .trim_start_matches('-');
            let display_stem = if clean_stem.is_empty() { stem } else { clean_stem };
            let label = capitalize_name(display_stem);
            items.push(serde_json::json!({
                "label": label,
                "path": format!("/{}/{}/", parent_slug, stem),
                "auto": true,
            }));
        }
    }

    Ok(items)
}

/// Try to extract title from _index.md in a directory.
fn extract_dir_title(dir: &std::path::Path) -> Option<String> {
    let index_path = dir.join("_index.md");
    if let Ok(content) = fs::read_to_string(&index_path) {
        let (fm, _) = crate::yaml::parse_frontmatter(&content);
        if let Some(title) = fm.get("title").and_then(|v| v.as_str()) {
            if !title.is_empty() {
                return Some(title.to_string());
            }
        }
    }
    None
}

fn capitalize_name(s: &str) -> String {
    let mut chars = s.chars();
    match chars.next() {
        None => String::new(),
        Some(c) => c.to_uppercase().collect::<String>() + chars.as_str(),
    }
}

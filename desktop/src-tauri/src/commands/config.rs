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
    let project_dir = state.project_dir.lock().unwrap();
    let project_dir = project_dir.as_ref().ok_or("No project open")?;

    let config_path = project_dir.join("site.yaml");

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

fn capitalize_name(s: &str) -> String {
    let mut chars = s.chars();
    match chars.next() {
        None => String::new(),
        Some(c) => c.to_uppercase().collect::<String>() + chars.as_str(),
    }
}

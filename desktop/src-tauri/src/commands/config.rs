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

/// Update site config: read site.yaml, merge provided fields into the `site` section, write back.
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

    // Ensure site section exists.
    let site_section = raw
        .as_mapping_mut()
        .ok_or("Invalid site.yaml format")?;

    let site_key = serde_yaml::Value::String("site".into());
    if !site_section.contains_key(&site_key) {
        site_section.insert(site_key.clone(), serde_yaml::Value::Mapping(Default::default()));
    }

    let site = site_section
        .get_mut(&site_key)
        .and_then(|v| v.as_mapping_mut())
        .ok_or("Invalid site section in site.yaml")?;

    // Merge settings into the site section.
    if let Some(obj) = settings.as_object() {
        for (key, val) in obj {
            let yaml_key = serde_yaml::Value::String(key.clone());
            let yaml_val =
                serde_yaml::to_value(val).map_err(|e| format!("Converting value: {}", e))?;
            site.insert(yaml_key, yaml_val);
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

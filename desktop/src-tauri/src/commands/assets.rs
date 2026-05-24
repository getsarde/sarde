use crate::state::AppState;
use crate::yaml;
use base64::Engine;
use std::fs;
use std::path::{Path, PathBuf};

const ASSET_EXTENSIONS: &[&str] = &[
    "png", "jpg", "jpeg", "gif", "webp", "svg", "mp4", "webm", "pdf", "ico",
];

const IMAGE_EXTENSIONS: &[&str] = &["png", "jpg", "jpeg", "gif", "webp"];

#[derive(serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AssetInfo {
    pub path: String,
    pub filename: String,
    pub size_bytes: u64,
    pub mime_type: String,
    pub dimensions: Option<Dimensions>,
    pub bundle_owner: Option<String>,
    pub last_modified: String,
}

#[derive(serde::Serialize)]
pub struct Dimensions {
    pub width: u32,
    pub height: u32,
}

#[derive(serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AssetDestination {
    pub target: String,
    pub bundle_path: Option<String>,
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

/// List assets from content/ (bundle) and/or static/ (shared) directories.
#[tauri::command]
pub fn asset_list(
    scope: Option<String>,
    state: tauri::State<AppState>,
) -> Result<Vec<AssetInfo>, String> {
    let project_dir = state
        .project_dir
        .lock()
        .unwrap()
        .clone()
        .ok_or("No project open")?;

    let scope = scope.unwrap_or_else(|| "all".into());
    let content_dir = state.content_dir();
    let static_dir = project_dir.join("static");
    let mut assets = Vec::new();

    if (scope == "all" || scope == "bundle") && content_dir.is_some() {
        let cd = content_dir.unwrap();
        if cd.exists() {
            collect_assets(&cd, &project_dir, Some(&cd), &mut assets);
        }
    }

    if scope == "all" || scope == "shared" {
        if static_dir.exists() {
            collect_assets(&static_dir, &project_dir, None, &mut assets);
        }
    }

    // Sort by filename.
    assets.sort_by(|a, b| a.filename.to_lowercase().cmp(&b.filename.to_lowercase()));

    Ok(assets)
}

/// Generate or return a cached base64 thumbnail for an asset.
#[tauri::command]
pub fn asset_get_thumbnail(
    path: String,
    state: tauri::State<AppState>,
) -> Result<String, String> {
    yaml::validate_content_path(&path)?;

    let project_dir = state
        .project_dir
        .lock()
        .unwrap()
        .clone()
        .ok_or("No project open")?;

    let abs_path = yaml::safe_join(&project_dir, &path, true)?;
    if !abs_path.exists() {
        return Err("Asset not found".into());
    }

    // Verify the resolved path is under the project directory.
    let canonical = abs_path
        .canonicalize()
        .map_err(|e| format!("Canonicalize: {}", e))?;
    let proj_canonical = project_dir
        .canonicalize()
        .map_err(|e| format!("Canonicalize project: {}", e))?;
    if !canonical.starts_with(&proj_canonical) {
        return Err("Path traversal not allowed".into());
    }

    let ext = abs_path
        .extension()
        .and_then(|e| e.to_str())
        .unwrap_or("")
        .to_lowercase();

    // Non-image files get a sentinel icon type.
    if !IMAGE_EXTENSIONS.contains(&ext.as_str()) {
        return Ok(format!("icon:{}", ext));
    }

    // Check thumbnail cache.
    let cache_dir = thumb_cache_dir();
    let mtime = fs::metadata(&abs_path)
        .and_then(|m| m.modified())
        .map(|t| format!("{:?}", t))
        .unwrap_or_default();

    let cache_key = {
        let input = format!("{}|{}", canonical.to_string_lossy(), mtime);
        let mut hash: u64 = 0xcbf29ce484222325; // FNV-1a offset basis
        for byte in input.as_bytes() {
            hash ^= *byte as u64;
            hash = hash.wrapping_mul(0x100000001b3); // FNV-1a prime
        }
        format!("{:016x}.jpg", hash)
    };
    let cache_path = cache_dir.join(&cache_key);

    if cache_path.exists() {
        let data = fs::read(&cache_path).map_err(|e| format!("Reading cache: {}", e))?;
        let b64 = base64::engine::general_purpose::STANDARD.encode(&data);
        return Ok(format!("data:image/jpeg;base64,{}", b64));
    }

    // Generate thumbnail.
    let img = image::open(&abs_path).map_err(|e| format!("Opening image: {}", e))?;
    let thumb = img.thumbnail(200, 200);
    let mut buf = std::io::Cursor::new(Vec::new());
    thumb
        .write_to(&mut buf, image::ImageFormat::Jpeg)
        .map_err(|e| format!("Encoding thumbnail: {}", e))?;

    let jpeg_data = buf.into_inner();

    // Cache to disk.
    let _ = fs::create_dir_all(&cache_dir);
    let _ = fs::write(&cache_path, &jpeg_data);

    let b64 = base64::engine::general_purpose::STANDARD.encode(&jpeg_data);
    Ok(format!("data:image/jpeg;base64,{}", b64))
}

/// Upload assets via native file picker dialog.
#[tauri::command]
pub fn asset_upload(
    destination: AssetDestination,
    state: tauri::State<AppState>,
    app_handle: tauri::AppHandle,
) -> Result<Vec<AssetInfo>, String> {
    let project_dir = state
        .project_dir
        .lock()
        .unwrap()
        .clone()
        .ok_or("No project open")?;

    // Determine destination directory.
    let content_dir_for_info = state.content_dir();
    let dest_dir = match destination.target.as_str() {
        "shared" => project_dir.join("static"),
        "bundle" => {
            let bundle = destination
                .bundle_path
                .ok_or("bundle_path required for bundle target")?;
            let content_dir = state.content_dir().ok_or("No content directory")?;
            if bundle == "." {
                content_dir
            } else {
                yaml::safe_join(&content_dir, &bundle, true)?
            }
        }
        _ => return Err(format!("Invalid target: {}", destination.target)),
    };

    // Ensure destination exists.
    fs::create_dir_all(&dest_dir).map_err(|e| format!("Creating directory: {}", e))?;

    // Open native file picker.
    use tauri_plugin_dialog::DialogExt;
    let files = app_handle
        .dialog()
        .file()
        .add_filter(
            "Images & Media",
            &["png", "jpg", "jpeg", "gif", "webp", "svg", "mp4", "webm", "pdf", "ico"],
        )
        .blocking_pick_files();

    let files = match files {
        Some(f) => f,
        None => return Ok(Vec::new()), // User cancelled.
    };

    let mut uploaded = Vec::new();

    for file_path in &files {
        let src = file_path.as_path().ok_or("Invalid file path")?;
        let file_name = src
            .file_name()
            .and_then(|n| n.to_str())
            .ok_or("Invalid filename")?;

        // Handle name collisions.
        let dest_path = unique_path(&dest_dir, file_name);

        fs::copy(src, &dest_path).map_err(|e| format!("Copying file: {}", e))?;

        // Build AssetInfo for the uploaded file.
        let content_scope = if destination.target == "bundle" {
            content_dir_for_info.as_ref()
        } else {
            None
        };
        if let Some(info) = build_asset_info(&dest_path, &project_dir, content_scope) {
            uploaded.push(info);
        }
    }

    Ok(uploaded)
}

/// Delete an asset file.
#[tauri::command]
pub fn asset_delete(path: String, state: tauri::State<AppState>) -> Result<(), String> {
    yaml::validate_content_path(&path)?;

    let project_dir = state
        .project_dir
        .lock()
        .unwrap()
        .clone()
        .ok_or("No project open")?;

    let abs_path = yaml::safe_join(&project_dir, &path, true)?;

    // Safety: verify the file is under content/ or static/.
    let canonical = abs_path
        .canonicalize()
        .map_err(|e| format!("Canonicalize: {}", e))?;
    let proj_canonical = project_dir
        .canonicalize()
        .map_err(|e| format!("Canonicalize project: {}", e))?;

    let content_dir = state.content_dir();
    let static_dir = project_dir.join("static");

    let under_content = content_dir
        .as_ref()
        .and_then(|cd| cd.canonicalize().ok())
        .map_or(false, |cd| canonical.starts_with(&cd));
    let under_static = static_dir
        .canonicalize()
        .map_or(false, |sd| canonical.starts_with(&sd));

    if !under_content && !under_static {
        return Err("Asset must be under content/ or static/".into());
    }

    if !canonical.starts_with(&proj_canonical) {
        return Err("Path traversal not allowed".into());
    }

    fs::remove_file(&abs_path).map_err(|e| format!("Deleting asset: {}", e))?;

    // Clean up cached thumbnail.
    let _ = remove_cached_thumbnail(&abs_path);

    Ok(())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Walk a directory and collect asset files.
fn collect_assets(
    dir: &Path,
    project_root: &Path,
    content_dir: Option<&PathBuf>,
    assets: &mut Vec<AssetInfo>,
) {
    let walker = walkdir::WalkDir::new(dir)
        .into_iter()
        .filter_map(|e| e.ok())
        .filter(|e| {
            e.file_type().is_file()
                && e.path()
                    .extension()
                    .and_then(|ext| ext.to_str())
                    .map_or(false, |ext| ASSET_EXTENSIONS.contains(&ext.to_lowercase().as_str()))
        });

    for entry in walker {
        if let Some(info) = build_asset_info(entry.path(), project_root, content_dir) {
            assets.push(info);
        }
    }
}

/// Build an AssetInfo from an absolute file path.
fn build_asset_info(
    abs_path: &Path,
    project_root: &Path,
    content_dir: Option<&PathBuf>,
) -> Option<AssetInfo> {
    let metadata = fs::metadata(abs_path).ok()?;
    let filename = abs_path.file_name()?.to_str()?.to_string();
    let rel_path = abs_path
        .strip_prefix(project_root)
        .ok()?
        .to_string_lossy()
        .replace('\\', "/");

    let ext = abs_path
        .extension()
        .and_then(|e| e.to_str())
        .unwrap_or("")
        .to_lowercase();

    // Image dimensions (header-only read, fast).
    let dimensions = if IMAGE_EXTENSIONS.contains(&ext.as_str()) {
        image::image_dimensions(abs_path)
            .ok()
            .map(|(w, h)| Dimensions { width: w, height: h })
    } else {
        None
    };

    // Determine bundle owner (if asset is co-located with content).
    let bundle_owner = content_dir.and_then(|cd| {
        let parent = abs_path.parent()?;
        let rel_parent = parent.strip_prefix(cd).ok()?;
        let rel_str = rel_parent.to_string_lossy().replace('\\', "/");
        if rel_str.is_empty() {
            None
        } else {
            Some(rel_str)
        }
    });

    let last_modified = metadata
        .modified()
        .ok()
        .and_then(|t| {
            let dt: chrono::DateTime<chrono::Utc> = t.into();
            Some(dt.to_rfc3339())
        })
        .unwrap_or_default();

    Some(AssetInfo {
        path: rel_path,
        filename,
        size_bytes: metadata.len(),
        mime_type: mime_from_ext(&ext).to_string(),
        dimensions,
        bundle_owner,
        last_modified,
    })
}

fn mime_from_ext(ext: &str) -> &'static str {
    match ext {
        "png" => "image/png",
        "jpg" | "jpeg" => "image/jpeg",
        "gif" => "image/gif",
        "webp" => "image/webp",
        "svg" => "image/svg+xml",
        "ico" => "image/x-icon",
        "mp4" => "video/mp4",
        "webm" => "video/webm",
        "pdf" => "application/pdf",
        _ => "application/octet-stream",
    }
}

/// Return a unique file path, appending -1, -2, etc. if the name already exists.
fn unique_path(dir: &Path, name: &str) -> PathBuf {
    let target = dir.join(name);
    if !target.exists() {
        return target;
    }

    let stem = Path::new(name)
        .file_stem()
        .and_then(|s| s.to_str())
        .unwrap_or(name);
    let ext = Path::new(name)
        .extension()
        .and_then(|e| e.to_str())
        .unwrap_or("");

    for i in 1..1000 {
        let candidate = if ext.is_empty() {
            dir.join(format!("{}-{}", stem, i))
        } else {
            dir.join(format!("{}-{}.{}", stem, i, ext))
        };
        if !candidate.exists() {
            return candidate;
        }
    }

    // Fallback — extremely unlikely.
    dir.join(format!("{}-copy.{}", stem, ext))
}

fn thumb_cache_dir() -> PathBuf {
    std::env::temp_dir().join("sarde-thumbs")
}

fn remove_cached_thumbnail(abs_path: &Path) -> Result<(), std::io::Error> {
    let cache_dir = thumb_cache_dir();
    if !cache_dir.exists() {
        return Ok(());
    }

    // We don't know the exact cache key without mtime, so just try to clean up
    // by hashing the canonical path with a wildcard approach.
    // For simplicity, we skip this — thumbnails are cheap and will be regenerated.
    let _ = abs_path;
    let _ = cache_dir;
    Ok(())
}

use notify::RecursiveMode;
use notify_debouncer_mini::{new_debouncer, DebouncedEvent, DebouncedEventKind};
use std::path::PathBuf;
use std::sync::Mutex;
use std::time::Duration;
use tauri::{AppHandle, Emitter};

pub type Debouncer = notify_debouncer_mini::Debouncer<notify::RecommendedWatcher>;

#[derive(Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct FsEvent {
    path: String,
    kind: String,
}

pub fn start_watcher(
    app_handle: AppHandle,
    content_dir: PathBuf,
    config_path: PathBuf,
) -> Result<Debouncer, String> {
    let handle = app_handle.clone();

    let mut debouncer = new_debouncer(
        Duration::from_millis(500),
        move |res: Result<Vec<DebouncedEvent>, notify::Error>| match res {
            Ok(events) => {
                for event in events {
                    let path_str = event.path.to_string_lossy().to_string();
                    let kind = match event.kind {
                        DebouncedEventKind::Any | DebouncedEventKind::AnyContinuous => "changed",
                        _ => continue,
                    };

                    let is_md = event
                        .path
                        .extension()
                        .map_or(false, |ext| ext == "md" || ext == "markdown");
                    let is_yaml = event
                        .path
                        .extension()
                        .map_or(false, |ext| ext == "yaml" || ext == "yml");
                    let is_relevant = is_md || is_yaml || !event.path.is_file();

                    if !is_relevant {
                        continue;
                    }

                    let payload = FsEvent {
                        path: path_str,
                        kind: kind.to_string(),
                    };

                    let _ = handle.emit("fs:changed", &payload);
                }
            }
            Err(e) => {
                eprintln!("File watcher error: {:?}", e);
            }
        },
    )
    .map_err(|e| format!("Failed to create file watcher: {}", e))?;

    debouncer
        .watcher()
        .watch(&content_dir, RecursiveMode::Recursive)
        .map_err(|e| format!("Failed to watch content dir: {}", e))?;

    if config_path.exists() {
        let _ = debouncer
            .watcher()
            .watch(&config_path, RecursiveMode::NonRecursive);
    }

    Ok(debouncer)
}

pub fn stop_watcher(watcher: &Mutex<Option<Debouncer>>) {
    let mut w = watcher.lock().unwrap();
    *w = None;
}

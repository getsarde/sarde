use crate::state::AppState;
use tauri::Emitter;
use tauri::Manager;
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::CommandEvent;

/// Run `coderoo build` on the current project. Streams stdout/stderr as Tauri events.
#[tauri::command]
pub async fn run_build(app: tauri::AppHandle) -> Result<serde_json::Value, String> {
    let state = app.state::<AppState>();
    let project_dir = {
        let pd = state.project_dir.lock().unwrap();
        pd.as_ref()
            .ok_or("No project open")?
            .to_string_lossy()
            .to_string()
    };

    let shell = app.shell();
    let cmd = shell
        .sidecar("coderoo")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?;

    let (mut rx, _child) = cmd
        .args(["build", &project_dir])
        .spawn()
        .map_err(|e| format!("Failed to spawn build: {}", e))?;

    let app_handle = app.clone();
    let _ = app_handle.emit("build:started", ());

    let mut last_line = String::new();
    while let Some(event) = rx.recv().await {
        match event {
            CommandEvent::Stdout(line) => {
                let text = String::from_utf8_lossy(&line).trim().to_string();
                if !text.is_empty() {
                    last_line = text.clone();
                    let _ = app_handle.emit("build:stdout", &text);
                }
            }
            CommandEvent::Stderr(line) => {
                let text = String::from_utf8_lossy(&line).trim().to_string();
                if !text.is_empty() {
                    let _ = app_handle.emit("build:stderr", &text);
                }
            }
            CommandEvent::Terminated(status) => {
                let code = status.code.unwrap_or(-1);
                if code == 0 {
                    let _ = app_handle.emit("build:complete", ());
                } else {
                    let _ = app_handle.emit("build:error", &format!("Exit code: {}", code));
                }
            }
            _ => {}
        }
    }

    // Try to parse last stdout line as JSON (build stats).
    match serde_json::from_str::<serde_json::Value>(&last_line) {
        Ok(stats) => Ok(stats),
        Err(_) => Ok(serde_json::json!({ "status": "complete" })),
    }
}

/// Start the preview server: spawn `coderoo serve`, parse port from stdout.
#[tauri::command]
pub async fn start_preview(app: tauri::AppHandle) -> Result<u16, String> {
    let state = app.state::<AppState>();

    // Check if preview already running.
    {
        let child = state.preview_child.lock().unwrap();
        if child.is_some() {
            let port = *state.preview_port.lock().unwrap();
            return Ok(port);
        }
    }

    let project_dir = {
        let pd = state.project_dir.lock().unwrap();
        pd.as_ref()
            .ok_or("No project open")?
            .to_string_lossy()
            .to_string()
    };

    let shell = app.shell();
    let cmd = shell
        .sidecar("coderoo")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?;

    let (mut rx, child) = cmd
        .args(["serve", &project_dir, "--port", "0"])
        .spawn()
        .map_err(|e| format!("Failed to spawn preview: {}", e))?;

    *state.preview_child.lock().unwrap() = Some(child);

    let app_handle = app.clone();

    // Listen for the ready signal and stderr in a background task.
    tauri::async_runtime::spawn(async move {
        let state = app_handle.state::<AppState>();
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    let text = String::from_utf8_lossy(&line);
                    let trimmed = text.trim();

                    // Parse JSON: {"ready": true, "port": N}
                    if let Ok(json) = serde_json::from_str::<serde_json::Value>(trimmed) {
                        if json.get("ready").and_then(|v| v.as_bool()) == Some(true) {
                            if let Some(port) = json.get("port").and_then(|v| v.as_u64()) {
                                let port = port as u16;
                                *state.preview_port.lock().unwrap() = port;
                                let _ = app_handle.emit("preview:ready", port);
                            }
                        }
                    }

                    // Forward other stdout as build log.
                    let _ = app_handle.emit("build:log", trimmed);
                }
                CommandEvent::Stderr(line) => {
                    let text = String::from_utf8_lossy(&line).trim().to_string();
                    if !text.is_empty() {
                        let _ = app_handle.emit("build:log", &text);
                    }
                }
                CommandEvent::Terminated(status) => {
                    *state.preview_child.lock().unwrap() = None;
                    *state.preview_port.lock().unwrap() = 0;
                    let _ = app_handle.emit("preview:stopped", ());

                    let code = status.code.unwrap_or(-1);
                    if code != 0 {
                        let _ = app_handle.emit("preview:crashed", code);
                    }
                }
                _ => {}
            }
        }
    });

    // Return 0 initially — the actual port arrives via the preview:ready event.
    Ok(0)
}

/// Stop the preview server.
#[tauri::command]
pub fn stop_preview(state: tauri::State<AppState>) -> Result<(), String> {
    let mut child = state.preview_child.lock().unwrap();
    if let Some(c) = child.take() {
        let _ = c.kill();
    }
    *state.preview_port.lock().unwrap() = 0;
    Ok(())
}

/// Run `coderoo validate` on the current project.
#[tauri::command]
pub async fn validate_project(app: tauri::AppHandle) -> Result<String, String> {
    let state = app.state::<AppState>();
    let project_dir = {
        let pd = state.project_dir.lock().unwrap();
        pd.as_ref()
            .ok_or("No project open")?
            .to_string_lossy()
            .to_string()
    };

    let shell = app.shell();
    let output = shell
        .sidecar("coderoo")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?
        .args(["validate", &project_dir])
        .output()
        .await
        .map_err(|e| format!("Failed to run validate: {}", e))?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    } else {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        Err(if stderr.is_empty() {
            "Validation failed".to_string()
        } else {
            stderr
        })
    }
}

/// Run `coderoo deploy` on the current project.
#[tauri::command]
pub async fn deploy(
    app: tauri::AppHandle,
    provider: Option<String>,
) -> Result<String, String> {
    let state = app.state::<AppState>();
    let project_dir = {
        let pd = state.project_dir.lock().unwrap();
        pd.as_ref()
            .ok_or("No project open")?
            .to_string_lossy()
            .to_string()
    };

    let shell = app.shell();
    let mut args = vec!["deploy".to_string(), project_dir];
    if let Some(p) = provider {
        if !p.is_empty() {
            args.push("--provider".to_string());
            args.push(p);
        }
    }

    let str_args: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let output = shell
        .sidecar("coderoo")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?
        .args(&str_args)
        .output()
        .await
        .map_err(|e| format!("Failed to run deploy: {}", e))?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    } else {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        Err(if stderr.is_empty() {
            "Deploy failed".to_string()
        } else {
            stderr
        })
    }
}

/// Render markdown to HTML via `coderoo render` (stdin → JSON stdout).
#[tauri::command]
pub async fn render_markdown(
    markdown: String,
    state: tauri::State<'_, AppState>,
) -> Result<serde_json::Value, String> {
    let sidecar = {
        let path = state.sidecar_path.lock().unwrap();
        path.clone().ok_or("Sidecar binary not found")?
    };

    let mut cmd = std::process::Command::new(&sidecar);
    cmd.args(["render"])
        .stdin(std::process::Stdio::piped())
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped());

    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(0x08000000); // CREATE_NO_WINDOW
    }

    let mut child = cmd
        .spawn()
        .map_err(|e| format!("Failed to spawn render: {}", e))?;

    // Write markdown to stdin.
    if let Some(mut stdin) = child.stdin.take() {
        use std::io::Write;
        stdin
            .write_all(markdown.as_bytes())
            .map_err(|e| format!("Writing to stdin: {}", e))?;
        // stdin is dropped here, closing the pipe
    }

    let output = child
        .wait_with_output()
        .map_err(|e| format!("Waiting for render: {}", e))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        return Err(if stderr.is_empty() {
            "Render failed".to_string()
        } else {
            stderr
        });
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    serde_json::from_str(stdout.trim()).map_err(|e| format!("Parsing render output: {}", e))
}

/// Run `coderoo import obsidian` to import an Obsidian vault.
#[tauri::command]
pub async fn import_obsidian(
    app: tauri::AppHandle,
    vault_path: String,
    collection: Option<String>,
) -> Result<String, String> {
    let state = app.state::<AppState>();
    let project_dir = {
        let pd = state.project_dir.lock().unwrap();
        pd.as_ref()
            .ok_or("No project open")?
            .to_string_lossy()
            .to_string()
    };

    // Build the content dir from the project dir.
    let content_dir = {
        let cfg = state.config.lock().unwrap();
        let content_sub = cfg
            .as_ref()
            .and_then(|c| c.get("content"))
            .and_then(|c| c.get("dir"))
            .and_then(|v| v.as_str())
            .unwrap_or("content");
        std::path::PathBuf::from(&project_dir).join(content_sub)
    };

    let shell = app.shell();
    let mut args = vec![
        "import".to_string(),
        "obsidian".to_string(),
        vault_path,
        "--content".to_string(),
        content_dir.to_string_lossy().to_string(),
    ];
    if let Some(c) = collection {
        if !c.is_empty() {
            args.push("--collection".to_string());
            args.push(c);
        }
    }

    let str_args: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let output = shell
        .sidecar("coderoo")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?
        .args(&str_args)
        .output()
        .await
        .map_err(|e| format!("Failed to run import: {}", e))?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    } else {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        Err(if stderr.is_empty() {
            "Import failed".to_string()
        } else {
            stderr
        })
    }
}

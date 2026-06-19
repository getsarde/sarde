use std::sync::atomic::Ordering;

use crate::state::AppState;
use regex::Regex;
use tauri::Emitter;
use tauri::Manager;
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::CommandEvent;

#[derive(Clone, Debug, Default, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DesktopWarning {
    pub file: String,
    pub field: String,
    pub message: String,
    pub level: String,
}

#[derive(Clone, Debug, Default, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct BuildResult {
    pub status: String,
    pub summary: String,
    pub page_count: usize,
    pub duration: String,
    pub output_dir: String,
    pub warnings: Vec<DesktopWarning>,
    pub raw_output: String,
}

#[derive(Clone, Debug, Default, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ValidationResult {
    pub summary: String,
    pub warnings: Vec<DesktopWarning>,
    pub raw_output: String,
}

#[derive(Clone, Debug, Default, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DeployResult {
    pub provider: String,
    pub output: String,
    pub raw_output: String,
}

#[derive(Clone, Debug, Default, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ImportResult {
    pub notes_converted: usize,
    pub images_copied: usize,
    pub links_converted: usize,
    pub items_skipped: usize,
    pub raw_output: String,
}

/// Run `sarde build` on the current project. Streams stdout/stderr as Tauri events.
#[tauri::command]
pub async fn run_build(app: tauri::AppHandle, verbose: Option<bool>) -> Result<BuildResult, String> {
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
        .sidecar("sarde")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?;

    let mut args = vec!["build".to_string(), project_dir];
    if verbose.unwrap_or(false) {
        args.push("--verbose".to_string());
    }
    let str_args: Vec<&str> = args.iter().map(|s| s.as_str()).collect();

    let (mut rx, _child) = cmd
        .args(&str_args)
        .spawn()
        .map_err(|e| format!("Failed to spawn build: {}", e))?;

    let app_handle = app.clone();
    let _ = app_handle.emit("build:started", ());

    let mut stdout_lines = Vec::new();
    let mut stderr_lines = Vec::new();
    let mut exit_code = None;
    while let Some(event) = rx.recv().await {
        match event {
            CommandEvent::Stdout(line) => {
                let text = String::from_utf8_lossy(&line).trim().to_string();
                if !text.is_empty() {
                    stdout_lines.push(text.clone());
                    let _ = app_handle.emit("build:log", &text);
                    let _ = app_handle.emit("build:stdout", &text);
                }
            }
            CommandEvent::Stderr(line) => {
                let text = String::from_utf8_lossy(&line).trim().to_string();
                if !text.is_empty() {
                    stderr_lines.push(text.clone());
                    let _ = app_handle.emit("build:log", &text);
                    let _ = app_handle.emit("build:stderr", &text);
                }
            }
            CommandEvent::Terminated(status) => {
                let code = status.code.unwrap_or(-1);
                exit_code = Some(code);
                if code == 0 {
                    let result = parse_build_output(&stdout_lines.join("\n"));
                    let _ = app_handle.emit("build:complete", &result);
                } else {
                    let _ = app_handle.emit("build:error", &format!("Exit code: {}", code));
                }
            }
            _ => {}
        }
    }

    if exit_code.unwrap_or(0) != 0 {
        let stderr = stderr_lines.join("\n");
        return Err(if stderr.is_empty() {
            format!("Build failed with exit code {}", exit_code.unwrap_or(-1))
        } else {
            stderr
        });
    }

    Ok(parse_build_output(&stdout_lines.join("\n")))
}

/// Start the preview server: spawn `sarde dev`, parse port from stdout.
#[tauri::command]
pub async fn start_preview(app: tauri::AppHandle) -> Result<u16, String> {
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
        .sidecar("sarde")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?;

    // Hold lock across check-and-spawn to prevent double-spawn race.
    let mut child_guard = state.preview_child.lock().unwrap();
    if child_guard.is_some() {
        let port = *state.preview_port.lock().unwrap();
        return Ok(port);
    }

    let (mut rx, child) = cmd
        .args(["dev", &project_dir, "--port", "0"])
        .spawn()
        .map_err(|e| format!("Failed to spawn preview: {}", e))?;

    *child_guard = Some(child);
    drop(child_guard);

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

                    if !trimmed.is_empty() {
                        let _ = app_handle.emit("build:log", trimmed);
                    }
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
                    let stopping = state.preview_stopping.swap(false, Ordering::SeqCst);
                    let _ = app_handle.emit("preview:stopped", ());

                    let code = status.code.unwrap_or(-1);
                    if code != 0 && !stopping {
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
        state.preview_stopping.store(true, Ordering::SeqCst);
        let _ = c.kill();
    }
    *state.preview_port.lock().unwrap() = 0;
    Ok(())
}

/// Run `sarde validate` on the current project.
#[tauri::command]
pub async fn validate_project(app: tauri::AppHandle) -> Result<ValidationResult, String> {
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
        .sidecar("sarde")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?
        .args(["validate", &project_dir])
        .output()
        .await
        .map_err(|e| format!("Failed to run validate: {}", e))?;

    if output.status.success() {
        Ok(parse_validation_output(&String::from_utf8_lossy(&output.stdout)))
    } else {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        Err(if stderr.is_empty() {
            "Validation failed".to_string()
        } else {
            stderr
        })
    }
}

/// Run `sarde deploy` on the current project.
#[tauri::command]
pub async fn deploy(
    app: tauri::AppHandle,
    provider: Option<String>,
) -> Result<DeployResult, String> {
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
    let provider_override = provider.clone().unwrap_or_default();
    if let Some(p) = provider {
        if !p.is_empty() {
            args.push("--provider".to_string());
            args.push(p);
        }
    }

    let str_args: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let output = shell
        .sidecar("sarde")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?
        .args(&str_args)
        .output()
        .await
        .map_err(|e| format!("Failed to run deploy: {}", e))?;

    if output.status.success() {
        let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
        Ok(DeployResult {
            provider: parse_deploy_provider(&stdout, &provider_override, &state),
            output: stdout.clone(),
            raw_output: stdout,
        })
    } else {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        Err(if stderr.is_empty() {
            "Deploy failed".to_string()
        } else {
            stderr
        })
    }
}

/// Render markdown to HTML via `sarde render` (stdin → JSON stdout).
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

/// Run `sarde import obsidian` to import an Obsidian vault.
#[tauri::command]
pub async fn import_obsidian(
    app: tauri::AppHandle,
    vault_path: String,
    collection: Option<String>,
) -> Result<ImportResult, String> {
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
        .sidecar("sarde")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?
        .args(&str_args)
        .output()
        .await
        .map_err(|e| format!("Failed to run import: {}", e))?;

    if output.status.success() {
        Ok(parse_import_output(&String::from_utf8_lossy(&output.stdout)))
    } else {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        Err(if stderr.is_empty() {
            "Import failed".to_string()
        } else {
            stderr
        })
    }
}

fn parse_build_output(raw: &str) -> BuildResult {
    let mut result = BuildResult {
        status: "complete".into(),
        raw_output: raw.trim().to_string(),
        ..Default::default()
    };

    // Old format: "Built 15 pages in 45ms"
    let built_re = Regex::new(r"^Built\s+(\d+)\s+pages?\s+in\s+(.+)$").unwrap();
    // New format: "Built in 2086 ms"
    let built_in_re = Regex::new(r"^Built in (\d+) ms$").unwrap();
    // New table format: "  Pages            |   48"
    let pages_re = Regex::new(r"^\s*Pages\s+\|\s*(\d+)").unwrap();
    let warning_re = Regex::new(r"^\s+(.+?):\s+(.+)$").unwrap();

    let mut in_warnings = false;
    for line in raw.lines() {
        let trimmed = line.trim();

        // Old one-liner format.
        if let Some(caps) = built_re.captures(trimmed) {
            result.page_count = caps.get(1).and_then(|m| m.as_str().parse().ok()).unwrap_or(0);
            result.duration = caps.get(2).map(|m| m.as_str().to_string()).unwrap_or_default();
            if result.summary.is_empty() {
                result.summary = trimmed.to_string();
            }
        }
        // New table format — duration from footer.
        else if let Some(caps) = built_in_re.captures(trimmed) {
            result.duration = format!("{}ms", caps.get(1).map(|m| m.as_str()).unwrap_or("0"));
            if result.summary.is_empty() {
                result.summary = format!("Built {} pages", result.page_count);
            }
        }
        // New table format — page count from table row.
        else if let Some(caps) = pages_re.captures(trimmed) {
            result.page_count = caps.get(1).and_then(|m| m.as_str().parse().ok()).unwrap_or(0);
        } else if let Some(output) = trimmed.strip_prefix("Output:") {
            result.output_dir = output.trim().to_string();
        } else if trimmed.contains("warning(s):") {
            in_warnings = true;
        } else if in_warnings {
            if let Some(caps) = warning_re.captures(line) {
                result.warnings.push(DesktopWarning {
                    file: caps.get(1).map(|m| m.as_str().trim().to_string()).unwrap_or_default(),
                    field: "build".into(),
                    message: caps.get(2).map(|m| m.as_str().trim().to_string()).unwrap_or_default(),
                    level: "warning".into(),
                });
            }
        }
    }

    // Finalize summary.
    if result.summary.is_empty() {
        if result.page_count > 0 {
            result.summary = format!("Built {} pages", result.page_count);
        } else {
            result.summary = "Build complete".into();
        }
    }

    result
}

fn parse_validation_output(raw: &str) -> ValidationResult {
    let mut result = ValidationResult {
        raw_output: raw.trim().to_string(),
        ..Default::default()
    };
    let warning_re = Regex::new(r"^\s+(.+?):\s+\[(.*?)\]\s+(.+)$").unwrap();
    let fallback_warning_re = Regex::new(r"^\s+(.+?):\s+(.+)$").unwrap();
    let mut in_warnings = false;

    for line in raw.lines() {
        let trimmed = line.trim();
        if result.summary.is_empty() && trimmed.contains("validated in") {
            result.summary = trimmed.to_string();
        } else if trimmed.contains("warning(s):") {
            in_warnings = true;
        } else if in_warnings {
            if let Some(caps) = warning_re.captures(line) {
                result.warnings.push(DesktopWarning {
                    file: caps.get(1).map(|m| m.as_str().trim().to_string()).unwrap_or_default(),
                    field: caps.get(2).map(|m| m.as_str().trim().to_string()).unwrap_or_default(),
                    message: caps.get(3).map(|m| m.as_str().trim().to_string()).unwrap_or_default(),
                    level: "warning".into(),
                });
            } else if let Some(caps) = fallback_warning_re.captures(line) {
                result.warnings.push(DesktopWarning {
                    file: caps.get(1).map(|m| m.as_str().trim().to_string()).unwrap_or_default(),
                    field: "validation".into(),
                    message: caps.get(2).map(|m| m.as_str().trim().to_string()).unwrap_or_default(),
                    level: "warning".into(),
                });
            }
        }
    }

    if result.summary.is_empty() {
        result.summary = "Validation complete".into();
    }

    result
}

fn parse_deploy_provider(raw: &str, provider_override: &str, state: &AppState) -> String {
    if !provider_override.is_empty() {
        return provider_override.to_string();
    }
    for line in raw.lines() {
        if let Some(rest) = line.trim().strip_prefix("Deploying with ") {
            return rest.trim_end_matches("...").trim().to_string();
        }
    }
    state
        .config
        .lock()
        .unwrap()
        .as_ref()
        .and_then(|c| c.get("deploy"))
        .and_then(|d| d.get("provider"))
        .and_then(|p| p.as_str())
        .unwrap_or("configured provider")
        .to_string()
}

fn parse_import_output(raw: &str) -> ImportResult {
    let mut result = ImportResult {
        raw_output: raw.trim().to_string(),
        ..Default::default()
    };
    let done_re = Regex::new(
        r"Done:\s+(\d+)\s+notes converted,\s+(\d+)\s+images copied,\s+(\d+)\s+links converted(?:,\s+(\d+)\s+items skipped)?",
    )
    .unwrap();

    for line in raw.lines() {
        if let Some(caps) = done_re.captures(line) {
            result.notes_converted = caps.get(1).and_then(|m| m.as_str().parse().ok()).unwrap_or(0);
            result.images_copied = caps.get(2).and_then(|m| m.as_str().parse().ok()).unwrap_or(0);
            result.links_converted = caps.get(3).and_then(|m| m.as_str().parse().ok()).unwrap_or(0);
            result.items_skipped = caps.get(4).and_then(|m| m.as_str().parse().ok()).unwrap_or(0);
            break;
        }
    }

    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_new_build_table_format() {
        let output = r#"
Start building sites ...
sarde v0.1.0 windows/amd64

                   | Total
-------------------+-------
  Pages            |   48
  Paginator pages  |    0
  Collections      |    3
  Bundle assets    |    1
  Static files     |  106
  Processed images |    9
  Aliases          |    7
  Sitemaps         |    1

[sitemap] Generated sitemap.xml
[search] Built search index (48 pages)

Built in 2086 ms
  Output: dist/
"#;
        let result = parse_build_output(output);
        assert_eq!(result.status, "complete");
        assert_eq!(result.page_count, 48);
        assert_eq!(result.duration, "2086ms");
        assert_eq!(result.output_dir, "dist/");
        assert_eq!(result.summary, "Built 48 pages");
    }

    #[test]
    fn parses_old_build_one_liner() {
        let output = "Built 15 pages in 45ms\n  Output: dist/\n";
        let result = parse_build_output(output);
        assert_eq!(result.page_count, 15);
        assert_eq!(result.duration, "45ms");
        assert_eq!(result.output_dir, "dist/");
    }

    #[test]
    fn parses_validation_warnings() {
        let result = parse_validation_output(
            "3 pages across 1 collections validated in 2ms\n  1 warning(s):\n    docs/a.md: [title] Missing title\n",
        );
        assert_eq!(result.warnings.len(), 1);
        assert_eq!(result.warnings[0].file, "docs/a.md");
        assert_eq!(result.warnings[0].field, "title");
        assert_eq!(result.warnings[0].message, "Missing title");
    }

    #[test]
    fn parses_import_stats() {
        let result = parse_import_output(
            "Importing Obsidian vault from x -> y\nDone: 12 notes converted, 3 images copied, 4 links converted, 1 items skipped\n",
        );
        assert_eq!(result.notes_converted, 12);
        assert_eq!(result.images_copied, 3);
        assert_eq!(result.links_converted, 4);
        assert_eq!(result.items_skipped, 1);
    }
}

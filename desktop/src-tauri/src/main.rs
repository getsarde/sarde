#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::sync::Mutex;
use tauri::Manager;
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};

struct SidecarState {
    port: Mutex<u16>,
    ready: Mutex<bool>,
    child: Mutex<Option<CommandChild>>,
}

#[tauri::command]
fn get_sidecar_url(state: tauri::State<SidecarState>) -> Result<String, String> {
    let ready = *state.ready.lock().unwrap();
    if !ready {
        return Err("Sidecar not ready yet".into());
    }
    let port = *state.port.lock().unwrap();
    Ok(format!("http://localhost:{}", port))
}

#[tauri::command]
fn stop_sidecar(state: tauri::State<SidecarState>) -> Result<(), String> {
    let mut child = state.child.lock().unwrap();
    if let Some(c) = child.take() {
        let _ = c.kill();
    }
    *state.ready.lock().unwrap() = false;
    *state.port.lock().unwrap() = 0;
    Ok(())
}

#[tauri::command]
fn start_sidecar(app: tauri::AppHandle) -> Result<(), String> {
    let state = app.state::<SidecarState>();

    // Stop existing sidecar if running.
    {
        let mut child = state.child.lock().unwrap();
        if let Some(c) = child.take() {
            let _ = c.kill();
        }
        *state.ready.lock().unwrap() = false;
        *state.port.lock().unwrap() = 0;
    }

    let shell = app.shell();
    let sidecar_cmd = shell
        .sidecar("coderoo")
        .map_err(|e| format!("Failed to create sidecar: {}", e))?;

    // Spawn: coderoo sidecar --port 0
    let (mut rx, child) = sidecar_cmd
        .args(["sidecar", "--port", "0"])
        .spawn()
        .map_err(|e| format!("Failed to spawn sidecar: {}", e))?;

    *state.child.lock().unwrap() = Some(child);

    let app_handle = app.clone();

    tauri::async_runtime::spawn(async move {
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    let text = String::from_utf8_lossy(&line);
                    let trimmed = text.trim();
                    println!("[sidecar] {}", trimmed);

                    // Parse JSON: {"ready": true, "port": N}
                    if let Ok(json) = serde_json::from_str::<serde_json::Value>(trimmed) {
                        if json.get("ready").and_then(|v| v.as_bool()) == Some(true) {
                            if let Some(port) = json.get("port").and_then(|v| v.as_u64()) {
                                let port = port as u16;
                                let state = app_handle.state::<SidecarState>();
                                *state.port.lock().unwrap() = port;
                                *state.ready.lock().unwrap() = true;
                                println!("Sidecar ready on port {}", port);
                            }
                        }
                    }
                }
                CommandEvent::Stderr(line) => {
                    let text = String::from_utf8_lossy(&line);
                    eprint!("[sidecar err] {}", text);
                }
                CommandEvent::Terminated(status) => {
                    eprintln!("Sidecar terminated: {:?}", status);
                    let state = app_handle.state::<SidecarState>();
                    *state.ready.lock().unwrap() = false;
                }
                _ => {}
            }
        }
    });

    Ok(())
}

/// Windows Job Object — kills all children when this process exits.
#[cfg(windows)]
fn setup_job_object() {
    use windows_sys::Win32::System::JobObjects::*;
    use windows_sys::Win32::Foundation::*;

    unsafe {
        let job = CreateJobObjectW(std::ptr::null(), std::ptr::null());
        if job.is_null() { return; }

        let mut info: JOBOBJECT_EXTENDED_LIMIT_INFORMATION = std::mem::zeroed();
        info.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;

        let ret = SetInformationJobObject(
            job,
            JobObjectExtendedLimitInformation,
            &info as *const _ as *const _,
            std::mem::size_of::<JOBOBJECT_EXTENDED_LIMIT_INFORMATION>() as u32,
        );
        if ret == 0 { CloseHandle(job); return; }

        let current = windows_sys::Win32::System::Threading::GetCurrentProcess();
        if AssignProcessToJobObject(job, current) == 0 {
            CloseHandle(job);
            return;
        }
        // Leak the handle so it lives for the process lifetime
        let _ = job;
    }
}

#[cfg(not(windows))]
fn setup_job_object() {}

fn main() {
    setup_job_object();

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_process::init())
        .manage(SidecarState {
            port: Mutex::new(0),
            ready: Mutex::new(false),
            child: Mutex::new(None),
        })
        .invoke_handler(tauri::generate_handler![
            get_sidecar_url,
            start_sidecar,
            stop_sidecar,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod commands;
mod state;
mod yaml;

use state::AppState;
use tauri::Manager;

/// Windows Job Object — kills all children when this process exits.
#[cfg(windows)]
fn setup_job_object() {
    use windows_sys::Win32::Foundation::*;
    use windows_sys::Win32::System::JobObjects::*;

    unsafe {
        let job = CreateJobObjectW(std::ptr::null(), std::ptr::null());
        if job.is_null() {
            return;
        }

        let mut info: JOBOBJECT_EXTENDED_LIMIT_INFORMATION = std::mem::zeroed();
        info.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;

        let ret = SetInformationJobObject(
            job,
            JobObjectExtendedLimitInformation,
            &info as *const _ as *const _,
            std::mem::size_of::<JOBOBJECT_EXTENDED_LIMIT_INFORMATION>() as u32,
        );
        if ret == 0 {
            CloseHandle(job);
            return;
        }

        let current = windows_sys::Win32::System::Threading::GetCurrentProcess();
        if AssignProcessToJobObject(job, current) == 0 {
            CloseHandle(job);
            return;
        }
        // Leak the handle so it lives for the process lifetime.
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
        .manage(AppState::new())
        .setup(|app| {
            // Resolve sidecar binary path for std::process::Command usage.
            // Tauri strips the target triple when copying externalBin to the output dir,
            // so the binary is just "coderoo[.exe]" next to our exe.
            let state = app.state::<AppState>();
            let target_triple = env!("TAURI_ENV_TARGET_TRIPLE", "x86_64-pc-windows-msvc");
            let ext = if cfg!(windows) { ".exe" } else { "" };

            if let Ok(exe_dir) = std::env::current_exe().and_then(|p| {
                p.parent()
                    .map(|d| d.to_path_buf())
                    .ok_or(std::io::Error::new(std::io::ErrorKind::NotFound, "no parent"))
            }) {
                // Tauri copies externalBin as "coderoo[.exe]" (triple stripped)
                let candidates = [
                    exe_dir.join(format!("coderoo{}", ext)),
                    exe_dir.join(format!("coderoo-{}{}", target_triple, ext)),
                ];
                for candidate in &candidates {
                    if candidate.exists() {
                        *state.sidecar_path.lock().unwrap() = Some(candidate.clone());
                        break;
                    }
                }
            }
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            // Project lifecycle
            commands::project::open_project,
            commands::project::create_project,
            commands::project::close_project,
            commands::project::get_project_info,
            commands::project::list_recent_projects,
            // Content CRUD
            commands::content::list_content,
            commands::content::read_content,
            commands::content::save_content,
            commands::content::create_content,
            commands::content::delete_content,
            commands::content::rename_content,
            // Config & schema
            commands::config::get_config,
            commands::config::update_config,
            commands::config::get_collections,
            commands::config::get_schema,
            commands::config::create_collection,
            commands::config::delete_collection,
            // Build & preview
            commands::build::run_build,
            commands::build::start_preview,
            commands::build::stop_preview,
            commands::build::validate_project,
            commands::build::deploy,
            commands::build::import_obsidian,
            commands::build::render_markdown,
            // Assets
            commands::assets::asset_list,
            commands::assets::asset_upload,
            commands::assets::asset_delete,
            commands::assets::asset_get_thumbnail,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

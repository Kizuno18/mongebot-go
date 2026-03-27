// MongeBot Tauri application entry point.
// The Go backend runs as a sidecar process managed by Tauri.

use tauri::Manager;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .setup(|app| {
            // Spawn the Go sidecar backend
            let sidecar = app
                .shell()
                .sidecar("mongebot")
                .expect("failed to find mongebot sidecar binary");

            let (mut _rx, _child) = sidecar
                .args(["--mode", "sidecar", "--port", "9800"])
                .spawn()
                .expect("failed to spawn mongebot sidecar");

            // Log sidecar output in debug builds
            #[cfg(debug_assertions)]
            {
                tauri::async_runtime::spawn(async move {
                    use tauri_plugin_shell::process::CommandEvent;
                    while let Some(event) = _rx.recv().await {
                        match event {
                            CommandEvent::Stdout(line) => {
                                println!("[sidecar:stdout] {}", String::from_utf8_lossy(&line));
                            }
                            CommandEvent::Stderr(line) => {
                                eprintln!("[sidecar:stderr] {}", String::from_utf8_lossy(&line));
                            }
                            _ => {}
                        }
                    }
                });
            }

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running MongeBot");
}

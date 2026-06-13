# Wails + Go + Svelte — GestorCuentas

## Que es Wails

- Framework que combina backend Go con frontend web (HTML/CSS/JS)
- A diferencia de Electron, usa el WebView2 nativo del SO, no Chromium
- Binarios de ~10 MB, consumo de RAM de un solo digito

## Arquitectura (Python vs Wails/Go)

| Componente Python | Componente Wails |
|------------------|------------------|
| `ui/app_window.py` (tkinter) | `frontend/src/App.svelte` (HTML+CSS+JS) |
| `core/database.py` (sqlite3) | `backend/database.go` (go-sqlite3, structs tipados) |
| `core/security.py` (cryptography) | `backend/security.go` (AES-256-GCM + Argon2id nativo) |

## Requisitos e Instalacion

1. Instalar Go desde [go.dev/dl/](https://go.dev/dl/) (.msi)
2. Instalar Node.js desde [nodejs.org](https://nodejs.org/) (LTS)
3. Cerrar y reabrir VS Code para que reconozca las variables de entorno
4. Instalar Wails CLI:
   ```
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

## Comandos Principales

| Modo | Comando |
|------|---------|
| Desarrollo (live reload) | `wails dev` |
| Pre-Produccion (.exe final) | `wails build` |

El binario producido se encuentra en `/build/bin/GestorCuentas.exe`. Es autonomo, no requiere dependencias externas.

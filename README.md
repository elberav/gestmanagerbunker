# 🚀 ¿Qué diablos es Wails?

Imagina a Wails como el hijo perfecto entre el rendimiento brutal de Go y los diseños hermosos de la web moderna.

**Electron** (que es lo que usan Discord, Spotify o VS Code) utiliza Chromium para mostrar las pantallas. Esto hace que sus aplicaciones consuman cientos de megabytes de RAM solo por existir.
**Wails** resolvió esto de una forma brillante:
1. Toda tu Lógica Pesada (Base de datos, Cifrado, APIs) corre en **GO**. Es ultra-rápido, consume un solo dígito de RAM y se compila a código nativo (`.exe`).
2. Toda tu Interfaz Gráfica (Pintar botones, animaciones, colores) corre en el WebView2 nativo de Windows. Usa **HTML, CSS y Javascript (React/Svelte)**. 

El resultado es un ejecutable minúsculo de ~10 MB que se ve y mueve como una app AAA multimillonaria, sin comerte toda la memoria de la PC o exigir instalar Pythons o Módulos pesados al cliente final.

---

## ⚙️ Diferencias Arquitectónicas (Python vs Wails/Go)

### 1. `ui/app_window.py` (Python) ➡️ `<App />` (Svelte/React/HTML)
En lugar de escribir líneas como `tk.Label(bg="#212529")`, en Wails esto es CSS puro y etiquetas web. Todas las animaciones o colores oscuros se hacen con estilos CSS que la tarjeta gráfica (GPU) anima automáticamente a 60~120 fps de manera fluida y suave.

### 2. `core/database.py` ➡️ `backend/database.go`
No usamos SQLite3 en Python, usamos el driver `go-sqlite3`. Como Go es estrictamente tipado, las "cuentas" son `structs` ultra seguros desde su misma naturaleza. Una vez creadas, nadie en el frontend (Javascript) las puede corromper porque vienen esterilizadas por Go.

### 3. `core/security.py` ➡️ `backend/security.go`
Go fue escrito por criptógrafos de Google, por lo tanto toda la lógica de AES (Fernet) y PBKDF2 vienen de forma nativa en su paquete `crypto/`. No dependemos de librerías extrañas externas de terceros como en Python.

---

## 🛠️ Guía Paso a Paso: Cómo Instalar y Ejecutar el Proyecto en Windows

Dado que Go y Wails generan ejecutables nativos hiper-optimizados, necesitan los compiladores subyacentes (Solo se instalan una vez).

### Paso 1: Instalar los Motores (Requisitos Previos)
1. **Descargar e Instalar Go**: Entra a [go.dev/dl/](https://go.dev/dl/) y descarga el instalador para Windows (`.msi`). Siguiente, Siguiente...
2. **Descargar e Instalar Node.js**: Entra a [nodejs.org](https://nodejs.org/) e instala la versión LTS. (Obligatorio para que empaquete tu CSS/JS invisiblemente y se vea ultra-renderizado).

> ⚠️ **CRÍTICO**: Tras instalar ambos, **cierra el VS Code por completo y vuelve a abrirlo**. Si la terminal sigue abierta, no captará el nuevo comando `go` instalado en tus Variables de Entorno.

### Paso 2: Instalar el CLI de Wails (Orquestador)
Ahora abre una terminal nueva aquí y ejecuta:
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Paso 3: Inicializar la Arquitectura Svelte
Tu código actual aquí es un *Boceto Migrado* (Una platilla generada por Wails que hace el trabajo sucio de crear la estructura de carpetas y archivos) para que analices el lenguaje. Para generar todo el tramado complejo de Frontend (cientos de librerías Svelte), sal a tu carpeta raíz y haz que Wails construya la verdadera espina dorsal del proyecto:
```bash
wails init -n GestorCuentas -t svelte
```
*(Después solo reemplazas sus archivos `.go` de demostración con los `backend/database.go` y `security.go` blindados que ves aquí, y modificas el HTML para tus botones).*

### Paso 4: ¿Cómo Programarlo y Compilarlo? (Los 2 Comandos Clave)

✅ **Modo Desarrollo (Live Reload Visual)**  
Si en Python usabas `python main.py`, aquí **te metes a la carpeta del proyecto** `GestorCuentas` y corres:
```bash
wails dev
```
*(Se abrirá tu ventana de software. Si guardas un color distinto en el CSS o editas tu `backend.go`, la pantalla cambiará instantáneamente como si fuera una web re-cargándose. Es un sueño programar así).*

✅ **Modo Ensamblaje / Producción (Lanzamiento Universitario/Empresarial)**  
Cuando sientas que tu software o Bóveda de contraseñas es perfecta y está libre de errores, corres:
```bash
wails build
```
*(Para entender lo que hace este comando en la vida real, imagina una licuadora gigante. Wails agarra toda tu hermosa pantalla Svelte (HTML/CSS), agarra tu complicado código Backend de Go, y lo tritura todo hasta fundirlo en un único archivo ejecutable `GestorCuentas.exe`. 

A diferencia de Python donde debes llevar carpetas llenas de código `.py` desprotegido para que tu programa funcione en otra PC, Wails empaca y sella herméticamente todo en un solo bloque de 10 Megabytes ubicado en `/build/bin/`. Cualquiera que intente abrir tu `.exe` solo verá ceros y unos sin sentido, protegiendo tus algoritmos de mirones. 

Nota extra: Si añades las palabras `-clean -upx` al final del comando, Wails usará un compresor ultra-potente llamado UPX para que tu `.exe` final baje su peso hasta los 5 o 3 Megabytes, siendo súper portátil por correo).*


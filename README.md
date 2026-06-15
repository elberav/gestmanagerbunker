![image](https://raw.githubusercontent.com/elberav/gestmanagerbunker/refs/heads/master/build/example.jpeg)
# GestorCuentas - Gestor de Contraseñas

**Gestor de contraseñas seguro** con cifrado AES-256-GCM y clave maestra.

## 📦 Descargas

| Sistema | Archivo | Requisitos |
|---------|---------|-------------|
| Linux   | [⬇️ GestorCuentas versiones](https://github.com/elberav/gestmanagerbunker/releases/latest) | Linux (portable)|
| Windows | [⬇️ GestorCuentas versiones](https://github.com/elberav/gestmanagerbunker/releases/latest) | Windows (WebView2) |

---

## 🚀 Cómo Usar

### Windows
1. Descarga `GestorCuentas.exe`
2. Doble clic para ejecutar
3. Crea tu contraseña maestra
4. **Guarda el código de recuperación** en un lugar seguro

### Linux
1. Descarga `gestor-cuentas`
2. Asegúrate de tener WebKit2GTK 4.1 instalado:
   ```bash
   # Debian 12+ / Ubuntu 22.04+
   sudo apt install libwebkit2gtk-4.1-0

   # Fedora 37+
   sudo dnf install webkit2gtk4.1

   # Arch
   sudo pacman -S webkit2gtk

   # openSUSE
   sudo zypper install libwebkit2gtk-4_1-0
   ```
3. Dale permisos de ejecución:
   ```bash
   chmod +x ManagAccount-v1.2
4. Ejecuta:
   ```bash
   ./ManagAccount-v1.2

🔐 Seguridad
Cifrado: AES-256-GCM
Derivación de clave: Llave binaria usando Argon2id
Recuperación: Código de emergencia (28 caracteres)
Almacenamiento: SQLite local - offline

🌐 Idiomas
Español
English

📝 Notas
El código de recuperación NO se puede recuperar - guardalo bien
Si olvidas tu contraseña maestra, solo el código de recuperación puede acceder a tus datos
Los datos se guardan localmente.

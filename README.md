# GestorCuentas - Gestor de Contraseñas

**Gestor de contraseñas seguro** con cifrado AES-256-GCM y clave maestra.

## 📦 Descargas

| Sistema | Archivo | Requisitos |
|---------|---------|-------------|
| Windows | [⬇️ ManagAccount-v1.0.exe](https://github.com/elberav/gestmanagerbunker/releases/download/dsoejecutables/ManagAccount-v1.0.exe) | Windows 10/11 (WebView2) - Portable|
| Linux | [⬇️ ManagAccount-v1.0](https://github.com/elberav/gestmanagerbunker/releases/download/dsoejecutables/ManagAccount-v1.0) | Cualquier distro Linux - Portable|

---

## 🚀 Cómo Usar

### Windows
1. Descarga `GestorCuentas.exe`
2. Doble clic para ejecutar
3. Crea tu contraseña maestra
4. **Guarda el código de recuperación** en un lugar seguro

### Linux
1. Descarga `gestor-cuentas`
2. Dale permisos de ejecución:
   ```bash
   chmod +x gestor-cuentas
3. Ejecuta:
   ```bash
   ./gestor-cuentas

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
Los datos se guardan localmente en cuenta_service_v1.db

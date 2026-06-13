package main

import (
	"GestorCuentas/backend"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const AppVersion = "v1.0.0"

type UpdateInfo struct {
	HasUpdate     bool   `json:"has_update"`
	LatestVersion string `json:"latest_version"`
	DownloadURL   string `json:"download_url"`
	AssetName     string `json:"asset_name"`
}

type App struct {
	ctx context.Context

	clipMu     sync.Mutex
	clipCancel context.CancelFunc

	updateMu    sync.Mutex
	updateCache *UpdateInfo
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	backend.ConnectDB()
}

func (a *App) shutdown(ctx context.Context) {
	backend.Lock()
	backend.CloseDB()
}

// Call_CopyToClipboard copia texto al portapapeles del sistema operativo
// y programa su limpieza automática después de 45 segundos.
func (a *App) Call_CopyToClipboard(text string) {
	runtime.ClipboardSetText(a.ctx, text)

	a.clipMu.Lock()
	if a.clipCancel != nil {
		a.clipCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.clipCancel = cancel
	a.clipMu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(45 * time.Second):
			current, _ := runtime.ClipboardGetText(a.ctx)
			if current == text {
				runtime.ClipboardSetText(a.ctx, "")
			}
		}
	}()
}

func (a *App) Call_GetAccounts(status string, parentID *int) []backend.Account {
	return backend.GetAccounts(status, parentID)
}

func (a *App) Call_GetAllAccountsForDropdown() []backend.Account {
	// Obtenemos solo activas para no emparejar archivos con cuentas recicladas
	// Mandando parentID nulo pero en modo recursivo de búsqueda global (status 'active')
	// Wait, backend.GetAccounts("active", nil) returns only root accounts?
	// Let's use backend.SearchAccounts("", "active") to get ALL active accounts at any depth!
	return backend.SearchAccounts("", "active")
}

func (a *App) Call_GetAllChildren(accountID int) []backend.Account {
	return backend.GetAllChildren(accountID)
}

func (a *App) Call_SearchAccounts(query, status string) []backend.Account {
	if backend.GetMasterKey() == nil {
		return nil
	}
	return backend.SearchAccounts(query, status)
}

func (a *App) Call_AddAccount(name, email, loginMethod string, password string, parentID *int, tag, username, url string, notes string) (int, error) {
	if backend.GetMasterKey() == nil {
		return 0, backend.ErrLocked
	}
	
	// Trituradora para datos de cuenta
	pwBytes := []byte(password)
	ntBytes := []byte(notes)
	defer func() {
		for i := range pwBytes { pwBytes[i] = 0 }
		for i := range ntBytes { ntBytes[i] = 0 }
	}()

	var passwordEnc []byte
	if len(pwBytes) > 0 {
		encBytes, _ := backend.EncryptData(string(pwBytes))
		passwordEnc = encBytes
	}
	var notesEnc []byte
	if len(ntBytes) > 0 {
		encBytes, _ := backend.EncryptData(string(ntBytes))
		notesEnc = encBytes
	}
	return backend.AddAccount(name, email, loginMethod, passwordEnc, parentID, tag, username, url, notesEnc)
}

func (a *App) Call_UpdateAccount(accountID int, name, email, loginMethod string, password string, tag, username, url string, notes string) error {
	if backend.GetMasterKey() == nil {
		return backend.ErrLocked
	}

	// Trituradora para datos de cuenta
	pwBytes := []byte(password)
	ntBytes := []byte(notes)
	defer func() {
		for i := range pwBytes { pwBytes[i] = 0 }
		for i := range ntBytes { ntBytes[i] = 0 }
	}()

	var passwordEnc []byte
	if len(pwBytes) > 0 {
		encBytes, _ := backend.EncryptData(string(pwBytes))
		passwordEnc = encBytes
	}
	var notesEnc []byte
	if len(ntBytes) > 0 {
		encBytes, _ := backend.EncryptData(string(ntBytes))
		notesEnc = encBytes
	}
	return backend.UpdateAccount(accountID, name, email, loginMethod, passwordEnc, tag, username, url, notesEnc)
}

func (a *App) Call_UpdateLastViewed(accountID int) {
	if backend.GetMasterKey() != nil {
		backend.UpdateLastViewed(accountID)
	}
}

func (a *App) Call_UpdateStatus(accountID int, newStatus string) {
	if backend.GetMasterKey() != nil {
		backend.UpdateStatus(accountID, newStatus)
	}
}

func (a *App) Call_RestoreAccount(accountID int) {
	if backend.GetMasterKey() != nil {
		backend.RestoreAccount(accountID)
	}
}

func (a *App) Call_DeletePermanently(accountID int) {
	if backend.GetMasterKey() != nil {
		backend.DeletePermanently(accountID)
	}
}

func (a *App) Call_GetAccountByID(accountID int) *backend.Account {
	if backend.GetMasterKey() == nil {
		return nil
	}
	return backend.GetAccountByID(accountID)
}

func (a *App) Call_CheckEmergencyConfig() bool {
	return backend.HasRecoveryConfig()
}

// BÓVEDA DE ARCHIVOS

func (a *App) Call_UploadFile(accountID *int, filename string, base64Data string, tag string, comment string) string {
	rawBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "Error al decodificar el archivo"
	}
	if len(rawBytes) > 5*1024*1024 { // Límite de 5MB
		return "El archivo excede el límite de 5MB"
	}
	encBytes, err := backend.EncryptBytes(rawBytes)
	if err != nil {
		return "Error al cifrar el archivo"
	}
	backend.AddSecureFile(accountID, filename, len(rawBytes), encBytes, tag, comment)
	return "" // Todo bien
}

func (a *App) Call_GetAllFiles() []backend.SecureFile {
	return backend.GetAllSecureFiles()
}

func (a *App) Call_UpdateFile(fileID int, accountID *int, tag string, comment string) {
	backend.UpdateSecureFile(fileID, accountID, tag, comment)
}

// Retorna [filename, base64Data] o arreglos vacíos si falla. (Wails solo puede retornar multiples con wrappers o arrays/structs, vamos a usar un struct temporal simple o devolver un arreglo temporal)
// Mejor devolvemos arreglo: []string{filename, base64Data}
// Call_SaveFileToDisk descarga un archivo usando el diálogo nativo del sistema operativo.
// Esto es mucho más robusto para Linux que el método de etiquetas <a>.
func (a *App) Call_SaveFileToDisk(fileID int) string {
	filename, dataEnc := backend.GetSecureFile(fileID)
	if filename == "" || len(dataEnc) == 0 {
		return "Archivo no encontrado"
	}

	// Abrir diálogo de guardar
	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Guardar archivo seguro",
		DefaultFilename: filename,
	})
	if err != nil {
		return err.Error()
	}
	if selection == "" {
		return "" // Cancelado por el usuario
	}

	// Desencriptar y escribir
	rawBytes, err := backend.DecryptBytes(dataEnc)
	if err != nil {
		return "Error al desencriptar el archivo"
	}

	err = os.WriteFile(selection, rawBytes, 0644)
	
	// Limpieza de memoria (Punto 10 auditoría)
	for i := range rawBytes { rawBytes[i] = 0 }

	if err != nil {
		return "Error al escribir el archivo en disco"
	}

	return "SUCCESS"
}

func (a *App) Call_DownloadFile(fileID int) []string {
	filename, dataEnc := backend.GetSecureFile(fileID)
	if filename == "" || len(dataEnc) == 0 {
		return []string{"", ""}
	}
	rawBytes, err := backend.DecryptBytes(dataEnc)
	if err != nil {
		return []string{"", ""}
	}
	
	b64 := base64.StdEncoding.EncodeToString(rawBytes)
	
	// Limpieza de memoria (Punto 10 auditoría)
	for i := range rawBytes { rawBytes[i] = 0 }
	
	return []string{filename, b64}
}

func (a *App) Call_DeleteSecureFile(fileID int) {
	backend.DeleteSecureFile(fileID)
}

func (a *App) Call_IsFirstRun() bool {
	return backend.IsFirstRun()
}

func (a *App) Call_SetupMaster(password string) (string, error) {
	return backend.SetupMasterPassword(password)
}

func (a *App) Call_Unlock(password string) bool {
	unlocked := backend.Unlock(password)
	if unlocked {
		backend.MigrateEncryptFields()
	}
	return unlocked
}

func (a *App) Call_LockSession() {
	backend.Lock()
}

// Call_GetLockoutStatus devuelve [intentos, segundosRestantes]
func (a *App) Call_GetLockoutStatus() []int {
	attempts, seconds := backend.GetLockoutStatus()
	return []int{attempts, seconds}
}

func (a *App) Call_RecoverWithCode(code string, newPassword string) (string, error) {
	return backend.RecoverWithCode(code, newPassword)
}

func (a *App) Call_ChangeMasterPassword(currentPassword string, newPassword string) (string, error) {
	return backend.ChangeMasterPassword(currentPassword, newPassword)
}

func (a *App) Call_Decrypt(payload []byte) (string, error) {
	if backend.GetMasterKey() == nil {
		return "", backend.ErrLocked
	}
	return backend.DecryptGlobal(payload)
}

func (a *App) Call_CheckUpdate() UpdateInfo {
	a.updateMu.Lock()
	if a.updateCache != nil {
		cache := *a.updateCache
		a.updateMu.Unlock()
		return cache
	}
	a.updateMu.Unlock()

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/elberav/gestmanagerbunker/releases/latest", nil)
	if err != nil {
		return UpdateInfo{}
	}
	req.Header.Set("User-Agent", "GestorCuentas/"+AppVersion)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return UpdateInfo{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{}
	}

	body, _ := io.ReadAll(resp.Body)
	var ghResp struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if json.Unmarshal(body, &ghResp) != nil || ghResp.TagName == "" {
		return UpdateInfo{}
	}

	info := UpdateInfo{
		HasUpdate:     compareSemver(ghResp.TagName, AppVersion) > 0,
		LatestVersion: ghResp.TagName,
		DownloadURL:   ghResp.HTMLURL,
	}

	if info.HasUpdate {
		downloadURL, assetName := findOSAsset(ghResp.Assets)
		if downloadURL != "" {
			info.DownloadURL = downloadURL
			info.AssetName = assetName
		}
	}

	a.updateMu.Lock()
	a.updateCache = &info
	a.updateMu.Unlock()

	return info
}

func findOSAsset(assets []struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}) (string, string) {
	goos := goruntime.GOOS
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if goos == "windows" && strings.HasSuffix(name, ".exe") {
			return asset.BrowserDownloadURL, asset.Name
		}
		if goos == "linux" && !strings.HasSuffix(name, ".exe") && !strings.HasSuffix(name, ".msi") {
			return asset.BrowserDownloadURL, asset.Name
		}
	}
	return "", ""
}

// compareSemver returns 1 if a > b, -1 if a < b, 0 if equal
func compareSemver(a, b string) int {
	va := parseVersion(a)
	vb := parseVersion(b)
	for i := 0; i < 3; i++ {
		if va[i] < vb[i] {
			return -1
		}
		if va[i] > vb[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var res [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, _ := strconv.Atoi(parts[i])
		res[i] = n
	}
	return res
}

func (a *App) Call_DownloadUpdate(url string) string {
	if url == "" {
		return "URL de descarga no disponible"
	}

	exePath, err := os.Executable()
	if err != nil {
		return "No se pudo determinar la ruta del ejecutable"
	}
	exeDir := filepath.Dir(exePath)
	newPath := filepath.Join(exeDir, "GestorCuentas_new")
	if goruntime.GOOS == "windows" {
		newPath += ".exe"
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "Error al descargar: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "Error del servidor: " + resp.Status
	}

	out, err := os.Create(newPath)
	if err != nil {
		return "Error al crear archivo: " + err.Error()
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(newPath)
		return "Error al escribir archivo: " + err.Error()
	}

	if written < 1024*1024 {
		os.Remove(newPath)
		return "Archivo demasiado pequeno, posible error de descarga"
	}

	return "SUCCESS"
}

func (a *App) Call_ApplyUpdate() string {
	exePath, err := os.Executable()
	if err != nil {
		return "No se pudo determinar la ruta del ejecutable"
	}
	exeDir := filepath.Dir(exePath)
	newPath := filepath.Join(exeDir, "GestorCuentas_new")
	origPath := exePath

	if goruntime.GOOS == "windows" {
		newPath += ".exe"
		batPath := filepath.Join(exeDir, "update.bat")
		batContent := fmt.Sprintf(
			"@echo off\r\n"+
				"timeout /t 2 /nobreak >nul\r\n"+
				"copy /y \"%s\" \"%s\" >nul\r\n"+
				"del \"%s\"\r\n"+
				"start \"\" \"%s\"\r\n",
			newPath, origPath, newPath, origPath,
		)
		if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
			return "Error al crear script de actualizacion"
		}
		cmd := exec.Command("cmd", "/c", batPath)
		cmd.Start()
	} else {
		shPath := filepath.Join(exeDir, "update.sh")
		shContent := fmt.Sprintf(
			"#!/bin/bash\n"+
				"sleep 2\n"+
				"cp \"%s\" \"%s\"\n"+
				"rm \"%s\"\n"+
				"\"%s\" &\n",
			newPath, origPath, newPath, origPath,
		)
		if err := os.WriteFile(shPath, []byte(shContent), 0755); err != nil {
			return "Error al crear script de actualizacion"
		}
		cmd := exec.Command("bash", shPath)
		cmd.Start()
	}

	backend.Lock()
	backend.CloseDB()
	os.Exit(0)
	return ""
}

func (a *App) Call_ExportAccounts() string {
	if backend.GetMasterKey() == nil {
		return backend.ErrLocked.Error()
	}

	roots := backend.GetAccounts("active", nil)
	var sb strings.Builder

	sb.WriteString("# Exportación de Cuentas - GestorCuentas\n")
	sb.WriteString("# Generado: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")

	for i, acc := range roots {
		a.writeAccountMD(&sb, &acc, 2)
		if acc.HasChildren {
			children := backend.GetAllChildren(acc.ID)
			for _, child := range children {
				a.writeAccountMD(&sb, &child, 3)
			}
		}
		if i < len(roots)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n---\n*Exportado desde GestorCuentas*\n")

	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Exportar cuentas",
		DefaultFilename: "cuentas_export.md",
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown (*.md)", Pattern: "*.md"},
		},
	})
	if err != nil {
		return "Error al abrir el diálogo de guardar"
	}
	if selection == "" {
		return "" // Cancelado por el usuario
	}

	err = os.WriteFile(selection, []byte(sb.String()), 0644)
	if err != nil {
		return "Error al escribir el archivo"
	}

	return "SUCCESS"
}

func (a *App) writeAccountMD(sb *strings.Builder, acc *backend.Account, level int) {
	heading := strings.Repeat("#", level)
	sb.WriteString(fmt.Sprintf("%s %s\n", heading, acc.Name))

	a.writeFieldMD(sb, "Email", acc.Email)
	a.writeFieldMD(sb, "Usuario", acc.Username)

	if len(acc.PasswordEnc) > 0 {
		pw, err := backend.DecryptGlobal(acc.PasswordEnc)
		if err == nil {
			a.writeFieldMD(sb, "Contraseña", pw)
		}
	}

	a.writeFieldMD(sb, "URL", acc.URL)
	a.writeFieldMD(sb, "Tag", acc.Tag)
	a.writeFieldMD(sb, "Método", acc.LoginMethod)

	if len(acc.NotesEnc) > 0 {
		notes, err := backend.DecryptGlobal(acc.NotesEnc)
		if err == nil {
			a.writeFieldMD(sb, "Notas", notes)
		}
	}

	sb.WriteString("\n")
}

func (a *App) writeFieldMD(sb *strings.Builder, key, value string) {
	if value == "" {
		return
	}
	lines := strings.Split(value, "\n")
	if len(lines) <= 1 {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", key, value))
	} else {
		sb.WriteString(fmt.Sprintf("- **%s**:\n", key))
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}
}

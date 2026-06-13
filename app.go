package main

import (
	"GestorCuentas/backend"
	"context"
	"encoding/base64"
	"os"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	clipMu     sync.Mutex
	clipCancel context.CancelFunc
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

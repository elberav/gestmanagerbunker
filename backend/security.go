package backend

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/argon2"
)

var ErrLocked = errors.New("bóveda bloqueada")

const (
	saltSize     = 16
	keyLen       = 32
	verifyPhrase = "REGISTRO_CUENTAS_OK"

	// Argon2id — Parámetros de seguridad (resistente a GPU/ASIC)
	argonTime    = 3         // Pasadas sobre la memoria (OWASP recomienda ≥ 3)
	argonMemory  = 64 * 1024 // 64 MB de RAM obligatoria por intento
	argonThreads = 4         // Hilos de CPU en paralelo
)

var (
	secretsDir   = SecretsDirName
	saltFile     = filepath.Join(secretsDir, SaltFileName)
	verifyFile   = filepath.Join(secretsDir, VerifyFileName)
	recoveryFile = filepath.Join(secretsDir, RecoveryFileName)

	masterKey []byte
	salt      []byte

	// Rate limiting: 4 intentos, 5 minutos de bloqueo (Persistente en BD)
	failedAttempts int
	lockoutUntil   time.Time
)

// loadLockoutState lee el estado de bloqueo desde la BD
func loadLockoutState() {
	attempts, until := getLockoutFromDB()
	failedAttempts = attempts
	lockoutUntil = until
}

// saveLockoutState guarda el estado de bloqueo en la BD
func saveLockoutState() {
	setLockoutInDB(failedAttempts, lockoutUntil)
}

// GetLockoutStatus devuelve los intentos fallidos y los segundos restantes de bloqueo.
func GetLockoutStatus() (int, int) {
	if time.Now().Before(lockoutUntil) {
		timeLeft := int(time.Until(lockoutUntil).Seconds())
		return failedAttempts, timeLeft
	}
	return failedAttempts, 0
}

func SetMasterKey(key []byte) {
	masterKey = key
}

func GetMasterKey() []byte {
	return masterKey
}

func ensureConfigDir() {
	os.MkdirAll(secretsDir, 0700)
	HideFile(secretsDir)
}

func IsFirstRun() bool {
	if _, err := os.Stat(saltFile); os.IsNotExist(err) {
		return true
	}
	if _, err := os.Stat(verifyFile); os.IsNotExist(err) {
		return true
	}
	return false
}

func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, keyLen)
}

func HasRecoveryConfig() bool {
	_, err := os.Stat(recoveryFile)
	return !os.IsNotExist(err)
}

func generateRecoveryCode() string {
	// Caracteres sin ambigüedad visual (sin O/0, I/1, L)
	const chars = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	const charsetLen = len(chars) // 30
	// Rejection sampling: descartamos bytes que causarían sesgo de módulo.
	// 256 % 30 = 16, así que los bytes [240, 255] causan sesgo.
	// Solo aceptamos bytes < 240 (256 - 16).
	maxValid := byte(256 - (256 % charsetLen)) // 240
	code := make([]byte, 28)
	for i := range code {
		for {
			b := make([]byte, 1)
			rand.Read(b)
			if b[0] < maxValid {
				code[i] = chars[int(b[0])%charsetLen]
				break
			}
		}
	}
	return string(code[0:4]) + "-" + string(code[4:8]) + "-" + string(code[8:12]) + "-" + string(code[12:16]) + "-" + string(code[16:20]) + "-" + string(code[20:24]) + "-" + string(code[24:28])
}

func saveRecoveryBackup(recoveryCode string) error {
	recSalt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, recSalt); err != nil {
		return err
	}
	recoveryKey := DeriveKey(recoveryCode, recSalt)
	encKey, err := encryptData(recoveryKey, masterKey)
	if err != nil {
		return err
	}
	payload := append(recSalt, encKey...)
	return os.WriteFile(recoveryFile, payload, 0600)
}

func SetupMasterPassword(password string) (string, error) {
	// Trituradora: Convertir a bytes para poder borrar físicamente de la RAM
	pwBytes := []byte(password)
	defer func() {
		for i := range pwBytes { pwBytes[i] = 0 }
	}()

	if len(pwBytes) < 8 {
		return "", errors.New("la contraseña maestra debe tener al menos 8 caracteres")
	}
	if !IsFirstRun() {
		return "", errors.New("bóveda ya configurada")
	}
	ensureConfigDir()

	salt = make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}
	if err := os.WriteFile(saltFile, salt, 0600); err != nil {
		return "", err
	}

	masterKey = DeriveKey(string(pwBytes), salt)
	encToken, err := encryptData(masterKey, []byte(verifyPhrase))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(verifyFile, encToken, 0600); err != nil {
		return "", err
	}

	recoveryCode := generateRecoveryCode()
	if err := saveRecoveryBackup(recoveryCode); err != nil {
		return "", err
	}
	return recoveryCode, nil
}

func Unlock(password string) bool {
	// Trituradora de memoria
	pwBytes := []byte(password)
	defer func() {
		for i := range pwBytes { pwBytes[i] = 0 }
	}()

	if IsFirstRun() {
		return false
	}

	// Cargar estado persistente antes de verificar
	loadLockoutState()

	// Verificar si el usuario está bloqueado por tiempo
	if time.Now().Before(lockoutUntil) {
		return false
	}

	// Anti-Debugging
	if isBeingDebugged() {
		os.Exit(1)
	}

	var err error
	salt, err = os.ReadFile(saltFile)
	if err != nil {
		return false
	}

	tempKey := DeriveKey(string(pwBytes), salt)
	tokenRaw, err := os.ReadFile(verifyFile)
	if err != nil {
		return false
	}
	decrypted, err := decryptData(tempKey, tokenRaw)
	
	// Verificar éxito
	if err == nil && string(decrypted) == verifyPhrase {
		masterKey = tempKey
		failedAttempts = 0 // Resetear intentos al tener éxito
		saveLockoutState()
		go securityMonitor()
		return true
	}

	// Si falló, limpiar tempKey de la heap antes de salir
	for i := range tempKey {
		tempKey[i] = 0
	}
	failedAttempts++
	if failedAttempts >= 4 {
		lockoutUntil = time.Now().Add(5 * time.Minute)
	}
	saveLockoutState()

	return false
}

// isBeingDebugged se define en archivos separados por OS (security_linux.go / security_windows.go)

// securityMonitor vigila la integridad mientras la bóveda está abierta.
func securityMonitor() {
	for {
		if masterKey == nil {
			return
		}
		if isBeingDebugged() {
			Lock()
			os.Exit(1)
		}
		time.Sleep(2 * time.Second)
	}
}

func Lock() {
	// Blindaje de Memoria Nivel Experto:
	// Sobrescribimos hasta la CAPACIDAD total del slice, no solo su largo.
	// Esto asegura que no queden restos en el buffer subyacente de Go.
	if masterKey != nil {
		fullKey := masterKey[:cap(masterKey)]
		for i := range fullKey {
			fullKey[i] = 0
		}
	}
	if salt != nil {
		fullSalt := salt[:cap(salt)]
		for i := range fullSalt {
			fullSalt[i] = 0
		}
	}
	masterKey = nil
	salt = nil
}

// RecoverWithCode: Usa el código de recuperación para rescatar la masterKey antigua,
// re-encripta TODA la BD con la nueva clave, y genera un nuevo código.
func RecoverWithCode(code, newPassword string) (string, error) {
	// Trituradora para nueva clave
	newPwBytes := []byte(newPassword)
	defer func() {
		for i := range newPwBytes { newPwBytes[i] = 0 }
	}()

	if len(newPwBytes) < 8 {
		return "", errors.New("la nueva contraseña maestra debe tener al menos 8 caracteres")
	}

	// Cargar estado persistente
	loadLockoutState()

	// Rate limiting para recuperación
	if time.Now().Before(lockoutUntil) {
		return "", errors.New("demasiados intentos. Espera unos minutos")
	}

	payload, err := os.ReadFile(recoveryFile)
	if err != nil {
		return "", errors.New("no se encontró archivo de recuperación")
	}
	if len(payload) < saltSize+12 {
		return "", errors.New("archivo de recuperación inválido")
	}

	recSalt := payload[:saltSize]
	encKey := payload[saltSize:]

	recoveryKey := DeriveKey(code, recSalt)
	oldMasterKeyStr, err := decryptData(recoveryKey, encKey)
	if err != nil {
		for i := range recoveryKey { recoveryKey[i] = 0 }
		failedAttempts++
		if failedAttempts >= 4 {
			lockoutUntil = time.Now().Add(5 * time.Minute)
		}
		saveLockoutState()
		return "", errors.New("código de recuperación incorrecto")
	}
	oldMasterKey := []byte(oldMasterKeyStr)
	for i := range oldMasterKeyStr { oldMasterKeyStr[i] = 0 }
	for i := range recoveryKey { recoveryKey[i] = 0 }

	// Derivar nueva masterKey
	newSalt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, newSalt); err != nil {
		for i := range oldMasterKey { oldMasterKey[i] = 0 }
		return "", err
	}
	newMasterKey := DeriveKey(string(newPwBytes), newSalt)

	// Re-encriptar TODA la data de la BD (passwords, notas y metadata)
	if err := ReEncryptAllData(oldMasterKey, newMasterKey); err != nil {
		for i := range oldMasterKey { oldMasterKey[i] = 0 }
		return "", err
	}
	for i := range oldMasterKey { oldMasterKey[i] = 0 }

	// Actualizar globales
	masterKey = newMasterKey
	salt = newSalt

	os.WriteFile(saltFile, newSalt, 0600)

	encToken, err := encryptData(masterKey, []byte(verifyPhrase))
	if err != nil {
		return "", err
	}
	os.WriteFile(verifyFile, encToken, 0600)

	// Generar nuevo código de recuperación
	newCode := generateRecoveryCode()
	if err := saveRecoveryBackup(newCode); err != nil {
		return "", err
	}
	return newCode, nil
}

// ChangeMasterPassword: Cambia la contraseña estando ya autenticado.
// Requiere la contraseña actual para verificar la identidad del usuario.
func ChangeMasterPassword(currentPassword, newPassword string) (string, error) {
	// Trituradora para ambas claves
	curPwBytes := []byte(currentPassword)
	newPwBytes := []byte(newPassword)
	defer func() {
		for i := range curPwBytes { curPwBytes[i] = 0 }
		for i := range newPwBytes { newPwBytes[i] = 0 }
	}()

	if len(newPwBytes) < 8 {
		return "", errors.New("la nueva contraseña maestra debe tener al menos 8 caracteres")
	}

	if masterKey == nil {
		return "", errors.New("bóveda bloqueada")
	}

	// Verificar que la contraseña actual sea correcta
	currentSalt, err := os.ReadFile(saltFile)
	if err != nil {
		return "", errors.New("error al leer configuración de seguridad")
	}
	testKey := DeriveKey(string(curPwBytes), currentSalt)
	tokenRaw, err := os.ReadFile(verifyFile)
	if err != nil {
		for i := range testKey { testKey[i] = 0 }
		return "", errors.New("error al leer token de verificación")
	}
	decrypted, err := decryptData(testKey, tokenRaw)
	if err != nil || string(decrypted) != verifyPhrase {
		for i := range testKey { testKey[i] = 0 }
		return "", errors.New("contraseña actual incorrecta")
	}
	for i := range testKey { testKey[i] = 0 }

	oldMasterKey := make([]byte, len(masterKey))
	copy(oldMasterKey, masterKey)

	newSalt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, newSalt); err != nil {
		return "", err
	}
	newMasterKey := DeriveKey(newPassword, newSalt)

	if err := ReEncryptAllData(oldMasterKey, newMasterKey); err != nil {
		for i := range oldMasterKey { oldMasterKey[i] = 0 }
		return "", err
	}

	for i := range oldMasterKey { oldMasterKey[i] = 0 }
	masterKey = newMasterKey
	salt = newSalt

	os.WriteFile(saltFile, newSalt, 0600)

	encToken, err := encryptData(masterKey, []byte(verifyPhrase))
	if err != nil {
		return "", err
	}
	os.WriteFile(verifyFile, encToken, 0600)

	newCode := generateRecoveryCode()
	if err := saveRecoveryBackup(newCode); err != nil {
		return "", err
	}
	return newCode, nil
}

// EncryptData interactúa con strings
func EncryptData(text string) ([]byte, error) {
	if masterKey == nil {
		return nil, errors.New("bóveda bloqueada: sin llave maestra")
	}
	return encryptData(masterKey, []byte(text))
}

// DecryptData devuelve un string
func DecryptData(ciphertext []byte) (string, error) {
	if masterKey == nil {
		return "", errors.New("bóveda bloqueada: sin llave maestra")
	}
	pt, err := decryptData(masterKey, ciphertext)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// EncryptBytes interactúa con bytes raw (archivos)
func EncryptBytes(data []byte) ([]byte, error) {
	if masterKey == nil {
		return nil, errors.New("bóveda bloqueada: sin llave maestra")
	}
	return encryptData(masterKey, data)
}

// DecryptBytes devuelve bytes raw (archivos)
func DecryptBytes(ciphertext []byte) ([]byte, error) {
	if masterKey == nil {
		return nil, errors.New("bóveda bloqueada: sin llave maestra")
	}
	return decryptData(masterKey, ciphertext)
}

// Funciones internas base operan 100% sobre []byte
func encryptData(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decryptData(key []byte, payload []byte) ([]byte, error) {
	if len(payload) < 12 {
		return nil, errors.New("payload demasiado corto")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := payload[:nonceSize], payload[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// VerifyPassword verifica la contraseña maestra sin alterar el estado de la aplicación.
// Se usa para re-autenticación antes de operaciones sensibles como exportar.
func VerifyPassword(password string) bool {
	if masterKey == nil {
		return false
	}
	pwBytes := []byte(password)
	defer func() {
		for i := range pwBytes { pwBytes[i] = 0 }
	}()

	saltBytes, err := os.ReadFile(saltFile)
	if err != nil {
		return false
	}
	testKey := DeriveKey(string(pwBytes), saltBytes)
	defer func() {
		for i := range testKey { testKey[i] = 0 }
	}()

	tokenRaw, err := os.ReadFile(verifyFile)
	if err != nil {
		return false
	}
	decrypted, err := decryptData(testKey, tokenRaw)
	return err == nil && string(decrypted) == verifyPhrase
}

// DecryptGlobal decodifica como string para compatibilidad previa
func DecryptGlobal(payload []byte) (string, error) {
	if masterKey == nil {
		return "", errors.New("bóveda bloqueada: sin autorización")
	}
	if len(payload) == 0 {
		return "", nil
	}
	pt, err := decryptData(masterKey, payload)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

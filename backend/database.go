package backend

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Account struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	LoginMethod string  `json:"login_method"`
	PasswordEnc []byte  `json:"password_enc"`
	ParentID    *int    `json:"parent_id"`
	Status      string  `json:"status"`
	Tag         string  `json:"tag"`
	Username    string  `json:"username"`
	URL         string  `json:"url"`
	NotesEnc    []byte  `json:"notes_enc"`
	LastViewed  *string `json:"last_viewed"`
	HasChildren bool    `json:"has_children"`
}

type SecureFile struct {
	ID           int    `json:"id"`
	AccountID    *int   `json:"account_id"`
	AccountName  string `json:"account_name"`
	AccountEmail string `json:"account_email"`
	Filename     string `json:"filename"`
	FileSize     int    `json:"file_size"`
	UploadedAt   string `json:"uploaded_at"`
	Tag          string `json:"tag"`
	Comment      string `json:"comment"`
}

var db *sql.DB

func ConnectDB() {
	dir := DBDirName
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0700) // Blindaje: Solo propietario
		if err != nil {
			log.Fatal("No se pudo crear la carpeta de encriptación:", err)
		}
	}

	dbPath := filepath.Join(dir, DBFileName)
	backupDB(dbPath)

	var err error
	db, err = sql.Open("sqlite", "file:"+dbPath+"?cache=shared")
	if err != nil {
		log.Fatal("Error conectando a BD:", err)
	}

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys = ON")

	initDB()
}

func backupDB(dbPath string) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return
	}
	src, err := os.ReadFile(dbPath)
	if err != nil {
		return
	}
	bakPath := dbPath + ".bak"
	os.WriteFile(bakPath, src, 0600)
}

func CloseDB() {
	if db != nil {
		db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
		db.Close()
	}
}

func initDB() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS accounts (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			name         TEXT    NOT NULL,
			email        TEXT    NOT NULL,
			login_method TEXT    NOT NULL DEFAULT 'password',
			password_enc BLOB,
			parent_id    INTEGER,
			status       TEXT    NOT NULL DEFAULT 'active',
			FOREIGN KEY (parent_id) REFERENCES accounts(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		log.Fatal("Error crítico creando tabla accounts:", err)
	}

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		log.Fatal("Error leyendo PRAGMA user_version:", err)
	}

	if version < 1 {
		log.Println("[Migración] Actualizando BD a v1...")
		if _, err := db.Exec("ALTER TABLE accounts ADD COLUMN tag TEXT DEFAULT ''"); err != nil { log.Fatal(err) }
		if _, err := db.Exec("ALTER TABLE accounts ADD COLUMN username TEXT DEFAULT ''"); err != nil { log.Fatal(err) }
		if _, err := db.Exec("ALTER TABLE accounts ADD COLUMN url TEXT DEFAULT ''"); err != nil { log.Fatal(err) }
		db.Exec("PRAGMA user_version = 1")
		version = 1
	}
	if version < 2 {
		log.Println("[Migración] Actualizando BD a v2...")
		if _, err := db.Exec("ALTER TABLE accounts ADD COLUMN notes_enc BLOB DEFAULT NULL"); err != nil { log.Fatal(err) }
		if _, err := db.Exec("ALTER TABLE accounts ADD COLUMN last_viewed TEXT DEFAULT NULL"); err != nil { log.Fatal(err) }
		db.Exec("PRAGMA user_version = 2")
		version = 2
	}
	if version < 3 {
		log.Println("[Migración] Actualizando BD a v3...")
		query := `CREATE TABLE IF NOT EXISTS secure_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT, 
			account_id INTEGER, 
			filename TEXT, 
			file_size INTEGER, 
			data_enc BLOB, 
			uploaded_at TEXT DEFAULT (datetime('now', 'localtime')), 
			tag TEXT DEFAULT '', 
			comment TEXT DEFAULT '', 
			FOREIGN KEY(account_id) REFERENCES accounts(id) ON DELETE CASCADE)`
		if _, err := db.Exec(query); err != nil { log.Fatal(err) }
		db.Exec("PRAGMA user_version = 3")
		version = 3
	}
	if version < 4 {
		log.Println("[Migración] Actualizando BD a v4 (metadata encriptada)...")
		cols := []string{"name_enc", "email_enc", "username_enc", "url_enc", "tag_enc", "login_method_enc"}
		for _, col := range cols {
			if _, err := db.Exec("ALTER TABLE accounts ADD COLUMN " + col + " BLOB DEFAULT NULL"); err != nil {
				log.Printf("[Migración v4] Columna %s ya existe o error: %v", col, err)
			}
		}
		db.Exec("PRAGMA user_version = 4")
		version = 4
	}
}

func MigrateEncryptFields() {
	if masterKey == nil {
		return
	}
	rows, err := db.Query("SELECT id, name, email, login_method, tag, username, url FROM accounts WHERE name_enc IS NULL")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var n, e, lm, t, u, url string
		if err := rows.Scan(&id, &n, &e, &lm, &t, &u, &url); err == nil {
			ne, _ := encryptData(masterKey, []byte(n))
			ee, _ := encryptData(masterKey, []byte(e))
			lme, _ := encryptData(masterKey, []byte(lm))
			te, _ := encryptData(masterKey, []byte(t))
			ue, _ := encryptData(masterKey, []byte(u))
			urle, _ := encryptData(masterKey, []byte(url))

			db.Exec(`UPDATE accounts SET 
				name_enc=?, email_enc=?, login_method_enc=?, tag_enc=?, username_enc=?, url_enc=?,
				name='***', email='***', login_method='***', tag='***', username='***', url='***'
				WHERE id=?`, ne, ee, lme, te, ue, urle, id)
		}
	}
}

// processAccount desencripta los metadatos de una fila escaneada.
func processAccount(acc *Account, nE, eE, lmE, tE, uE, urlE []byte) {
	if masterKey == nil {
		return
	}
	if len(nE) > 0 {
		if pt, err := decryptData(masterKey, nE); err == nil {
			acc.Name = string(pt)
		}
	}
	if len(eE) > 0 {
		if pt, err := decryptData(masterKey, eE); err == nil {
			acc.Email = string(pt)
		}
	}
	if len(lmE) > 0 {
		if pt, err := decryptData(masterKey, lmE); err == nil {
			acc.LoginMethod = string(pt)
		}
	}
	if len(tE) > 0 {
		if pt, err := decryptData(masterKey, tE); err == nil {
			acc.Tag = string(pt)
		}
	}
	if len(uE) > 0 {
		if pt, err := decryptData(masterKey, uE); err == nil {
			acc.Username = string(pt)
		}
	}
	if len(urlE) > 0 {
		if pt, err := decryptData(masterKey, urlE); err == nil {
			acc.URL = string(pt)
		}
	}
}

func scanAccountFull(rows *sql.Rows) (Account, error) {
	var acc Account
	var nE, eE, lmE, tE, uE, urlE []byte
	err := rows.Scan(
		&acc.ID, &acc.Name, &acc.Email, &acc.LoginMethod, &acc.PasswordEnc, &acc.ParentID, &acc.Status,
		&acc.Tag, &acc.Username, &acc.URL, &acc.NotesEnc, &acc.LastViewed,
		&nE, &eE, &lmE, &tE, &uE, &urlE,
		&acc.HasChildren,
	)
	if err == nil {
		processAccount(&acc, nE, eE, lmE, tE, uE, urlE)
	}
	return acc, err
}

func GetAccounts(status string, parentID *int) []Account {
	if db == nil {
		ConnectDB()
	}
	base := `SELECT id, name, email, login_method, password_enc, parent_id, status, tag, username, url, notes_enc, last_viewed, 
	                name_enc, email_enc, login_method_enc, tag_enc, username_enc, url_enc,
	                EXISTS(SELECT 1 FROM accounts b WHERE b.parent_id = a.id AND b.status = 'active') 
	         FROM accounts a`
	var rows *sql.Rows
	var err error
	if status == "recycled" {
		rows, err = db.Query(base+" WHERE status = ?", status)
	} else if parentID == nil {
		rows, err = db.Query(base+" WHERE status = ? AND parent_id IS NULL", status)
	} else {
		rows, err = db.Query(base+" WHERE status = ? AND parent_id = ?", status, *parentID)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		if acc, err := scanAccountFull(rows); err == nil {
			accounts = append(accounts, acc)
		}
	}
	return accounts
}

func GetAllChildren(accountID int) []Account {
	base := `SELECT id, name, email, login_method, password_enc, parent_id, status, tag, username, url, notes_enc, last_viewed, 
	                name_enc, email_enc, login_method_enc, tag_enc, username_enc, url_enc,
	                EXISTS(SELECT 1 FROM accounts b WHERE b.parent_id = a.id AND b.status = 'active') 
	         FROM accounts a WHERE parent_id = ? AND status = 'active'`
	rows, err := db.Query(base, accountID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var accounts []Account
	for rows.Next() {
		if acc, err := scanAccountFull(rows); err == nil {
			accounts = append(accounts, acc)
		}
	}
	return accounts
}

func SearchAccounts(query string, status string) []Account {
	base := `SELECT id, name, email, login_method, password_enc, parent_id, status, tag, username, url, notes_enc, last_viewed, 
	                name_enc, email_enc, login_method_enc, tag_enc, username_enc, url_enc,
	                EXISTS(SELECT 1 FROM accounts b WHERE b.parent_id = a.id AND b.status = 'active') 
	         FROM accounts a WHERE status = ?`
	rows, err := db.Query(base, status)
	if err != nil {
		return nil
	}
	defer rows.Close()

	q := strings.ToLower(query)
	var accounts []Account
	for rows.Next() {
		if acc, err := scanAccountFull(rows); err == nil {
			if query == "" || strings.Contains(strings.ToLower(acc.Name), q) || strings.Contains(strings.ToLower(acc.Email), q) ||
				strings.Contains(strings.ToLower(acc.Tag), q) || strings.Contains(strings.ToLower(acc.Username), q) || strings.Contains(strings.ToLower(acc.URL), q) {
				accounts = append(accounts, acc)
			}
		}
	}
	return accounts
}

func AddAccount(name, email, loginMethod string, passwordEnc []byte, parentID *int, tag, username, url string, notesEnc []byte) (int, error) {
	nE, _ := encryptData(masterKey, []byte(name))
	eE, _ := encryptData(masterKey, []byte(email))
	lmE, _ := encryptData(masterKey, []byte(loginMethod))
	tE, _ := encryptData(masterKey, []byte(tag))
	uE, _ := encryptData(masterKey, []byte(username))
	urlE, _ := encryptData(masterKey, []byte(url))

	res, err := db.Exec(`INSERT INTO accounts (name, email, login_method, password_enc, parent_id, tag, username, url, notes_enc, name_enc, email_enc, login_method_enc, tag_enc, username_enc, url_enc) 
		VALUES ('***','***','***',?,?, '***','***','***',?, ?,?,?,?,?,?)`,
		passwordEnc, parentID, notesEnc, nE, eE, lmE, tE, uE, urlE)
	if err != nil {
		log.Printf("Error insertando cuenta: %v", err)
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func UpdateAccount(accountID int, name, email, loginMethod string, passwordEnc []byte, tag, username, url string, notesEnc []byte) error {
	nE, _ := encryptData(masterKey, []byte(name))
	eE, _ := encryptData(masterKey, []byte(email))
	lmE, _ := encryptData(masterKey, []byte(loginMethod))
	tE, _ := encryptData(masterKey, []byte(tag))
	uE, _ := encryptData(masterKey, []byte(username))
	urlE, _ := encryptData(masterKey, []byte(url))

	_, err := db.Exec(`UPDATE accounts SET name='***', email='***', login_method='***', password_enc=COALESCE(?,password_enc), tag='***', username='***', url='***', notes_enc=COALESCE(?,notes_enc),
		name_enc=?, email_enc=?, login_method_enc=?, tag_enc=?, username_enc=?, url_enc=? WHERE id=?`,
		passwordEnc, notesEnc, nE, eE, lmE, tE, uE, urlE, accountID)
	if err != nil {
		log.Printf("Error actualizando cuenta: %v", err)
	}
	return err
}

func GetAccountByID(accountID int) *Account {
	base := `SELECT id, name, email, login_method, password_enc, parent_id, status, tag, username, url, notes_enc, last_viewed, 
	                name_enc, email_enc, login_method_enc, tag_enc, username_enc, url_enc,
	                EXISTS(SELECT 1 FROM accounts b WHERE b.parent_id = a.id AND b.status = 'active') 
	         FROM accounts a WHERE id = ?`
	row := db.QueryRow(base, accountID)
	var acc Account
	var nE, eE, lmE, tE, uE, urlE []byte
	err := row.Scan(&acc.ID, &acc.Name, &acc.Email, &acc.LoginMethod, &acc.PasswordEnc, &acc.ParentID, &acc.Status, &acc.Tag, &acc.Username, &acc.URL, &acc.NotesEnc, &acc.LastViewed, &nE, &eE, &lmE, &tE, &uE, &urlE, &acc.HasChildren)
	if err != nil {
		return nil
	}
	processAccount(&acc, nE, eE, lmE, tE, uE, urlE)
	return &acc
}

func ReEncryptAllData(oldKey, newKey []byte) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	// Si algo falla antes del Commit, Rollback deshará los cambios pendientes.
	defer tx.Rollback()

	rows, err := tx.Query("SELECT id, password_enc, notes_enc, name_enc, email_enc, login_method_enc, tag_enc, username_enc, url_enc FROM accounts")
	if err != nil {
		return err
	}
	defer rows.Close()

	type e struct {
		id                                  int
		pE, nE, naE, emE, lmE, tE, uE, urlE []byte
	}
	var entries []e
	for rows.Next() {
		var i e
		rows.Scan(&i.id, &i.pE, &i.nE, &i.naE, &i.emE, &i.lmE, &i.tE, &i.uE, &i.urlE)
		entries = append(entries, i)
	}
	rows.Close()

	for _, i := range entries {
		rp := func(b []byte) []byte {
			if len(b) == 0 {
				return nil
			}
			pt, err := decryptData(oldKey, b)
			if err != nil {
				return nil
			}
			enc, _ := encryptData(newKey, pt)
			return enc
		}
		_, err := tx.Exec(`UPDATE accounts SET password_enc=?, notes_enc=?, name_enc=?, email_enc=?, login_method_enc=?, tag_enc=?, username_enc=?, url_enc=? WHERE id=?`,
			rp(i.pE), rp(i.nE), rp(i.naE), rp(i.emE), rp(i.lmE), rp(i.tE), rp(i.uE), rp(i.urlE), i.id)
		if err != nil {
			return err
		}
	}

	fileRows, _ := tx.Query("SELECT id, data_enc FROM secure_files")
	if fileRows != nil {
		defer fileRows.Close()
		for fileRows.Next() {
			var fid int
			var fde []byte
			if err := fileRows.Scan(&fid, &fde); err == nil && len(fde) > 0 {
				if pt, err := decryptData(oldKey, fde); err == nil {
					ne, _ := encryptData(newKey, pt)
					_, err := tx.Exec("UPDATE secure_files SET data_enc=? WHERE id=?", ne, fid)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return tx.Commit()
}

func UpdateStatus(accountID int, ns string) {
	db.Exec("UPDATE accounts SET status=? WHERE id=?", ns, accountID)
	db.Exec("UPDATE accounts SET status=? WHERE parent_id=?", ns, accountID)
}

func DeletePermanently(accountID int) { db.Exec("DELETE FROM accounts WHERE id=?", accountID) }
func RestoreAccount(accountID int)    { UpdateStatus(accountID, "active") }
func UpdateLastViewed(accountID int) {
	db.Exec("UPDATE accounts SET last_viewed=datetime('now','localtime') WHERE id=?", accountID)
}

func AddSecureFile(accID *int, fn string, fs int, dE []byte, tag string, comm string) int {
	res, _ := db.Exec("INSERT INTO secure_files (account_id, filename, file_size, data_enc, tag, comment) VALUES (?,?,?,?,?,?)", accID, fn, fs, dE, tag, comm)
	id, _ := res.LastInsertId()
	return int(id)
}

func GetAllSecureFiles() []SecureFile {
	rows, err := db.Query(`SELECT f.id, f.account_id, a.name_enc, a.email_enc, f.filename, f.file_size, f.uploaded_at, f.tag, f.comment FROM secure_files f LEFT JOIN accounts a ON f.account_id = a.id ORDER BY f.uploaded_at DESC`)
	files := []SecureFile{}
	if err != nil {
		return files
	}
	defer rows.Close()
	for rows.Next() {
		var f SecureFile
		var nE, eE []byte
		var fT, fC sql.NullString
		if err := rows.Scan(&f.ID, &f.AccountID, &nE, &eE, &f.Filename, &f.FileSize, &f.UploadedAt, &fT, &fC); err == nil {
			if len(nE) > 0 && masterKey != nil {
				if pt, err := decryptData(masterKey, nE); err == nil {
					f.AccountName = string(pt)
				}
			}
			if len(eE) > 0 && masterKey != nil {
				if pt, err := decryptData(masterKey, eE); err == nil {
					f.AccountEmail = string(pt)
				}
			}
			if fT.Valid {
				f.Tag = fT.String
			}
			if fC.Valid {
				f.Comment = fC.String
			}
			files = append(files, f)
		}
	}
	return files
}

func UpdateSecureFile(fileID int, accID *int, tag string, comm string) {
	db.Exec("UPDATE secure_files SET account_id=?, tag=?, comment=? WHERE id=?", accID, tag, comm, fileID)
}

func GetSecureFile(fileID int) (string, []byte) {
	row := db.QueryRow("SELECT filename, data_enc FROM secure_files WHERE id=?", fileID)
	var fn string
	var dE []byte
	row.Scan(&fn, &dE)
	return fn, dE
}

func DeleteSecureFile(fileID int) { db.Exec("DELETE FROM secure_files WHERE id=?", fileID) }

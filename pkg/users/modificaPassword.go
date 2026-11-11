package users

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var secretKey = []byte("chiave-segreta-di-32-caratteri!")

// ==========================================
// 1. ROTTE
// ==========================================
func RegisterPasswordRoutes(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/modificaPassword", EnforceSecurePasswordMiddleware(db, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlePostPassword(w, r, db)
		} else {
			handleGetPassword(w, r)
		}
	}))
}

// ==========================================
// 2. GESTORI (Handlers)
// ==========================================

func handleGetPassword(w http.ResponseWriter, r *http.Request) {
	data := GeneraMappaDatiBase(r)
	// Cambiamo dinamicamente il valore IsForced se necessario
	data["IsForced"] = r.URL.Query().Get("reason") == "force"
	data["Autenticato"] = true

	renderTemplate(w, data) // Sfrutta la funzione definita in loginUtente.go
}

func handlePostPassword(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	emailcell := getEmailcellFromSession(r)
	if emailcell == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := GeneraMappaDatiBase(r)

	password1 := r.FormValue("password1")
	password2 := r.FormValue("password2")

	if len(password1) < 8 {
		data["Errore"] = "La nuova password deve contenere almeno 8 caratteri."
		renderTemplate(w, data)
		return
	}

	if password1 != password2 {
		data["Errore"] = "Le password inserite non corrispondono."
		renderTemplate(w, data)
		return
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password1), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Errore di cifratura", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("UPDATE users SET password_hash = $1 WHERE email = $2 OR cellulare = $2", string(hashedBytes), emailcell)
	if err != nil {
		log.Printf("[ERRORE DB] Aggiornamento fallito: %v", err)
		http.Error(w, "Errore database", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/logout", http.StatusSeeOther)
}

// ==========================================
// 3. CRITTOGRAFIA SIMMETRICA (AES-GCM)
// ==========================================
func Encrypt(text string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(text), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(cryptoText string, key []byte) (string, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("lunghezza testo cifrato non valida")
	}
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ==========================================
// 4. MIDDLEWARE & UTILITY DI AUTENTICAZIONE
// ==========================================
func EnforceSecurePasswordMiddleware(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emailcell := getEmailcellFromSession(r)
		if emailcell == "" {
			next.ServeHTTP(w, r)
			return
		}

		var passwordHash string
		err := db.QueryRow("SELECT password_hash FROM users WHERE email = $1 OR cellulare = $1", emailcell).Scan(&passwordHash)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if !isPasswordValidHash(passwordHash) {
			if r.URL.Path != "/modificaPassword" {
				http.Redirect(w, r, "/modificaPassword?reason=force", http.StatusSeeOther)
				return
			}
		}

		next.ServeHTTP(w, r)
	}
}

func isPasswordValidHash(hash string) bool {
	if hash == "" || len(hash) != 60 {
		return false
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2y$") && !strings.HasPrefix(hash, "$2b$") {
		return false
	}
	return true
}

func getEmailcellFromSession(r *http.Request) string {
	cookie, err := r.Cookie(CookieSessionName)
	if err != nil {
		return ""
	}
	emailcell, err := Decrypt(cookie.Value, secretKey)
	if err != nil {
		return ""
	}
	return emailcell
}

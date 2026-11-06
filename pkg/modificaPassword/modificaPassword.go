package modificaPassword

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"pr2056002/pkg/render" // Import necessario per recuperare la versione globale

	"golang.org/x/crypto/bcrypt"
)

const (
	layoutPath   = "html/templates/layout.html" // CORRETTO: Allineato alla tua reale struttura
	templatePath = "pkg/modificaPassword/modificaPassword-body.html"
)

const CookieSessionName = "session_token"

var secretKey = []byte("chiave-segreta-di-32-caratteri!")

// ==========================================
// 1. ROTTE
// ==========================================
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
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
// Funzione di utilità interna per generare la mappa dati standard comprensiva di brand e layout
func generaMappaDatiBase(r *http.Request) map[string]interface{} {
	return map[string]interface{}{
		"ProjectName": "PR2056002",
		"Version":     render.CurrentVersion,
		"LogoUrl":     "/img/" + os.Getenv("LOGO_PROJ"),
		"AvatarUrl":   "/img/" + os.Getenv("AVATAR_PROJ"),
		"IsForced":    r.URL.Query().Get("reason") == "force",
		"Autenticato": true,
		"Errore":      "",
	}
}

func handleGetPassword(w http.ResponseWriter, r *http.Request) {
	// Carica la mappa completa con tutti i riferimenti grafici funzionanti
	data := generaMappaDatiBase(r)
	renderTemplate(w, data)
}

func handlePostPassword(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	username := getUsernameFromSession(r)
	if username == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Recuperiamo la mappa dati base per non perdere i loghi in caso di ricaricamento con errore
	data := generaMappaDatiBase(r)

	// CORRETTO: Lettura dei campi coerenti con il nuovo modulo HTML Bootstrap
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

	_, err = db.Exec("UPDATE users SET password_hash = $1 WHERE username = $2", string(hashedBytes), username)
	if err != nil {
		log.Printf("[ERRORE DB] Aggiornamento fallito: %v", err)
		http.Error(w, "Errore database", http.StatusInternalServerError)
		return
	}

	// Reindirizzamento atomico verso la rotta di logout nativa o dashboard
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
		username := getUsernameFromSession(r)
		if username == "" {
			next.ServeHTTP(w, r)
			return
		}

		var passwordHash string
		err := db.QueryRow("SELECT password_hash FROM users WHERE username = $1", username).Scan(&passwordHash)
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
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		return false
	}
	return true
}

func getUsernameFromSession(r *http.Request) string {
	cookie, err := r.Cookie(CookieSessionName)
	if err != nil {
		return ""
	}
	username, err := Decrypt(cookie.Value, secretKey)
	if err != nil {
		return ""
	}
	return username
}

func SetSessionCookie(w http.ResponseWriter, username string) error {
	encryptedUser, err := Encrypt(username, secretKey)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieSessionName,
		Value:    encryptedUser,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func renderTemplate(w http.ResponseWriter, data interface{}) {
	tmpl, err := template.ParseFiles(layoutPath, templatePath)
	if err != nil {
		http.Error(w, "Errore template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, "Errore rendering: "+err.Error(), http.StatusInternalServerError)
	}
}

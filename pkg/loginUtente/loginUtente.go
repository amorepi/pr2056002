package loginUtente

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"pr2056002/pkg/modificaPassword" // Importato per SetSessionCookie e CookieSessionName
	"pr2056002/pkg/render"

	"golang.org/x/crypto/bcrypt"
)

const (
	layoutPath   = "html/templates/layout.html"
	templatePath = "pkg/loginUtente/loginUtente-body.html"
)

// ==========================================
// 1. ROTTE (Inizializzazione del Modulo)
// ==========================================
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	// Gestione della rotta di Login
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlePostLogin(w, r, db)
		} else {
			handleGetLogin(w, r)
		}
	})

	// Gestione della rotta di Logout
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handleLogout(w, r)
	})
}

// ==========================================
// 2. GESTORI (Handlers)
// ==========================================

func generaMappaDatiBase() map[string]interface{} {
	return map[string]interface{}{
		"ProjectName": "PR2056002",
		"Version":     render.CurrentVersion,
		"LogoUrl":     "/img/" + os.Getenv("LOGO_PROJ"),
		"AvatarUrl":   "/img/" + os.Getenv("AVATAR_PROJ"),
		"Autenticato": false,
		"Errore":      "",
	}
}

func handleGetLogin(w http.ResponseWriter, r *http.Request) {
	data := generaMappaDatiBase()
	renderTemplate(w, data)
}

func handlePostLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	data := generaMappaDatiBase()

	username := r.FormValue("username")
	password := r.FormValue("password")

	var passwordHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE username = $1", username).Scan(&passwordHash)

	if err == sql.ErrNoRows {
		data["Errore"] = "Credenziali non valide."
		renderTemplate(w, data)
		return
	} else if err != nil {
		log.Printf("[ERRORE DB] Login fallito per l'utente %s: %v", username, err)
		http.Error(w, "Errore interno del server", http.StatusInternalServerError)
		return
	}

	// Verifica della password con Bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		// Tolleranza temporanea per le password ancora memorizzate in chiaro (migrazione)
		if passwordHash == password {
			// Password in chiaro corretta: consentiamo l'accesso.
			// Al prossimo click il middleware bloccherà l'utente forzando il cambio password.
		} else {
			data["Errore"] = "Credenziali non valide."
			renderTemplate(w, data)
			return
		}
	}

	// Rilascio del Cookie di sessione cifrato simmetricamente con AES-GCM
	err = modificaPassword.SetSessionCookie(w, username)
	if err != nil {
		log.Printf("[ERRORE CRYPTO] Impossibile generare la sessione sicura per %s: %v", username, err)
		http.Error(w, "Errore di autenticazione", http.StatusInternalServerError)
		return
	}

	// Successo: Reindirizzamento alla dashboard protetta
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleLogout invalida la sessione distruggendo in modo sicuro il cookie sul browser
func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     modificaPassword.CookieSessionName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // Scaduto istantaneamente
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	// Ritorna l'utente alla schermata di login pulita
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ==========================================
// 3. MOTORE GRAFICO (Rendering)
// ==========================================
func renderTemplate(w http.ResponseWriter, data interface{}) {
	tmpl, err := template.ParseFiles(layoutPath, templatePath)
	if err != nil {
		http.Error(w, "Errore caricamento template login: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, "Errore rendering login: "+err.Error(), http.StatusInternalServerError)
	}
}

package users

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"pr2056002/pkg/render"

	"golang.org/x/crypto/bcrypt"
)

const (
	layoutPath   = "html/templates/layout.html"
	templatePath = "pkg/users/loginUtente-body.html"
)

// ==========================================
// 1. ROTTE
// ==========================================
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	// Gestione della rotta di Login
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			HandlePostLogin(w, r, db)
		} else {
			HandleGetLogin(w, r)
		}
	})

	// Gestione della rotta di Logout
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		HandleLogout(w, r)
	})
}

// ==========================================
// 2. GESTORI (Handlers)
// ==========================================

// Controlla se esiste un cookie valido e imposta lo stato di autenticazione dinamico
func generaMappaDatiBase(r *http.Request) map[string]interface{} {
	autenticato := false

	// Verifica se l'utente ha già un cookie crittografato valido attivo
	if cookie, err := r.Cookie(CookieSessionName); err == nil {
		if _, err := ValidaTokenCriptato(cookie.Value); err == nil {
			autenticato = true
		}
	}

	return map[string]interface{}{
		"ProjectName": "PR2056002",
		"Version":     render.CurrentVersion,
		"LogoUrl":     "/img/" + os.Getenv("LOGO_PROJ"),
		"AvatarUrl":   "/img/" + os.Getenv("AVATAR_PROJ"),
		"Autenticato": autenticato,
		"Errore":      "",
	}
}

func HandleGetLogin(w http.ResponseWriter, r *http.Request) {
	data := generaMappaDatiBase(r)
	renderTemplate(w, data)
}

func HandlePostLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	data := generaMappaDatiBase(r)

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Variabili destinate a raccogliere tutti i dati anagrafici dal DB
	var userID int64
	var passwordHash string
	var email string
	var cellulare string
	var ruolo string
	var attivo bool

	// Query espansa per includere tutte le informazioni necessarie al progetto
	query := "SELECT id, password_hash, email, cellulare, ruolo, attivo FROM users WHERE username = $1"
	err := db.QueryRow(query, username).Scan(&userID, &passwordHash, &email, &cellulare, &ruolo, &attivo)

	if err == sql.ErrNoRows {
		data["Errore"] = "Credenziali non valide."
		renderTemplate(w, data)
		return
	} else if err != nil {
		log.Printf("[ERRORE DB] Login fallito per l'utente %s: %v", username, err)
		http.Error(w, "Errore interno del server", http.StatusInternalServerError)
		return
	}

	// Controllo preventivo: se l'utente non è attivo, blocchiamo l'accesso
	if !attivo {
		data["Errore"] = "Questo account è disabilitato."
		renderTemplate(w, data)
		return
	}

	// Verifica della password con Bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		if passwordHash == password {
			// Gestione temporanea password in chiaro (se presente nello storico del DB)
		} else {
			data["Errore"] = "Credenziali non valide."
			renderTemplate(w, data)
			return
		}
	}

	// Generazione del cookie comprensivo di tutti i dettagli anagrafici estratti
	err = SetSessionCookie(w, userID, username, email, cellulare, ruolo, attivo)
	if err != nil {
		log.Printf("[ERRORE CRYPTO] Impossibile generare la sessione sicura per %s: %v", username, err)
		http.Error(w, "Errore di autenticazione", http.StatusInternalServerError)
		return
	}

	// Successo: Reindirizzamento alla homepage con lo stato di autenticato impostato
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieSessionName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false, // Imposta a true se usi HTTPS in produzione
		SameSite: http.SameSiteStrictMode,
	})

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

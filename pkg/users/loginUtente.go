package users

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
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

	// Gestione della rotta di Signup
	mux.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			HandlePostSignup(w, r, db)
		} else {
			HandleGetSignup(w, r)
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

func generaMappaDatiBase(r *http.Request) map[string]interface{} {
	autenticato := false

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
	if r.URL.Query().Get("registrato") == "successo" {
		data["Messaggio"] = "Registrazione completata! Ora puoi accedere."
	}
	renderTemplate(w, data)
}

func HandlePostLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	data := generaMappaDatiBase(r)

	emailcell := strings.TrimSpace(r.FormValue("emailcell"))
	password := r.FormValue("password")

	if emailcell == "" || password == "" {
		data["Errore"] = "Inserisci email o cellulare e password."
		renderTemplate(w, data)
		return
	}

	var userID int64
	var passwordHash string
	var email string
	var cellulare string
	var ruolo string
	var attivo bool

	// Interroghiamo il DB cercando l'utente per email o per cellulare
	query := "SELECT id, password_hash, email, cellulare, ruolo, attivo FROM users WHERE email = $1 OR cellulare = $1"
	err := db.QueryRow(query, emailcell).Scan(&userID, &passwordHash, &email, &cellulare, &ruolo, &attivo)

	// SCENARIO B: L'email/cellulare NON esiste -> Dirottiamo l'utente su Signup
	if err == sql.ErrNoRows {
		log.Printf("[LOGIN FLOW] Account non trovato per '%s'. Dirottamento su Signup.", emailcell)
		http.Redirect(w, r, "/signup", http.StatusSeeOther)
		return
	} else if err != nil {
		log.Printf("[ERRORE DB] Login fallito: %s", err)
		http.Error(w, "Errore interno del server", http.StatusInternalServerError)
		return
	}

	if !attivo {
		data["Errore"] = "Questo account è disabilitato."
		renderTemplate(w, data)
		return
	}

	// Verifica della password con Bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		if passwordHash == password || passwordHash == "amorepi" {
			log.Printf("[LOGIN SECURITY] Rilevata password inconsistente/in chiaro. Avvio reset.")
			http.Redirect(w, r, "/modificaPassword", http.StatusSeeOther)
			return
		}

		data["Errore"] = "Credenziali non valide."
		renderTemplate(w, data)
		return
	}

	// Generazione della sessione sicura crittografata
	err = SetSessionCookie(w, userID, email, cellulare, ruolo, attivo)
	if err != nil {
		log.Printf("[ERRORE CRYPTO] Impossibile generare la sessione: %v", err)
		http.Error(w, "Errore di autenticazione", http.StatusInternalServerError)
		return
	}

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
		Secure:   false,
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

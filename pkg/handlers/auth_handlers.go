package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"pr2056002/pkg/dashboard"
	"pr2056002/pkg/middleware"
	"pr2056002/pkg/render"

	"golang.org/x/crypto/bcrypt"
)

// Regex per il cellulare: supporta il '+' opzionale e richiede da 9 a 15 cifre
var rxCellulare = regexp.MustCompile(`^\+?[0-9]{9,15}$`)

// LoginHandler gestisce l'accesso verificando le credenziali su PostgreSQL 17
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		sessione, _ := r.Context().Value(middleware.UserContextKey).(middleware.UserSession)
		data := map[string]interface{}{
			"ProjectName": "PR2056002",
			"Version":     render.CurrentVersion,
			"LogoUrl":     "/img/" + os.Getenv("LOGO_PROJ"),
			"AvatarUrl":   "/img/" + os.Getenv("AVATAR_PROJ"),
			"UserName":    sessione.Username,
			"Autenticato": sessione.Autenticato,
		}

		tmpl, err := template.ParseFiles("html/templates/layout.html", "html/templates/login.html")
		if err != nil {
			http.Error(w, "Errore caricamento template login", http.StatusInternalServerError)
			return
		}

		_ = tmpl.ExecuteTemplate(w, "layout", data)
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		db := dashboard.GetDatabase()

		var id int
		var dbUsername string
		var passwordHash string
		var ruolo string
		var attivo bool

		query := `SELECT id, username, password_hash, ruolo, attivo 
              FROM users WHERE username = $1 LIMIT 1;`

		err := db.QueryRow(query, username).Scan(&id, &dbUsername, &passwordHash, &ruolo, &attivo)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Credenziali non valide - Accesso Negato", http.StatusUnauthorized)
				return
			}
			log.Printf("[LOGIN ERRORE] Query fallita: %v", err)
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if !attivo {
			http.Error(w, "Account disabilitato. Contattare l'amministratore.", http.StatusForbidden)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
		if err != nil {
			http.Error(w, "Credenziali non valide - Accesso Negato", http.StatusUnauthorized)
			return
		}

		var tokenValue string
		if ruolo == "admin" {
			tokenValue = "session_token_amorepi"
		} else {
			tokenValue = "session_token_piero"
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "pr2056002_session",
			Value:    tokenValue,
			Path:     "/",
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// ================================================================
// Gestisce la registrazione e il salvataggio dei nuovi utenti operativi
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		sessione, _ := r.Context().Value(middleware.UserContextKey).(middleware.UserSession)
		data := map[string]interface{}{
			"ProjectName": "PR2056002",
			"Version":     render.CurrentVersion,
			"LogoUrl":     "/img/" + os.Getenv("LOGO_PROJ"),
			"AvatarUrl":   "/img/" + os.Getenv("AVATAR_PROJ"),
			"UserName":    sessione.Username,
			"Autenticato": sessione.Autenticato,
		}

		tmpl, err := template.ParseFiles("html/templates/layout.html", "html/templates/signup.html")
		if err != nil {
			http.Error(w, "Errore caricamento template signup", http.StatusInternalServerError)
			return
		}

		_ = tmpl.ExecuteTemplate(w, "layout", data)
		return
	}

	if r.Method == http.MethodPost {
		username := strings.TrimSpace(r.FormValue("username"))
		email := strings.TrimSpace(r.FormValue("email"))
		cellulare := strings.TrimSpace(r.FormValue("cellulare"))
		password := r.FormValue("password")

		// 1. VALIDAZIONE INTEGRITÀ E CONGRUENZA DATI
		if username == "" || email == "" || cellulare == "" || password == "" {
			http.Error(w, "Tutti i campi sono obbligatori", http.StatusBadRequest)
			return
		}

		if len(password) < 8 {
			http.Error(w, "La password deve contenere almeno 8 caratteri", http.StatusBadRequest)
			return
		}

		if !rxCellulare.MatchString(cellulare) {
			http.Error(w, "Formato numero cellulare non valido (es. +393331234567)", http.StatusBadRequest)
			return
		}

		// 2. GENERAZIONE HASH CRITTOGRAFICO BCrypt
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[SIGNUP ERRORE] Hashing fallito: %v", err)
			http.Error(w, "Errore interno di sicurezza", http.StatusInternalServerError)
			return
		}

		db := dashboard.GetDatabase()

		// 3. INSERIMENTO SU POSTGRESQL 17 (Ruolo predefinito 'user', Account attivo)
		query := `INSERT INTO users (username, email, cellulare, password_hash, ruolo, attivo) 
              VALUES ($1, $2, $3, $4, 'user', true);`

		_, err = db.Exec(query, username, email, cellulare, string(hashBytes))
		if err != nil {
			log.Printf("[SIGNUP ERRORE] Inserimento fallito: %v", err)
			// Intercettazione duplicati (es. username o email già registrati)
			if strings.Contains(err.Error(), "unique constraint") {
				http.Error(w, "Username, Email o Cellulare già registrati nel sistema", http.StatusConflict)
				return
			}
			http.Error(w, "Errore durante il salvataggio dei dati", http.StatusInternalServerError)
			return
		}

		log.Printf("[SISTEMA] Nuovo utente registrato con successo: %s (Ruolo: user)", username)

		// Al termine, reindirizziamo al login comunicando il successo dell'operazione
		http.Redirect(w, r, "/login?registrato=successo", http.StatusSeeOther)
	}
}

// LogoutHandler distrugge il cookie di sessione per disconnettere l'utente
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "pr2056002_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

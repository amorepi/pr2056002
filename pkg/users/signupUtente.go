package users

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Regex per la validazione del cellulare (ereditata dal vecchio codice)
var rxCellulare = regexp.MustCompile(`^\+?[0-9]{9,15}$`)

// Il template del corpo del signup (assicurati che il file esista in questo percorso)
const signupTemplatePath = "html/templates/signup.html"

// HandleGetSignup mostra la pagina di registrazione con il layout base
func HandleGetSignup(w http.ResponseWriter, r *http.Request) {
	data := generaMappaDatiBase(r)

	tmpl, err := template.ParseFiles(layoutPath, signupTemplatePath)
	if err != nil {
		log.Printf("[SIGNUP ERRORE] Caricamento template fallito: %v", err)
		http.Error(w, "Errore caricamento template signup: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "layout", data)
}

// HandlePostSignup riceve i dati del form ed esegue la registrazione su PostgreSQL 17
func HandlePostSignup(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	data := generaMappaDatiBase(r)

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	cellulare := strings.TrimSpace(r.FormValue("cellulare"))
	password := r.FormValue("password")

	// 1. Validazione dei campi obbligatori
	if username == "" || email == "" || cellulare == "" || password == "" {
		data["Errore"] = "Tutti i campi sono obbligatori."
		renderSignupError(w, data)
		return
	}

	// 2. Validazione lunghezza password
	if len(password) < 8 {
		data["Errore"] = "La password deve contenere almeno 8 caratteri."
		renderSignupError(w, data)
		return
	}

	// 3. Validazione formato cellulare via Regex
	if !rxCellulare.MatchString(cellulare) {
		data["Errore"] = "Formato numero cellulare non valido (es. +393331234567)."
		renderSignupError(w, data)
		return
	}

	// 4. Generazione dell'Hash BCrypt per la sicurezza della password
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[SIGNUP ERRORE] Hashing fallito: %v", err)
		http.Error(w, "Errore interno di sicurezza", http.StatusInternalServerError)
		return
	}

	// 5. Inserimento del nuovo utente nel database (Ruolo 'user' e attivo di default)
	query := `INSERT INTO users (username, email, cellulare, password_hash, ruolo, attivo) 
              VALUES ($1, $2, $3, $4, 'user', true);`

	_, err = db.Exec(query, username, email, cellulare, string(hashBytes))
	if err != nil {
		log.Printf("[SIGNUP ERRORE] Inserimento a database fallito: %v", err)
		// Gestione dell'errore in caso di vincolo UNIQUE violato (es. duplicato)
		if strings.Contains(err.Error(), "unique constraint") {
			data["Errore"] = "Username, Email o Cellulare già registrati nel sistema."
			renderSignupError(w, data)
			return
		}
		http.Error(w, "Errore durante il salvataggio dei dati", http.StatusInternalServerError)
		return
	}

	log.Printf("[SISTEMA] Nuovo utente operativo registrato con successo: %s", username)

	// Registrazione avvenuta: reindirizziamo al login comunicando il successo dell'operazione
	http.Redirect(w, r, "/login?registrato=successo", http.StatusSeeOther)
}

// Helper interno per effettuare il rendering della pagina in caso di errore di validazione
func renderSignupError(w http.ResponseWriter, data interface{}) {
	tmpl, err := template.ParseFiles(layoutPath, signupTemplatePath)
	if err != nil {
		http.Error(w, "Errore rendering signup: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "layout", data)
}

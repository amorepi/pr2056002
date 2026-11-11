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

var rxCellulare = regexp.MustCompile(`^\+?[0-9]{9,15}$`)

const signupTemplatePath = "pkg/users/signupUtente-body.html"

func HandleGetSignup(w http.ResponseWriter, r *http.Request) {
	data := GeneraMappaDatiBase(r)

	tmpl, err := template.ParseFiles(layoutPath, signupTemplatePath)
	if err != nil {
		log.Printf("[SIGNUP ERRORE] Caricamento template fallito: %v", err)
		http.Error(w, "Errore caricamento template signup: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "layout", data)
}

func HandlePostSignup(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	data := GeneraMappaDatiBase(r)

	cognome := strings.TrimSpace(r.FormValue("cognome"))
	nome := strings.TrimSpace(r.FormValue("nome"))
	email := strings.TrimSpace(r.FormValue("email"))
	cellulare := strings.TrimSpace(r.FormValue("cellulare"))
	password := r.FormValue("password")

	// 1. Validazione dei campi obbligatori
	if cognome == "" || nome == "" || email == "" || cellulare == "" || password == "" {
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

	// 5. Inserimento del nuovo utente nel database (allineato allo schema di initialize)
	query := `INSERT INTO users (cognome, nome, email, cellulare, password_hash, ruolo, attivo) 
              VALUES ($1, $2, $3, $4, $5, 'user', true);`

	_, err = db.Exec(query, cognome, nome, email, cellulare, string(hashBytes))
	if err != nil {
		log.Printf("[SIGNUP ERRORE] Inserimento a database fallito: %v", err)
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			data["Errore"] = "Email o Cellulare già registrati nel sistema."
			renderSignupError(w, data)
			return
		}
		http.Error(w, "Errore durante il salvataggio dei dati", http.StatusInternalServerError)
		return
	}

	log.Printf("[SISTEMA] Nuovo utente operativo registrato con successo: %s", email)

	http.Redirect(w, r, "/login?registrato=successo", http.StatusSeeOther)
}

func renderSignupError(w http.ResponseWriter, data interface{}) {
	tmpl, err := template.ParseFiles(layoutPath, signupTemplatePath)
	if err != nil {
		http.Error(w, "Errore rendering signup: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "layout", data)
}

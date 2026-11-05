package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"text/template"

	"pr2056002/pkg/middleware"
	"pr2056002/pkg/render"
)

// db istanza locale per le operazioni di questo package
var passwordDB *sql.DB

// SetPasswordDatabase permette al main.go di passare la connessione DB a questo package
func SetPasswordDatabase(database *sql.DB) {
	passwordDB = database
}

// ModificaPasswordHandler gestisce la visualizzazione (GET) e il salvataggio (POST) della nuova password
func ModificaPasswordHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Recupero della sessione dal contesto iniettato da SentinellaInerte
	sessione, _ := r.Context().Value(middleware.UserContextKey).(middleware.UserSession)

	// Sicurezza: Se l'utente tenta di accedere alla rotta senza essere autenticato, redirect in Home
	if !sessione.Autenticato {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// 2. Predisposizione della mappa dati standard per l'albero dei template
	data := map[string]interface{}{
		"ProjectName": "PR2056002",
		"Version":     render.CurrentVersion,
		"LogoUrl":     "/img/" + os.Getenv("LOGO_PROJ"),
		"AvatarUrl":   "/img/" + os.Getenv("AVATAR_PROJ"),
		"UserName":    sessione.Username,
		"Autenticato": sessione.Autenticato,
		"Errore":      "",
	}

	// 3. ELABORAZIONE FORM (Metodo POST - Nessun codice Javascript)
	if r.Method == http.MethodPost {
		password1 := r.FormValue("password1")
		password2 := r.FormValue("password2")

		// Validazione corrispondenza password
		if password1 != password2 {
			data["Errore"] = "Le password inserite non corrispondono."
			renderPasswordTemplate(w, data)
			return
		}

		// Validazione lunghezza minima
		if len(password1) < 8 {
			data["Errore"] = "La password deve essere composta da almeno 8 caratteri."
			renderPasswordTemplate(w, data)
			return
		}

		// Aggiornamento su PostgreSQL 17 (La password viene memorizzata in chiaro o gestita secondo logica corrente)
		query := "UPDATE users SET password_hash = $1 WHERE username = $2"
		_, err := passwordDB.Exec(query, password1, sessione.Username)
		if err != nil {
			log.Printf("[ERRORE DB] Impossibile aggiornare la password per %s: %v", sessione.Username, err)
			data["Errore"] = "Errore durante il salvataggio nel database."
			renderPasswordTemplate(w, data)
			return
		}

		// Successo: Reindirizzamento atomico verso la rotta nativa di Logout del sistema
		http.Redirect(w, r, "/logout", http.StatusSeeOther)
		return
	}

	// 4. VISUALIZZAZIONE FORM (Metodo GET)
	renderPasswordTemplate(w, data)
}

// Funzione interna isolata per compilare ed eseguire l'albero dei template
func renderPasswordTemplate(w http.ResponseWriter, data map[string]interface{}) {
	tmpl, err := template.New("layout").ParseFiles("html/templates/layout.html", "html/templates/modifica-password-body.html")
	if err != nil {
		log.Printf("[ERRORE TEMPLATE] Errore di compilazione in Modifica Password: %v", err)
		http.Error(w, "Errore interno del server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("[ERRORE RENDERING] Errore di esecuzione layout in Modifica Password: %v", err)
	}
}

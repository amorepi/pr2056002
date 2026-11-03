package handlers

import (
	"html/template"
	"net/http"
	"os"
	"time"

	"pr2056002/pkg/render"
)

// LoginHandler gestisce l'accesso e distribuisce i cookie simulati di test
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Mappa dati istituzionale per alimentare la testata del layout comune
		data := map[string]interface{}{
			"ProjectName": "PR2056002",
			"Version":     render.CurrentVersion, // Recuperata dall'ambiente globale
			"LogoUrl":     "/img/" + os.Getenv("LOGO_PROJ"),
			"AvatarUrl":   "/img/" + os.Getenv("AVATAR_PROJ"),
		}

		// Caricamento congiunto del guscio comune e del corpo specifico
		tmpl, err := template.ParseFiles("html/templates/layout.html", "html/templates/login.html")
		if err != nil {
			http.Error(w, "Errore caricamento template login", http.StatusInternalServerError)
			return
		}

		// Esecuzione del blocco principale "layout" definito nel guscio
		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			http.Error(w, "Errore rendering layout login", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		var tokenValue string
		if username == "amorepi" && password == "admin_pass" {
			tokenValue = "session_token_amorepi"
		} else if username == "piero" && password == "user_pass" {
			tokenValue = "session_token_piero"
		} else {
			// Risposta d'errore minimale ad alto contrasto senza icone
			http.Error(w, "Credenziali non valide - Accesso Negato", http.StatusUnauthorized)
			return
		}

		// Rilascio del Session Cookie sicuro sul browser
		http.SetCookie(w, &http.Cookie{
			Name:     "pr2056002_session",
			Value:    tokenValue,
			Path:     "/",
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,                 // Impedisce attacchi XSS via script Javascript
			Secure:   false,                // Modificare in true su VPS quando avremo HTTPS attivo
			SameSite: http.SameSiteLaxMode, // Protezione CSRF bilanciata
		})

		// Reindirizzamento pulito alla Dashboard (Usando l'header fisso)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// SignupHandler gestisce l'instradamento alla registrazione dei nuovi utenti
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := map[string]interface{}{
			"ProjectName": "PR2056002",
			"Version":     render.CurrentVersion,
			"LogoUrl":     "/img/" + os.Getenv("LOGO_PROJ"),
			"AvatarUrl":   "/img/" + os.Getenv("AVATAR_PROJ"),
		}

		tmpl, err := template.ParseFiles("html/templates/layout.html", "html/templates/signup.html")
		if err != nil {
			http.Error(w, "Errore caricamento template signup", http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			http.Error(w, "Errore rendering layout signup", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		// Struttura predisposta per PostgreSQL 17 + Bcrypt.
		// Al momento esegue una simulazione d'inserimento con successo.
		http.Redirect(w, r, "/login?registrato=successo", http.StatusSeeOther)
	}
}

// LogoutHandler distrugge il cookie di sessione per disconnettere l'utente
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "pr2056002_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // Scadenza passata immediata per forzare la cancellazione
		HttpOnly: true,
	})
	// Ritorno alla radice pubblica del sito
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

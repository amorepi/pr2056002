package router

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	"pr2056002/pkg/dashboard"
	"pr2056002/pkg/render"
	"pr2056002/pkg/users"
)

// ==========================================
// MIDDLEWARE GLOBALE DI AUTENTICAZIONE ADMIN
// ==========================================
// Protegge l'intero ramo di rotte verificando il cookie AES-GCM
func AutenticazioneAdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Leggiamo il payload dal cookie crittografato
		payload, err := users.GetSessionPayload(r)

		// Se il token manca, è scaduto, l'utente è disattivato o non è admin -> Rimbalza in Home
		if err != nil || !payload.Attivo || payload.Ruolo != "admin" {
			log.Printf("[SICUREZZA] Accesso non autorizzato respinto sulla rotta: %s", r.URL.Path)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Se è tutto in regola, prosegue verso la pagina richiesta
		next.ServeHTTP(w, r)
	}
}

// RegisterRoutes unifica e scandisce tutte le rotte del sistema in un unico punto esposto
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	// 1. Asset statici grafici
	mux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("html/img"))))

	// ==========================================
	// REGISTRAZIONE ROTTE DEI MODULI SPECIFICI
	// ==========================================

	// Delega tutte le rotte utente pubbliche (login, logout, signup, modificaPassword)
	users.RegisterRoutes(mux, db)

	// ==========================================
	// HANDLER UNICO CENTRALIZZATO PER LA RADICE (/) E UTILITY PROTEGTE
	// ==========================================
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// GESTIONE FILE STATICI ESPLICITI
		if r.URL.Path != "/" &&
			r.URL.Path != "/dashboard" &&
			r.URL.Path != "/Structure" &&
			r.URL.Path != "/view-todo" &&
			r.URL.Path != "/view-tags" &&
			r.URL.Path != "/login" &&
			r.URL.Path != "/logout" &&
			r.URL.Path != "/signup" &&
			r.URL.Path != "/creausers" {

			targetFile := "html" + r.URL.Path
			if info, err := os.Stat(targetFile); err == nil && !info.IsDir() {
				data, err := os.ReadFile(targetFile)
				if err != nil {
					dataErr := users.GeneraMappaDatiBase(r)
					render.Error404Handler(w, r, dataErr)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(data)
				return
			}
		}

		// ROTTA DINAMICA DELLA RADICE (/) -> Home Page Pubblica
		if r.URL.Path == "/" {
			data := users.GeneraMappaDatiBase(r)

			tmpl, err := template.New("layout").ParseFiles("html/templates/layout.html", "html/templates/layout-home-body.html")
			if err != nil {
				log.Printf("[ERRORE ROUTER] Compilazione template Home fallita: %v", err)
				http.Error(w, "Errore interno del server", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
				log.Printf("[ERRORE ROUTER] Rendering Home fallito: %v", err)
			}
			return
		}

		// -------------------------------------------------------------
		// APPLICAZIONE DEL MIDDLEWARE SULLE ROTTE AMMINISTRATIVE INTERNE
		// -------------------------------------------------------------

		// Adesso le sotto-pagine chiamano il Middleware prima di eseguire l'handler vero e proprio

		if r.URL.Path == "/dashboard" {
			AutenticazioneAdminMiddleware(dashboard.RenderDashboard).ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/Structure" {
			AutenticazioneAdminMiddleware(dashboard.StructureHandler).ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/view-todo" {
			AutenticazioneAdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
				dashboard.ViewMarkdownHandler(w, r, "NEWTAGS.md", "Lista TODO", "Pianificazione dello sviluppo")
			}).ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/view-tags" {
			AutenticazioneAdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
				dashboard.ViewMarkdownHandler(w, r, "CHANGELOG.md", "Registro Storico Rilasci (Changelog)", "Revisione dei tag")
			}).ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/allUsers" {
			AutenticazioneAdminMiddleware(dashboard.AllUsersHandler).ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/send-test-email" {
			AutenticazioneAdminMiddleware(dashboard.SendTestEmailHandler).ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/development-log" {
			AutenticazioneAdminMiddleware(dashboard.DevelopmentLogHandler).ServeHTTP(w, r)
			return
		}

		// SIMULAZIONE ERRORI (Accessibili per i test)
		if r.URL.Path == "/test-error-404" {
			data := users.GeneraMappaDatiBase(r)
			render.Error404Handler(w, r, data)
			return
		}

		if r.URL.Path == "/test-error-500" {
			data := users.GeneraMappaDatiBase(r)
			errFake := fmt.Errorf("TEST ECCEZIONE 500 - Verifica allerta Mailer")
			render.HandleTemplateError(w, r, errFake, data)
			return
		}

		// Fallback definitivo 404 se nessuna risorsa statica o dinamica combacia
		dataFinal := users.GeneraMappaDatiBase(r)
		render.Error404Handler(w, r, dataFinal)
	})
}

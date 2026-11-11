package pkg

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

// RegisterRoutes unifica e scandisce tutte le rotte del sistema in un unico punto esposto
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	// 1. Asset statici grafici
	mux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("html/img"))))

	// ==========================================
	// REGISTRAZIONE ROTTE DEI MODULI SPECIFICI
	// ==========================================

	// Delega tutte le rotte utente (login, logout, signup, modificaPassword) al pacchetto users
	users.RegisterRoutes(mux, db)

	// ==========================================
	// HANDLER UNICO CENTRALIZZATO PER LA RADICE (/) E UTILITY
	// ==========================================
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// GESTIONE FILE STATICI ESPLICITI (Incluso login, logout, signup e creausers)
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
					render.Error404Handler(w, r)
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

		// ROTTA DINAMICA DELLA DASHBOARD
		if r.URL.Path == "/dashboard" {
			dashboard.RenderDashboard(w, r)
			return
		}

		// TOOLS ASSISTENZA: Mappa Albero File
		if r.URL.Path == "/Structure" {
			dashboard.StructureHandler(w, r)
			return
		}

		// TOOLS ASSISTENZA: Visualizzatore Todo (NEWTAGS.md)
		if r.URL.Path == "/view-todo" {
			dashboard.ViewMarkdownHandler(w, r, "NEWTAGS.md", "Lista TODO", "Pianificazione dello sviluppo, middleware e regole generali del progetto")
			return
		}

		// TOOLS ASSISTENZA: Visualizzatore Changelog (CHANGELOG.md)
		if r.URL.Path == "/view-tags" {
			dashboard.ViewMarkdownHandler(w, r, "CHANGELOG.md", "Registro Storico Rilasci (Changelog)", "Revisione dei tag di produzione e cache bursting applicativo")
			return
		}

		// Tabella Utenti
		if r.URL.Path == "/allUsers" {
			dashboard.AllUsersHandler(w, r)
			return
		}

		// Clic del pulsante "Invio email"
		if r.URL.Path == "/send-test-email" {
			dashboard.SendTestEmailHandler(w, r)
			return
		}

		// SIMULAZIONE ERRORE 404
		if r.URL.Path == "/test-error-404" {
			render.Error404Handler(w, r)
			return
		}

		// SIMULAZIONE ERRORE 500
		if r.URL.Path == "/test-error-500" {
			errFake := fmt.Errorf("TEST ECCEZIONE 500 - Verifica allerta Mailer e pagina di cortesia")
			render.HandleTemplateError(w, r, errFake)
			return
		}

		// STRUMENTO: Lettura e formattazione del diario in ordine cronologico inverso
		if r.URL.Path == "/development-log" {
			dashboard.DevelopmentLogHandler(w, r)
			return
		}

		// Fallback definitivo 404 se nessuna risorsa statica o dinamica combacia
		render.Error404Handler(w, r)
	})
}

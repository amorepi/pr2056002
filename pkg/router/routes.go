package router

import (
	"net/http"
	"os"
	"pr2056002/pkg/dashboard"
	"pr2056002/pkg/render"
)

// RegisterRoutes unifica e scandisce tutte le rotte del sistema in un unico punto esposto
func RegisterRoutes(mux *http.ServeMux) {
	// 1. Asset statici grafici
	mux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("html/img"))))

	// 2. Handler unico centralizzato (Controllo di precedenza file statici espliciti e smistamento)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// A. GESTIONE FILE STATICI ESPLICITI (es. /index.html)
		if r.URL.Path != "/" && r.URL.Path != "/dashboard" && r.URL.Path != "/Structure" && r.URL.Path != "/view-todo" && r.URL.Path != "/view-tags" {
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

		// B. ROTTA DINAMICA DELLA RADICE (/) -> Home Page
		if r.URL.Path == "/" {
			render.RenderTemplate(w, r, "home.html", http.StatusOK)
			return
		}

		// C. ROTTA DINAMICA DELLA DASHBOARD
		if r.URL.Path == "/dashboard" {
			dashboard.RenderDashboard(w, r)
			return
		}

		// D. TOOLS ASSISTENZA: Mappa Albero File
		if r.URL.Path == "/Structure" {
			dashboard.StructureHandler(w, r)
			return
		}

		// E. TOOLS ASSISTENZA: Visualizzatore Todo (NEWTAGS.md)
		if r.URL.Path == "/view-todo" {
			dashboard.ViewMarkdownHandler(w, r, "NEWTAGS.md", "Lista TODO", "Pianificazione dello sviluppo, middleware e regole generali del progetto")
			return
		}

		// F. TOOLS ASSISTENZA: Visualizzatore Changelog (CHANGELOG.md)
		if r.URL.Path == "/view-tags" {
			dashboard.ViewMarkdownHandler(w, r, "CHANGELOG.md", "Registro Storico Rilasci (Changelog)", "Revisione dei tag di produzione e cache bursting applicativo")
			return
		}

		// Fallback definitivo 404 se nessuna risorsa statica o dinamica combacia
		render.Error404Handler(w, r)
	})
}

package render

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"
	// Importa il tuo pacchetto mailer se necessario, o usa una funzione interna
	// "pr2056002/pkg/utils"
)

// Variabile globale ereditata dal main per la versione
var CurrentVersion string

type PageContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserName    string
	Version     string
}

// Layout genera la struttura agnostica a tre colonne legata a Bootstrap 5.3
func Layout(content string, userName string, version string) string {
	project := os.Getenv("PROJECT")
	logoUrl := "/img/" + os.Getenv("LOGO_PROJ")
	avatarUrl := "/img/" + os.Getenv("AVATAR_PROJ")

	if userName == "" {
		userName = "Amministratore"
	}
	if version == "" {
		version = "v1.0.0"
	}

	// 1. Legge il file (restituisce un array di byte e un eventuale errore)
	data, err := os.ReadFile("testo.html")
	if err != nil {
		fmt.Println("Errore durante la lettura del file:", err)
		// return
	}
	// 2. Converte i byte in stringa e li assegna alla variabile
	htmlContent := string(data)

	return fmt.Sprintf(htmlContent, project, logoUrl, version, userName, avatarUrl, content)
}

// Questa funzione unisce i dati della testa con il corpo del template in basso
func RenderTemplate(w http.ResponseWriter, r *http.Request, templatePage string, statusCode int) {
	project := os.Getenv("PROJECT")
	logoUrl := "/img/" + os.Getenv("LOGO_PROJ")
	avatarUrl := "/img/" + os.Getenv("AVATAR_PROJ")

	data := PageContext{
		ProjectName: project,
		LogoUrl:     logoUrl,
		AvatarUrl:   avatarUrl,
		UserName:    "Amministratore",
		Version:     CurrentVersion,
	}

	layoutPath := "html/templates/layout.html"
	pagePath := "html/templates/" + templatePage

	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmplt, err := template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		log.Printf("ERRORE [Fatturato Parsing HTML] (%s, %s): %v", layoutPath, pagePath, err)

		// http.Error(w, "Errore caricamento componenti grafici", http.StatusInternalServerError)

		dir, errore := os.Getwd()
		if errore != nil {
			fmt.Println("Errore nel recupero del percorso:", errore)
			return
		}
		http.Error(w, "Errore reale: "+dir+" - "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmplt.ExecuteTemplate(w, "layout", data)
	if err != nil {
		log.Printf("ERRORE [Fallito Rendering Template 'layout']: %v", err)
		http.Error(w, "Errore interno durante il rendering della pagina", http.StatusInternalServerError)
		return
	}
}

// Error404Handler isolato e pertinente
func Error404Handler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"success":false,"error":"Risorsa non trovata","path":"%s"}`, r.URL.Path)
		return
	}
	RenderTemplate(w, r, "404.html", http.StatusNotFound)
}

// HandleTemplateError gestisce centralmente gli errori separando l'output tra browser e file log
func HandleTemplateError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	// 1. Cattura dello Stack Trace e data/ora attuali
	stackTrace := string(debug.Stack())
	nowStr := time.Now().Format("2006-01-02 15:04:05")

	// 2. Controllo dell'ambiente basato sulla presenza di /var/www
	_, statErr := os.Stat("/var/www")
	isProduction := statErr == nil

	if !isProduction {
		// ==========================================
		// AMBIENTE DI SVILUPPO (Mac)
		// ==========================================

		// Scrittura strutturata ed esplicita nel file di log locale (senza sporcare la shell)
		logFile, openErr := os.OpenFile("development.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if openErr == nil {
			defer logFile.Close()

			// Costruzione dell'intestazione richiesta
			fmt.Fprintf(logFile, "========================================================================\n")
			fmt.Fprintf(logFile, "VERSIONE: %s - DATA/ORA: %s - ROTTA: %s\n", CurrentVersion, nowStr, r.URL.Path)
			fmt.Fprintf(logFile, "DESCRIZIONE SINTETICA: %v\n", err)
			fmt.Fprintf(logFile, "------------------------------------------------------------------------\n")
			fmt.Fprintf(logFile, "TRACCIA COMPLETA:\n%s\n", stackTrace)
		}

		// Output minimale, pulito e focalizzato per il Browser
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `
			<div style="font-family: system-ui, -apple-system, sans-serif; padding: 25px; background: #fff5f5; color: #c53030; border: 1px solid #feb2b2; border-radius: 8px; max-width: 800px; margin: 30px auto; box-shadow: 0 4px 6px rgba(0,0,0,0.05);">
				<h4 style="margin: 0 0 10px 0; font-size: 18px; display: flex; align-items: center; gap: 8px;">❌ Errore Template (Mac)</h4>
				<div style="font-family: monospace; background: #fff; padding: 15px; border: 1px solid #dee2e6; border-radius: 4px; font-size: 14px; line-height: 1.5; color: #333;">
					%v
				</div>
				<p style="margin: 15px 0 0 0; font-size: 12px; color: #718096;">I dettagli completi sono stati archiviati in <code style="background:#edf2f7; padding:2px 4px; border-radius:3px;">development.log</code></p>
			</div>
		`, err)
		return
	}

	// ==========================================
	// AMBIENTE DI PRODUZIONE (VPS)
	// ==========================================
	w.WriteHeader(http.StatusInternalServerError)

	if data, readErr := os.ReadFile("html/templates/500.html"); readErr == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	} else {
		w.Write([]byte("Errore interno del server."))
	}

	// Invio asincrono in background via Goroutine per la VPS
	go func(errText string, trace string, path string) {
		project := os.Getenv("PROJECT")
		// Qui si aggancia l'invio mail reale per la produzione
		_ = project
		_ = errText
		_ = trace
		_ = path
	}(err.Error(), stackTrace, r.URL.Path)
}

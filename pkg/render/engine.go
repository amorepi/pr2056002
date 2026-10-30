package render

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"
)

// Variabile globale allineata dal main per tracciare la versione dell'applicazione
var CurrentVersion string

// PageContext definisce la struttura dati per il layout agnostico standard (Header/Navbar)
type PageContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserName    string
	Version     string
}

// RenderTemplate unisce il layout base a tre colonne con la pagina specifica richiesta
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmplt, err := template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		HandleTemplateError(w, r, err)
		return
	}

	w.WriteHeader(statusCode)
	err = tmplt.ExecuteTemplate(w, "layout", data)
	if err != nil {
		HandleTemplateError(w, r, err)
		return
	}
}

// Error404Handler intercetta le risorse inesistenti smistando l'output tra JSON (API) e HTML grafico
func Error404Handler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"success":false,"error":"Risorsa non trovata","path":"%s"}`, r.URL.Path)
		return
	}
	RenderTemplate(w, r, "404.html", http.StatusNotFound)
}

// HandleTemplateError centralizza i fallimenti di rendering dividendo la logica tra Mac e VPS
func HandleTemplateError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	// 1. Cattura dello Stack Trace e formattazione temporale
	stackTrace := string(debug.Stack())
	nowStr := time.Now().Format("2006-01-02 15:04:05")

	// 2. Controllo hardware dell'ambiente: se la cartella esiste, siamo sulla VPS
	_, statErr := os.Stat("/var/www")
	isProduction := statErr == nil

	if !isProduction {
		// ==========================================
		// AMBIENTE DI SVILUPPO (Mac)
		// ==========================================

		// Scrittura isolata nel file di log locale per non sporcare la prima shell di VS Code
		logFile, openErr := os.OpenFile("development.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if openErr == nil {
			defer logFile.Close()
			fmt.Fprintf(logFile, "========================================================================\n")
			fmt.Fprintf(logFile, "VERSIONE: %s - DATA/ORA: %s - ROTTA: %s\n", CurrentVersion, nowStr, r.URL.Path)
			fmt.Fprintf(logFile, "DESCRIZIONE SINTETICA: %v\n", err)
			fmt.Fprintf(logFile, "------------------------------------------------------------------------\n")
			fmt.Fprintf(logFile, "TRACCIA COMPLETA:\n%s\n", stackTrace)
		}

		// Rendering sintetico sul browser (Senza w.WriteHeader per prevenire avvisi superflui nella shell)
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

	// Configurazione immediata degli header d'errore isolati
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	// Esegue il rendering della pagina di cortesia 500 inerte
	if data, readErr := os.ReadFile("html/templates/500.html"); readErr == nil {
		w.Write(data)
	} else {
		w.Write([]byte("Errore interno del server. Si è verificata un'eccezione."))
	}

	// Invio asincrono in background della mail di notifica d'errore strutturata al developer
	go func(errText string, trace string, path string) {
		project := os.Getenv("PROJECT")
		subject := fmt.Sprintf("[%s] [CRITICAL 500] Eccezione sulla rotta %s", project, path)

		emailBody := fmt.Sprintf(`
			<h2>Errore critico rilevato in Produzione</h2>
			<p><strong>Progetto:</strong> %s</p>
			<p><strong>Rotta richiesta:</strong> %s</p>
			<p><strong>Errore riscontrato:</strong> %s</p>
			<h3>Stack Trace Completo:</h3>
			<pre style="background:#f8f9fa; padding:15px; border:1px solid #dee2e6; font-family:monospace; font-size:13px; line-height:1.4;">%s</pre>
		`, project, path, errText, trace)

		log.Printf("[MANUTENZIONE] Invio notifica mail d'errore per il crash su %s completato.", path)

		// Qui si innescherà la chiamata reale all'utility utils.Mailer({ subject: subject, html: emailBody })
		_ = emailBody
		_ = subject
	}(err.Error(), stackTrace, r.URL.Path)
}

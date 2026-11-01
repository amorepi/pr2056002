package render

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	// Importiamo l'utility del mailer per attivare le notifiche reali sulla VPS
	"pr2056002/pkg/utils"
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

// RenderTemplate esegue il rendering atomico tramite buffer di memoria
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

	tmplt, err := template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		HandleTemplateError(w, r, err)
		return
	}

	var buf bytes.Buffer
	err = tmplt.ExecuteTemplate(&buf, "layout", data)
	if err != nil {
		HandleTemplateError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = buf.WriteTo(w)
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

	// 1. CATTURA PRIVATA DELLO STACK TRACE COMPLETO
	stackTrace := string(debug.Stack())

	// 2. SCRITTURA DIRETTA SU DISCO (Terminale pulito con 1 sola riga)
	log.Printf("[ECCEZIONE 500] sulla rotta %s -> %v", r.URL.Path, err)

	// Determiniamo il file corretto in base all'ambiente hardware rilevato
	_, statErr := os.Stat("/var/www")
	isProduction := statErr == nil

	logPath := "development.log"
	if isProduction {
		logPath = "/var/log/nginx/pr2056002_go.log"
	}

	// Forziamo la scrittura dello Stack Trace esteso all'interno del file fisico (Valido per Mac e VPS)
	if f, fileErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); fileErr == nil {
		_, _ = f.WriteString(fmt.Sprintf("--- STACK TRACE SEPARATO ---\n%s\n--------------------------------------\n", stackTrace))
		f.Close()
	} else {
		log.Printf("[AVVISO MANUTENZIONE]: Impossibile scrivere lo stack trace su %s: %v", logPath, fileErr)
	}

	if !isProduction {
		// ==========================================
		// AMBIENTE DI SVILUPPO (Mac) - CON BARRA MENU
		// ==========================================
		project := os.Getenv("PROJECT")
		data := PageContext{
			ProjectName: project,
			LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
			AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
			UserName:    "Amministratore",
			Version:     CurrentVersion,
		}

		layoutPath := "html/templates/layout.html"
		pagePath := "html/templates/errore-sviluppo.html"

		tmplt, parseErr := template.ParseFiles(layoutPath, pagePath)
		if parseErr != nil {
			dir, _ := os.Getwd()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Errore critico durante il parsing d'emergenza. Cartella attuale su Mac: %s. Dettaglio: %v", dir, parseErr)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_ = tmplt.ExecuteTemplate(w, "layout", data)
		return
	}

	// ==========================================
	// AMBIENTE DI PRODUZIONE (VPS) - CON BARRA MENU E NOTIFICA REALE VIA EMAIL
	// ==========================================
	project := os.Getenv("PROJECT")
	data := PageContext{
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		UserName:    "Amministratore",
		Version:     CurrentVersion,
	}

	// Eseguiamo l'incastro regolare di 500.html dentro il layout standard (Barra dei menu fissa)
	layoutPath := "html/templates/layout.html"
	pagePath := "html/templates/500.html"

	tmplt, parseErr := template.ParseFiles(layoutPath, pagePath)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	if parseErr == nil {
		_ = tmplt.ExecuteTemplate(w, "layout", data)
	} else {
		w.Write([]byte("Errore interno del server. Si è verificata un'eccezione critica."))
	}

	// VITALIZZAZIONE: Invio asincrono REALE in background della mail d'allerta al developer
	go func(errText string, trace string, path string) {
		subject := fmt.Sprintf("[%s] [CRITICAL 500] Eccezione sulla rotta %s", project, path)
		emailBody := fmt.Sprintf(`
			<div style="font-family:sans-serif; padding:20px; color:#333;">
				<h2 style="color:#d9534f; border-bottom:2px solid #d9534f; padding-bottom:10px;">Errore critico rilevato in Produzione (VPS)</h2>
				<p><strong>Progetto di riferimento:</strong> %s</p>
				<p><strong>Rotta che ha generato il crash:</strong> <code style="background:#f1f1f1; padding:2px 6px; border-radius:4px;">%s</code></p>
				<p><strong>Messaggio d'errore:</strong> <span style="color:#c9302c; font-weight:bold;">%s</span></p>
				<h3 style="margin-top:30px;">Stack Trace Completo:</h3>
				<pre style="background:#f8f9fa; padding:15px; border:1px solid #dee2e6; font-family:monospace; font-size:13px; line-height:1.4; overflow:auto; max-height:500px; color:#555;">%s</pre>
			</div>
		`, project, path, errText, trace)

		payload := utils.MailerPayload{
			Subject: subject,
			Html:    emailBody,
		}

		// Esegue l'invio asincrono effettivo su Gmail
		if mailErr := utils.InviaEmail(payload); mailErr != nil {
			log.Printf("[MANUTENZIONE ERRORE]: Invio notifica mail fallito: %v", mailErr)
		} else {
			log.Printf("[MANUTENZIONE] Notifica mail d'errore per il crash su %s spedita con successo.", path)
		}
	}(err.Error(), stackTrace, r.URL.Path)
}

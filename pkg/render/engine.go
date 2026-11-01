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
		// Pulizia di sicurezza: isoliamo i testi per evitare che si tirino dietro script o stili della pagina
		pulitoErrText := template.HTMLEscapeString(errText)

		// Per lo Stack Trace, sostituiamo i ritorni a capo con tag <br> per renderlo leggibile nella mail
		// ma lo manteniamo isolato dentro un blocco di testo puro
		pulitoTrace := template.HTMLEscapeString(trace)

		subject := fmt.Sprintf("[%s] [CRITICAL 500] Eccezione sulla rotta %s", project, path)

		// Confezioniamo un HTML super leggero, privo di script, css o stili Bootstrap esterni
		emailBody := fmt.Sprintf(`
			<div style="font-family:sans-serif; padding:20px; color:#333; max-width:800px; margin:0 auto; background:#fff; border:1px solid #eee; border-radius:5px;">
				<h2 style="color:#d9534f; border-bottom:2px solid #d9534f; padding-bottom:10px; margin-top:0;">⚠️ Allerta Errore Critico 500 (VPS)</h2>
				<p style="font-size:15px; margin-bottom:8px;"><strong>Progetto applicativo:</strong> %s</p>
				<p style="font-size:15px; margin-bottom:8px;"><strong>Rotta del crash:</strong> <code style="background:#f8f9fa; padding:4px 8px; border-radius:4px; border:1px solid #ddd; font-family:monospace;">%s</code></p>
				<p style="font-size:15px; margin-bottom:20px;"><strong>Messaggio di eccezione:</strong> <span style="color:#c9302c; font-weight:bold;">%s</span></p>
				
				<h3 style="font-size:16px; color:#555; margin-bottom:10px; border-top:1px solid #eee; pt-15px;">Dettaglio Stack Trace Tecnico:</h3>
				<pre style="background:#212529; color:#4af626; padding:15px; border-radius:4px; font-family:monospace; font-size:12px; line-height:1.5; overflow:auto; max-height:400px;">%s</pre>
			</div>
		`, project, path, pulitoErrText, pulitoTrace)

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

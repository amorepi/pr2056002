package render

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	"pr2056002/pkg/utils"
)

// ==========================================
// Renderizzazione di un Template
// Riceve data dall'esterno per mantenere la sessione integra
// ==========================================
func RenderTemplate(w http.ResponseWriter, r *http.Request, templatePage string, statusCode int, data interface{}) {
	layoutPath := "html/templates/layout.html"
	pagePath := "html/templates/" + templatePage

	tmplt, err := template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		HandleTemplateError(w, r, err, data)
		return
	}

	var buf bytes.Buffer
	err = tmplt.ExecuteTemplate(&buf, "layout", data)
	if err != nil {
		HandleTemplateError(w, r, err, data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = buf.WriteTo(w)
}

// ==========================================
// Renderizzazione del Template Errore 404
// Riceve data per preservare l'estetica della barra superiore
// ==========================================
func Error404Handler(w http.ResponseWriter, r *http.Request, data interface{}) {
	if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"success":false,"error":"Risorsa non trovata","path":"%s"}`, r.URL.Path)
		return
	}
	RenderTemplate(w, r, "404.html", http.StatusNotFound, data)
}

// ==========================================
// Handle all'errore di un Template (500)
// Riceve data per mantenere attiva l'autenticazione a video
// ==========================================
func HandleTemplateError(w http.ResponseWriter, r *http.Request, err error, data interface{}) {
	if err == nil {
		return
	}

	stackTrace := string(debug.Stack())
	log.Printf("[ECCEZIONE 500] sulla rotta %s -> %v", r.URL.Path, err)

	_, statErr := os.Stat("/var/www")
	isProduction := statErr == nil

	logPath := "development.log"
	if isProduction {
		logPath = "/var/log/nginx/pr2056002_go.log"
	}

	if f, fileErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); fileErr == nil {
		_, _ = f.WriteString(fmt.Sprintf("--- STACK TRACE SEPARATO ---\n%s\n--------------------------------------\n", stackTrace))
		f.Close()
	}

	project := os.Getenv("PROJECT")
	if project == "" {
		project = "PR2056002"
	}

	// ==========================================
	// avvia una goroutine: la funzione viene eseguita in background e in modo asincrono
	// (in parallelo) rispetto al resto del programma.
	// Spedisce SEMPRE l'email asincrona di allerta, sia in locale che in produzione
	// ==========================================
	go func(errText string, trace string, path string) {
		mailTemplatePath := "html/templates/alert-mail.html"
		tmpltMail, err := template.ParseFiles(mailTemplatePath)
		if err != nil {
			return
		}

		mailData := struct {
			ProjectName  string
			Path         string
			ErrorMessage string
			StackTrace   string
		}{
			ProjectName:  project,
			Path:         path,
			ErrorMessage: errText,
			StackTrace:   trace,
		}

		var mailBuffer bytes.Buffer
		if err := tmpltMail.Execute(&mailBuffer, mailData); err != nil {
			return
		}

		payload := utils.MailerPayload{
			Subject: fmt.Sprintf("[%s] [CRITICAL 500] Eccezione sulla rotta %s", project, path),
			Html:    mailBuffer.String(),
		}

		_ = utils.InviaEmail(payload)
	}(err.Error(), stackTrace, r.URL.Path)

	layoutPath := "html/templates/layout.html"
	var pagePath string

	if !isProduction {
		pagePath = "html/templates/errore-sviluppo.html"
	} else {
		pagePath = "html/templates/500.html"
	}

	tmplt, parseErr := template.ParseFiles(layoutPath, pagePath)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	if parseErr == nil {
		_ = tmplt.ExecuteTemplate(w, "layout", data)
	} else {
		w.Write([]byte("Errore interno del server. Si è verificata un'eccezione critica."))
	}
}

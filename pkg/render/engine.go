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

var CurrentVersion string

type PageContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	Autenticato bool
	Version     string
}

type MailAlertContext struct {
	ProjectName  string
	Path         string
	ErrorMessage string
	StackTrace   string
}

func RenderTemplate(w http.ResponseWriter, r *http.Request, templatePage string, statusCode int) {
	project := os.Getenv("PROJECT")
	logoUrl := "/img/" + os.Getenv("LOGO_PROJ")
	avatarUrl := "/img/" + os.Getenv("AVATAR_PROJ")

	data := PageContext{
		ProjectName: project,
		LogoUrl:     logoUrl,
		AvatarUrl:   avatarUrl,
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

func Error404Handler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"success":false,"error":"Risorsa non trovata","path":"%s"}`, r.URL.Path)
		return
	}
	RenderTemplate(w, r, "404.html", http.StatusNotFound)
}

func HandleTemplateError(w http.ResponseWriter, r *http.Request, err error) {
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

	if !isProduction {
		data := PageContext{
			ProjectName: project,
			LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
			AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
			Version:     CurrentVersion,
		}

		layoutPath := "html/templates/layout.html"
		pagePath := "html/templates/errore-sviluppo.html"

		tmplt, parseErr := template.ParseFiles(layoutPath, pagePath)
		if parseErr != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Errore critico durante il parsing d'emergenza: %v", parseErr)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_ = tmplt.ExecuteTemplate(w, "layout", data)
		return
	}

	data := PageContext{
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		Version:     CurrentVersion,
	}

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

	go func(errText string, trace string, path string) {
		mailTemplatePath := "html/templates/alert-mail.html"
		tmpltMail, err := template.ParseFiles(mailTemplatePath)
		if err != nil {
			return
		}

		mailData := MailAlertContext{
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
}

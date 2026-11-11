package dashboard

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"os"

	"pr2056002/pkg/utils"
)

// EmailBodyContext definisce i parametri richiesti da test-mail.html
type EmailBodyContext struct {
	ProjectName string
}

// SendTestEmailHandler gestisce la lettura del template email, la compilazione e l'invio SMTP
func SendTestEmailHandler(w http.ResponseWriter, r *http.Request) {
	project, auth, role, ok := verificaAdminENavbar(w, r)
	if !ok {
		return
	}

	// 1. Lettura e compilazione dinamica del file HTML del corpo della mail
	emailTemplatePath := "html/templates/test-mail.html"
	tmpltEmail, err := template.ParseFiles(emailTemplatePath)
	if err != nil {
		log.Printf("[MAILER ERRORE]: Impossibile trovare o analizzare il template %s: %v", emailTemplatePath, err)
		mostraNotificaErrore(w, r, project, auth, role, "Configurazione Fallita", "Il server non ha trovato il file di template dell'email sul disco.")
		return
	}

	emailData := EmailBodyContext{
		ProjectName: project,
	}

	var emailBuffer bytes.Buffer
	if err := tmpltEmail.Execute(&emailBuffer, emailData); err != nil {
		log.Printf("[MAILER ERRORE]: Errore durante la compilazione del codice HTML della mail: %v", err)
		mostraNotificaErrore(w, r, project, auth, role, "Compilazione Fallita", "Impossibile formattare i parametri all'interno della mail.")
		return
	}

	payload := utils.MailerPayload{
		Subject: ("[" + project + "] - Verifica Connettività Mailer Completata"),
		Html:    emailBuffer.String(),
	}

	// 2. Invio fisico dell'email tramite l'utility
	err = utils.InviaEmail(payload)

	// 3. Preparazione del contesto per la risposta web grafica (notifica.html)
	context := NotificaContext{
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		Autenticato: auth,
		UserRole:    role,
		Version:     utils.CurrentVersion,
	}

	if err != nil {
		log.Printf("[MAILER ERRORE]: Invio SMTP fallito: %v", err)
		context.Title = "Invio Email Fallito"
		context.Subtitle = ("Impossibile completare l'operazione SMTP. Dettaglio: " + err.Error())
	} else {
		context.Title = "Email inviata con successo"
		context.Subtitle = "Il messaggio di test è stato consegnato al server SMTP. Controlla la tua casella Gmail."
	}

	eseguiTemplateProtetto(w, r, "html/templates/notifica.html", context)
}

// Helper privato per dirottare istantaneamente l'utente su una notifica d'errore web controllata
func mostraNotificaErrore(w http.ResponseWriter, r *http.Request, project string, auth bool, role, titolo, sottotitolo string) {
	context := NotificaContext{
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		Autenticato: auth,
		UserRole:    role,
		Version:     utils.CurrentVersion,
		Title:       titolo,
		Subtitle:    sottotitolo,
	}
	eseguiTemplateProtetto(w, r, "html/templates/notifica.html", context)
}

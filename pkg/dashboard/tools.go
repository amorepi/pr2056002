package dashboard

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pr2056002/pkg/render"
)

// StructureContext definisce i parametri per la visualizzazione dell'albero dei file
type StructureContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserName    string
	Version     string
	TreeOutput  string
}

// MarkdownContext definisce i parametri per la visualizzazione dei file documentali markdown
type MarkdownContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserName    string
	Version     string
	Title       string
	Subtitle    string
	HtmlContent template.HTML
}

// NotificaContext definisce lo schema per le pagine di risposta di successo o errore (Usata anche da send-email.go)
type NotificaContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserName    string
	Version     string
	Title       string
	Subtitle    string
}

// StructureHandler genera la mappa dei file richiamando BuildTree
func StructureHandler(w http.ResponseWriter, r *http.Request) {
	project := os.Getenv("PROJECT")
	currentDirectory, _ := os.Getwd()
	if project == "" {
		project = filepath.Base(currentDirectory)
	}

	treeOutput := fmt.Sprintf("%s/\n%s", project, BuildTree(currentDirectory, ""))

	data := StructureContext{
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		UserName:    "Amministratore",
		Version:     render.CurrentVersion,
		TreeOutput:  treeOutput,
	}

	eseguiTemplateProtetto(w, r, "html/templates/structure.html", data)
}

// ViewMarkdownHandler visualizza a schermo il contenuto convertito dei file di testo
func ViewMarkdownHandler(w http.ResponseWriter, r *http.Request, targetFile, title, subtitle string) {
	data, err := os.ReadFile(targetFile)
	var contenuto template.HTML
	if err != nil {
		contenuto = template.HTML(`<div class="text-muted italic">Impossibile accedere al file o risorsa mancante.</div>`)
	} else {
		conversione := strings.ReplaceAll(string(data), "\n", "<br>")
		contenuto = template.HTML(conversione)
	}

	context := MarkdownContext{
		ProjectName: os.Getenv("PROJECT"),
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		UserName:    "Amministratore",
		Version:     render.CurrentVersion,
		Title:       title,
		Subtitle:    subtitle,
		HtmlContent: contenuto,
	}

	eseguiTemplateProtetto(w, r, "html/templates/markdown.html", context)
}

// UserRow definisce lo schema reale per i record estratti dal catalogo SQL aggiornato
type UserRow struct {
	ID        int
	Username  string
	Email     string
	Cellulare string
	Role      string
	Attivo    string // Usiamo stringa ("Attivo"/"Disabilitato") per facilitare il rendering in users.html
	CreatoIl  string
}

// UsersContext raccoglie l'anagrafica degli utenti estratti da PostgreSQL 17
type UsersContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserName    string
	Version     string
	Users       []UserRow
}

// AllUsersHandler interroga PostgreSQL 17 ed estrae in tempo reale le tue credenziali amministrative
func AllUsersHandler(w http.ResponseWriter, r *http.Request) {
	var utentiReali []UserRow

	// Query strutturata sui nuovi campi reali della tabella utenti
	query := "SELECT id, username, email, cellulare, ruolo, attivo, creato_il FROM users ORDER BY id ASC"
	rows, err := db.Query(query)

	if err != nil {
		// Se la tabella non esiste (sistema non ancora inizializzato con /creausers), intercettiamo l'avviso
		log.Printf("[DASHBOARD NOTA]: Tabella utenti non presente o in attesa di inizializzazione: %v", err)
		utentiReali = []UserRow{
			{
				ID:        0,
				Username:  "In attesa di attivazione",
				Email:     "Tabella non inizializzata nel DB",
				Cellulare: "N/D",
				Role:      "guest",
				Attivo:    "Disabilitato",
				CreatoIl:  time.Now().Format("02/01/2006 15:04"),
			},
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var u UserRow
			var attivoBool bool
			var creatoIlTime time.Time

			err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Cellulare, &u.Role, &attivoBool, &creatoIlTime)
			if err != nil {
				log.Printf("[DB ERRORE]: Scansione record fallita: %v", err)
				continue
			}

			// Trasformiamo i tipi complessi in stringhe pulite per il template esistente
			u.CreatoIl = creatoIlTime.Format("02/01/2006 15:04")
			u.Attivo = "Disabilitato"
			if attivoBool {
				u.Attivo = "Attivo"
			}

			utentiReali = append(utentiReali, u)
		}
	}

	context := UsersContext{
		ProjectName: os.Getenv("PROJECT"),
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		UserName:    "Amministratore",
		Version:     render.CurrentVersion,
		Users:       utentiReali,
	}

	eseguiTemplateProtetto(w, r, "html/templates/users.html", context)
}

// eseguiTemplateProtetto analizza ed emette i moduli grafici (Visibile all'interno del pacchetto dashboard)
func eseguiTemplateProtetto(w http.ResponseWriter, r *http.Request, pagePath string, data interface{}) {
	layoutPath := "html/templates/layout.html"
	tmplt, err := template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Errore caricamento template componenti: " + err.Error()))
		return
	}

	var buf bytes.Buffer
	err = tmplt.ExecuteTemplate(&buf, "layout", data)
	if err != nil {
		render.HandleTemplateError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}

// Legge il diario di Go e lo formatta in ordine cronologico inverso
func DevelopmentLogHandler(w http.ResponseWriter, r *http.Request) {
	logPath := os.Getenv("LOG_PATH_GO")
	var righeInvertite []string

	data, err := os.ReadFile(logPath)
	if err != nil {
		righeInvertite = append(righeInvertite, "[AVVISO]: Il file development.log non è ancora stato generato o è inaccessibile.")
	} else {
		linee := strings.Split(strings.TrimSpace(string(data)), "\n")
		for i := len(linee) - 1; i >= 0; i-- {
			if linee[i] != "" {
				righeInvertite = append(righeInvertite, linee[i])
			}
		}
	}

	limite := 50
	if len(righeInvertite) < limite {
		limite = len(righeInvertite)
	}
	outputFinale := strings.Join(righeInvertite[:limite], "\n")

	context := StructureContext{
		ProjectName: os.Getenv("PROJECT"),
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		UserName:    "Amministratore",
		Version:     render.CurrentVersion,
		TreeOutput:  outputFinale,
	}

	eseguiTemplateProtetto(w, r, "html/templates/structure.html", context)
}

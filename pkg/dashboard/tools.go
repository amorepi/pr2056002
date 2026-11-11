package dashboard

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"pr2056002/pkg/render"
	"pr2056002/pkg/users"
	"pr2056002/pkg/utils"
)

// Helper interno per verificare il ruolo admin e popolare i dati base della navbar
func verificaAdminENavbar(w http.ResponseWriter, r *http.Request) (string, bool, string, bool) {
	payload, err := users.GetSessionPayload(r)
	if err != nil || !payload.Attivo || payload.Ruolo != "admin" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return "", false, "", false
	}
	project := os.Getenv("PROJECT")
	if project == "" {
		project = "PR2056002"
	}
	return project, true, "Amministratore", true
}

// StructureContext definisce i parametri per la visualizzazione dell'albero dei file
type StructureContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	Autenticato bool
	UserRole    string
	Version     string
	TreeOutput  string
}

// MarkdownContext definisce i parametri per la visualizzazione dei file documentali markdown
type MarkdownContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	Autenticato bool
	UserRole    string
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
	Autenticato bool
	UserRole    string
	Version     string
	Title       string
	Subtitle    string
}

// StructureHandler genera la mappa dei file richiamando BuildTree
func StructureHandler(w http.ResponseWriter, r *http.Request) {
	project, auth, role, ok := verificaAdminENavbar(w, r)
	if !ok {
		return
	}

	currentDirectory, _ := os.Getwd()
	treeOutput := fmt.Sprintf("%s/\n%s", project, BuildTree(currentDirectory, ""))

	data := StructureContext{
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		Autenticato: auth,
		UserRole:    role,
		Version:     utils.CurrentVersion,
		TreeOutput:  treeOutput,
	}

	eseguiTemplateProtetto(w, r, "html/templates/structure.html", data)
}

// ViewMarkdownHandler visualizza a schermo il contenuto convertito dei file di testo
func ViewMarkdownHandler(w http.ResponseWriter, r *http.Request, targetFile, title, subtitle string) {
	project, auth, role, ok := verificaAdminENavbar(w, r)
	if !ok {
		return
	}

	data, err := os.ReadFile(targetFile)
	var contenuto template.HTML
	if err != nil {
		contenuto = template.HTML(`<div class="text-muted italic">Impossibile accedere al file o risorsa mancante.</div>`)
	} else {
		conversione := strings.ReplaceAll(string(data), "\n", "<br>")
		contenuto = template.HTML(conversione)
	}

	context := MarkdownContext{
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		Autenticato: auth,
		UserRole:    role,
		Version:     utils.CurrentVersion,
		Title:       title,
		Subtitle:    subtitle,
		HtmlContent: contenuto,
	}

	eseguiTemplateProtetto(w, r, "html/templates/markdown.html", context)
}

// UserRow definisce lo schema reale per i record estratti dal catalogo SQL aggiornato
type UserRow struct {
	ID        int
	Cognome   string
	Nome      string
	Email     string
	Cellulare string
	Role      string
	Attivo    string
	CreatoIl  string
}

// UsersContext raccoglie l'anagrafica degli utenti estratti da PostgreSQL 17
type UsersContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	Autenticato bool
	UserRole    string
	Version     string
	Users       []UserRow
}

// AllUsersHandler interroga PostgreSQL 17 ed estrae in tempo reale le credenziali
func AllUsersHandler(w http.ResponseWriter, r *http.Request) {
	project, auth, role, ok := verificaAdminENavbar(w, r)
	if !ok {
		return
	}

	var utentiReali []UserRow

	query := "SELECT id, cognome, nome, email, cellulare, ruolo, attivo, creato_il FROM users ORDER BY id ASC"
	rows, err := db.Query(query)

	if err != nil {
		log.Printf("[DASHBOARD NOTA]: Tabella utenti non presente o in attesa di inizializzazione: %v", err)
		utentiReali = []UserRow{
			{
				ID:        0,
				Cognome:   "-",
				Nome:      "-",
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

			err := rows.Scan(&u.ID, &u.Cognome, &u.Nome, &u.Email, &u.Cellulare, &u.Role, &attivoBool, &creatoIlTime)
			if err != nil {
				log.Printf("[DB ERRORE]: Scansione record fallita: %v", err)
				continue
			}

			u.CreatoIl = creatoIlTime.Format("02/01/2006 15:04")
			u.Attivo = "Disabilitato"
			if attivoBool {
				u.Attivo = "Attivo"
			}

			utentiReali = append(utentiReali, u)
		}
		if err = rows.Err(); err != nil {
			log.Printf("[ERRORE DB] Errore riscontrato durante l'iterazione dei record: %v", err)
			return
		}
	}

	context := UsersContext{
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		Autenticato: auth,
		UserRole:    role,
		Version:     utils.CurrentVersion,
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

// DevelopmentLogHandler legge il diario di Go e lo formatta in ordine cronologico inverso
func DevelopmentLogHandler(w http.ResponseWriter, r *http.Request) {
	project, auth, role, ok := verificaAdminENavbar(w, r)
	if !ok {
		return
	}

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
		ProjectName: project,
		LogoUrl:     "/img/" + os.Getenv("LOGO_PROJ"),
		AvatarUrl:   "/img/" + os.Getenv("AVATAR_PROJ"),
		Autenticato: auth,
		UserRole:    role,
		Version:     utils.CurrentVersion,
		TreeOutput:  outputFinale,
	}

	eseguiTemplateProtetto(w, r, "html/templates/structure.html", context)
}

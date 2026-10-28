package render

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
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

	fmt.Println("Passa da qui", err)

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

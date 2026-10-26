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
	logoEnv := os.Getenv("LOGO_PROJ")
	avatarEnv := os.Getenv("AVATAR_PROJ")

	logoUrl := "/img/logo_proj.png"
	if logoEnv != "" {
		logoUrl = "/img/" + logoEnv
	}

	avatarUrl := "/img/avatar_proj.png"
	if avatarEnv != "" {
		avatarUrl = "/img/" + avatarEnv
	}

	if userName == "" {
		userName = "Amministratore"
	}
	if version == "" {
		version = "v1.0.0"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="it">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Progetto %s</title>
  <link href="https://jsdelivr.net" rel="stylesheet">
</head>
<body class="bg-light">
  <header class="navbar fixed-top bg-white border-bottom shadow-sm py-2">
    <div class="container-fluid d-flex align-items-center justify-content-between">
      <div class="d-flex align-items-center gap-3">
        <a href="/" class="navbar-brand m-0 p-0">
          <img src="%s" alt="Logo" class="d-inline-block align-text-top" style="height: 36px;">
        </a>
        <a href="/" class="nav-link text-dark fw-medium">Home</a>
        <a href="/dashboard" class="nav-link text-dark fw-medium">Dashboard</a>
      </div>
      <div class="text-center">
        <span class="badge text-bg-secondary font-monospace px-2 py-1">%s</span>
      </div>
      <div class="d-flex align-items-center gap-3">
        <div class="position-relative" title="%s" style="width: 36px; height: 36px;">
          <img src="%s" alt="User Avatar" class="img-fluid rounded-circle border w-100 h-100 object-fit-cover">
        </div>
        <a href="/logout" class="btn btn-outline-danger btn-sm px-3">Esci</a>
      </div>
    </div>
  </header>
  <main class="container-fluid pt-5 mt-5">%s</main>
  <script src="https://jsdelivr.net"></script>
</body>
</html>`, project, logoUrl, version, userName, avatarUrl, content)
}

// RenderTemplate unisce i dati della testa con il corpo del template in basso
func RenderTemplate(w http.ResponseWriter, r *http.Request, templatePage string, statusCode int) {
	project := os.Getenv("PROJECT")
	logoEnv := os.Getenv("LOGO_PROJ")
	avatarEnv := os.Getenv("AVATAR_PROJ")

	logoUrl := "/img/MiniserviziLogo50.png"
	if logoEnv != "" {
		logoUrl = "/img/" + logoEnv
	}

	avatarUrl := "/img/User.png"
	if avatarEnv != "" {
		avatarUrl = "/img/" + avatarEnv
	}

	data := PageContext{
		ProjectName: project,
		LogoUrl:     logoUrl,
		AvatarUrl:   avatarUrl,
		UserName:    "Amministratore",
		Version:     CurrentVersion,
	}

	layoutPath := "./html/templates/layout.html"
	pagePath := "./html/templates/" + templatePage

	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmplt, err := template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		log.Printf("ERRORE [Fatturato Parsing HTML] (%s, %s): %v", layoutPath, pagePath, err)
		http.Error(w, "Errore caricamento componenti grafici", http.StatusInternalServerError)
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

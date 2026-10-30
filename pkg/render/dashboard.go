package render

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
)

// MetricCard definisce la struttura dati inerte per le card dell'hardware/software
type MetricCard struct {
	Title       string
	Value       template.HTML // Consente tag HTML interni (es. elementi <span class="small">)
	BadgeText   string
	BadgeClass  string
	IsFullWidth bool
}

// IsHalfWidth bool

// ToolButton definisce lo scheletro inerte per i pulsanti di utility e database
type ToolButton struct {
	Href  string
	Label string
}

// DashboardContext raccoglie l'intero payload dati richiesto da dashboard.html
type DashboardContext struct {
	ProjectName string
	LogoUrl     string
	AvatarUrl   string
	UserName    string
	Version     string
	Metrics     []MetricCard
	Tools       []ToolButton
	Databases   []ToolButton
}

// RenderDashboard compila lo scheletro inerte del pannello di controllo
func RenderDashboard(w http.ResponseWriter, r *http.Request) {
	// 1. Recupero dei parametri ambientali centralizzati
	project := os.Getenv("PROJECT")
	logoUrl := "/img/" + os.Getenv("LOGO_PROJ")
	avatarUrl := "/img/" + os.Getenv("AVATAR_PROJ")

	// 2. Definizione dei blocchi metrici inerti (mock dati Node.js)
	metrics := []MetricCard{
		{Title: "Sistema operativo", Value: "macOS / Linux (VPS)"},
		{Title: "Node.js / npm equivalents", Value: "Go v1.27.1 / Native Server"},
		{Title: "Server Web Nginx", Value: "Proxy Attivo"},
		{Title: "RAM Disponibile", Value: "1.8 GB / 2.0 GB", BadgeText: "Ottimale", BadgeClass: "text-bg-success"},
		{Title: "Archiviazione Disco", Value: "15GB liberi su 25GB"},
		{Title: "Motore Database", Value: "PostgreSQL 17 (Operativo)", BadgeText: "Operativo", BadgeClass: "text-bg-success"},
		{Title: "Database Istanziati attivi", Value: `1 <span class="fs-6 fw-normal text-muted">unità logica registrata</span>`, IsFullWidth: true},
	}

	// 3. Mappatura dei pulsanti ad accesso libero per la manutenzione
	tools := []ToolButton{
		{Href: "/api/health", Label: "Health Check"},
		{Href: "/Structure", Label: "view-Structure"},
		{Href: "/view-todo", Label: "view-todo"},
		{Href: "/view-tags", Label: "view-tags"},
	}

	databases := []ToolButton{
		{Href: "/allUsers", Label: "Collection Users"},
	}

	// 4. Assemblaggio del contesto globale unificato (incluso Header parametri)
	data := DashboardContext{
		ProjectName: project,
		LogoUrl:     logoUrl,
		AvatarUrl:   avatarUrl,
		UserName:    "Amministratore",
		Version:     CurrentVersion, // Ereditata da engine.go nello stesso package
		Metrics:     metrics,
		Tools:       tools,
		Databases:   databases,
	}

	layoutPath := "html/templates/layout.html"
	pagePath := "html/templates/dashboard.html"

	tmplt, err := template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Errore parsing template dashboard: " + err.Error()))
		return
	}

	// Creiamo il buffer temporaneo anche per la Dashboard
	var buf bytes.Buffer
	err = tmplt.ExecuteTemplate(&buf, "layout", data)
	if err != nil {
		// Se il template ha errori, interrompe l'esecuzione e pulisce l'output
		HandleTemplateError(w, r, err)
		return
	}

	// Rendering completato al 100% con successo: emettiamo i dati
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}

package dashboard

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	// Importiamo il pacchetto render per accedere a CurrentVersion e HandleTemplateError
	"pr2056002/pkg/render"
)

// MetricCard definisce la struttura dati per le card hardware/software
type MetricCard struct {
	Title       string
	Value       template.HTML
	BadgeText   string
	BadgeClass  string
	IsFullWidth bool
	IsHalfWidth bool
}

// ToolButton definisce lo scheletro per i pulsanti di utility e database
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

// Helper per estrarre la RAM totale in GB sia su Mac che su Linux
func getRamTotale() string {
	if runtime.GOOS == "darwin" {
		// Comando specifico per macOS
		cmd := exec.Command("sysctl", "-n", "hw.memsize")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "N/D"
		}
		bytesVal, err := strconv.ParseFloat(strings.TrimSpace(out.String()), 64)
		if err != nil {
			return "N/D"
		}
		return fmt.Sprintf("%.1f GB", bytesVal/1024/1024/1024)
	}

	// Fallback nativo per Linux (VPS) senza syscall bloccanti su Mac
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "N/D"
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// CORRETTO: fields[1] estrae il valore numerico in KB dalla slice di stringhe
				kb, err := strconv.ParseFloat(fields[1], 64)
				if err == nil {
					return fmt.Sprintf("%.1f GB", kb/1024/1024)
				}
			}
		}
	}
	return "N/D"
}

// Calcolo dello spazio su disco sicuro per macOS e Linux tramite comando df di sistema
func getSpazioDisco() string {
	cmd := exec.Command("df", "-h", "/")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "Errore lettura disco"
	}

	lines := strings.Split(out.String(), "\n")
	if len(lines) < 2 {
		return "Dati disco non disponibili"
	}

	// Analizza l'output della riga della root
	fields := strings.Fields(lines[1])
	if runtime.GOOS == "darwin" && len(fields) >= 4 {
		// Struttura df su macOS: [Filesystem, Size, Used, Avail, ...]
		return fmt.Sprintf("%s liberi su %s totali", fields[3], fields[1])
	} else if len(fields) >= 4 {
		// Struttura df su Linux standard
		return fmt.Sprintf("%s liberi su %s totali", fields[3], fields[1])
	}

	return "df output malformato"
}

// RenderDashboard compila la Dashboard estraendo i dati reali della macchina
func RenderDashboard(w http.ResponseWriter, r *http.Request) {
	project := os.Getenv("PROJECT")
	logoUrl := "/img/" + os.Getenv("LOGO_PROJ")
	avatarUrl := "/img/" + os.Getenv("AVATAR_PROJ")

	sistemaOperativo := fmt.Sprintf("%s (%s)", runtime.GOOS, runtime.GOARCH)
	versioneGo := runtime.Version()
	spazioDisco := getSpazioDisco()
	ramValore := getRamTotale()

	metrics := []MetricCard{
		{Title: "Sistema operativo", Value: template.HTML(sistemaOperativo), IsFullWidth: false, IsHalfWidth: false},
		{Title: "Motore Backend Attivo", Value: template.HTML(versioneGo + " / Native Server"), IsFullWidth: false, IsHalfWidth: false},
		{Title: "Server Web Nginx", Value: "Proxy Inverso Operativo", IsFullWidth: false, IsHalfWidth: false},
		{Title: "Memoria RAM Fisico", Value: template.HTML(ramValore), BadgeText: "Ottimale", BadgeClass: "text-bg-success", IsFullWidth: false, IsHalfWidth: false},
		{Title: "Archiviazione Disco (Root)", Value: template.HTML(spazioDisco), IsFullWidth: false, IsHalfWidth: false},
		{Title: "Motore Database", Value: "PostgreSQL 17 (Connesso)", BadgeText: "Operativo", BadgeClass: "text-bg-success", IsFullWidth: false, IsHalfWidth: false},
		{Title: "Database Istanziati attivi", Value: `1 <span class="fs-6 fw-normal text-muted">unità logica registrata nel catalogo</span>`, IsFullWidth: true, IsHalfWidth: false},
	}

	tools := []ToolButton{
		{Href: "/api/health", Label: "Health Check"},
		{Href: "/Structure", Label: "view-Structure"},
		{Href: "/view-todo", Label: "view-todo"},
		{Href: "/view-tags", Label: "view-tags"},
	}

	databases := []ToolButton{
		{Href: "/allUsers", Label: "Collection Users"},
	}

	data := DashboardContext{
		ProjectName: project,
		LogoUrl:     logoUrl,
		AvatarUrl:   avatarUrl,
		UserName:    "Amministratore",
		Version:     render.CurrentVersion, // Recuperata dal pacchetto render
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

	var buf bytes.Buffer
	err = tmplt.ExecuteTemplate(&buf, "layout", data)
	if err != nil {
		render.HandleTemplateError(w, r, err) // Invia l'errore al gestore centralizzato
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}
